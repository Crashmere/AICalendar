package calendar

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed schema.sql
var schema string

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func Open(path string, create bool) (*Store, error) {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	if create {
		if e = os.MkdirAll(filepath.Dir(absolute), 0700); e != nil {
			return nil, e
		}
		f, e := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil, e
		}
		f.Close()
	} else {
		info, e := os.Stat(absolute)
		if e != nil {
			return nil, e
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, fmt.Errorf("database is empty or not regular")
		}
	}
	u := url.URL{Scheme: "file", Path: absolute, RawQuery: url.Values{"mode": {"rw"}, "_pragma": {"foreign_keys(1)", "busy_timeout(5000)", "journal_mode(WAL)", "synchronous(FULL)"}}.Encode()}
	db, e := sql.Open("sqlite", u.String())
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if create {
		_, e = db.Exec("BEGIN;" + schema + archiveSchema + "COMMIT;")
	} else {
		var v int
		e = db.QueryRow("PRAGMA user_version").Scan(&v)
		if e == nil && v != 1 {
			e = fmt.Errorf("unsupported database version %d", v)
		}
	}
	if e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

type scanner interface{ Scan(...any) error }

func scanActivity(row scanner) (Activity, error) {
	var a Activity
	var r, n string
	e := row.Scan(&a.ID, &a.Version, &a.AnnotationVersion, &r, &n, &a.UpdatedAt)
	if e != nil {
		return a, e
	}
	if e = json.Unmarshal([]byte(r), &a.Record); e != nil {
		return a, e
	}
	e = json.Unmarshal([]byte(n), &a.Annotation)
	return a, e
}

const activityColumns = "id,version,annotation_version,record_json,annotation_json,updated_at"

func (s *Store) Import(ctx context.Context, req ImportRequest, commit bool) (ImportResult, error) {
	out := ImportResult{Items: []ImportItem{}, Coverage: req.Coverage}
	if out.Coverage == nil {
		out.Coverage = []Coverage{}
	}
	if e := Normalize(&req); e != nil {
		return out, e
	}
	requestHash := digest(req)
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	var oldHash, oldJSON string
	e = tx.QueryRowContext(ctx, "SELECT request_hash,result_json FROM import_batches WHERE idempotency_key=?", req.IdempotencyKey).Scan(&oldHash, &oldJSON)
	if e == nil {
		if oldHash != requestHash {
			return out, conflict("同一幂等键已用于不同请求")
		}
		if e = json.Unmarshal([]byte(oldJSON), &out); e != nil {
			return out, e
		}
		out.Replay = true
		return out, nil
	}
	if e != sql.ErrNoRows {
		return out, e
	}
	previous := map[string]*Activity{}
	for _, r := range req.Records {
		id := recordID(r)
		item := ImportItem{Source: r.Source, ExternalID: r.ExternalID, ID: id, Action: "insert", Version: 1}
		a, e := scanActivity(tx.QueryRowContext(ctx, "SELECT "+activityColumns+" FROM activities WHERE id=?", id))
		if e == nil {
			copy := a
			previous[id] = &copy
			item.Version = a.Version
			if recordDigest(a.Record) == recordDigest(r) {
				item.Action = "unchanged"
			} else {
				item.Action = "update"
				item.Version = a.Version + 1
				switch {
				case req.Mode != "upsert":
					item.Reason = "已有记录内容不同，请显式使用 upsert 和 expected_version"
				case r.ExpectedVersion == nil || *r.ExpectedVersion != a.Version:
					item.Reason = "版本已变化，请重新读取记录"
				case a.Record.UserMessageCount != nil && (r.UserMessageCount == nil || *r.UserMessageCount < *a.Record.UserMessageCount):
					item.Reason = "新记录减少或清空已知次数，需人工核对后使用独立更正流程"
				case a.Record.Quality.Count == "observed" && r.Quality.Count != "observed":
					item.Reason = "不能降低已确认次数的质量"
				case a.Record.RecordState == "final" && r.RecordState == "partial":
					item.Reason = "不能用部分记录覆盖完整记录"
				case a.Record.FirstActivityAt != nil && r.FirstActivityAt == nil:
					item.Reason = "不能清空已知活动时间"
				case len(a.Record.Spans) > 0 && len(r.Spans) == 0:
					item.Reason = "不能清空已知互动时段"
				case a.Record.Quality.Time == "observed_interval" && r.Quality.Time != "observed_interval":
					item.Reason = "不能降低已确认时间的质量"
				case a.Record.ConversationID != r.ConversationID || a.Record.ActivityDate != r.ActivityDate || a.Record.Timezone != r.Timezone:
					item.Reason = "不能更换记录的会话、日期或时区"
				}
				if item.Reason != "" {
					item.Action = "conflict"
					item.Version = a.Version
				}
			}
		} else if e != sql.ErrNoRows {
			return out, e
		} else if r.ExpectedVersion != nil {
			item.Action = "conflict"
			item.Reason = "预期存在的记录不存在"
			item.Version = 0
		}
		switch item.Action {
		case "insert":
			out.Inserted++
		case "update":
			out.Updated++
		case "unchanged":
			out.Unchanged++
		case "conflict":
			out.Conflicts++
		}
		out.Items = append(out.Items, item)
	}
	if !commit || out.Conflicts > 0 {
		return out, nil
	}
	out.ID = digest([]string{req.IdempotencyKey, requestHash})[:32]
	out.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	out.Committed = true
	resultJSON, _ := json.Marshal(out)
	if _, e = tx.ExecContext(ctx, "INSERT INTO import_batches VALUES(?,?,?,?,?)", out.ID, req.IdempotencyKey, requestHash, string(resultJSON), out.CreatedAt); e != nil {
		return out, e
	}
	for i, item := range out.Items {
		if item.Action == "unchanged" {
			continue
		}
		r := req.Records[i]
		r.ExpectedVersion = nil
		b, _ := json.Marshal(r)
		if item.Action == "insert" {
			_, e = tx.ExecContext(ctx, "INSERT INTO activities(id,source,external_id,activity_date,record_json,record_hash,version,updated_at) VALUES(?,?,?,?,?,?,?,?)", item.ID, r.Source, r.ExternalID, r.ActivityDate, string(b), recordDigest(r), item.Version, out.CreatedAt)
		} else {
			_, e = tx.ExecContext(ctx, "UPDATE activities SET record_json=?,record_hash=?,version=?,updated_at=? WHERE id=?", string(b), recordDigest(r), item.Version, out.CreatedAt, item.ID)
		}
		if e != nil {
			return out, e
		}
		var before any
		if a := previous[item.ID]; a != nil {
			old, _ := json.Marshal(a)
			before = string(old)
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO import_changes VALUES(?,?,?,?)", out.ID, item.ID, before, string(b)); e != nil {
			return out, e
		}
	}
	if e = tx.Commit(); e != nil {
		return ImportResult{}, e
	}
	return out, nil
}

type Filter struct {
	From, To, Source, Query, Tag string
	IncludeHidden                bool
}

func (s *Store) Activities(ctx context.Context, f Filter) ([]Activity, error) {
	query := "SELECT " + activityColumns + " FROM activities WHERE 1=1"
	args := []any{}
	if f.From != "" {
		query += " AND activity_date>=?"
		args = append(args, f.From)
	}
	if f.To != "" {
		query += " AND activity_date<=?"
		args = append(args, f.To)
	}
	if f.Source != "" {
		query += " AND source=?"
		args = append(args, f.Source)
	}
	query += " ORDER BY activity_date DESC,updated_at DESC"
	rows, e := s.db.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Activity{}
	for rows.Next() {
		a, e := scanActivity(rows)
		if e != nil {
			return nil, e
		}
		if a.Annotation.Hidden && !f.IncludeHidden {
			continue
		}
		title, summary, tags := a.Record.Title, a.Record.Summary, a.Record.Tags
		if a.Annotation.Title != nil {
			title = *a.Annotation.Title
		}
		if a.Annotation.Summary != nil {
			summary = *a.Annotation.Summary
		}
		if a.Annotation.Tags != nil {
			tags = *a.Annotation.Tags
		}
		if f.Query != "" && !strings.Contains(strings.ToLower(title+" "+summary+" "+a.Record.SourceLabel+" "+strings.Join(tags, " ")), strings.ToLower(f.Query)) {
			continue
		}
		if f.Tag != "" {
			found := a.Record.SourceLabel == f.Tag
			for _, t := range tags {
				if t == f.Tag {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Store) Annotate(ctx context.Context, id string, n Annotation, expected int) (Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, e := scanActivity(s.db.QueryRowContext(ctx, "SELECT "+activityColumns+" FROM activities WHERE id=?", id))
	if e == sql.ErrNoRows {
		return a, &APIError{404, "记录不存在"}
	}
	if e != nil {
		return a, e
	}
	if a.AnnotationVersion != expected {
		return a, conflict("记录已被修改，请刷新后重试")
	}
	if n.Title != nil {
		v := strings.TrimSpace(*n.Title)
		if v == "" || len([]rune(v)) > 160 {
			return a, invalid("标题需为 1–160 字")
		}
		n.Title = &v
	}
	if n.Summary != nil && len([]rune(*n.Summary)) > 4000 {
		return a, invalid("摘要最多 4000 字")
	}
	if n.Tags != nil {
		tags, e := cleanTags(*n.Tags)
		if e != nil {
			return a, e
		}
		n.Tags = &tags
	}
	b, _ := json.Marshal(n)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, e = s.db.ExecContext(ctx, "UPDATE activities SET annotation_json=?,annotation_version=annotation_version+1,updated_at=? WHERE id=?", string(b), now, id)
	if e != nil {
		return a, e
	}
	a.Annotation = n
	a.AnnotationVersion++
	a.UpdatedAt = now
	return a, nil
}
func (s *Store) Imports(ctx context.Context) ([]ImportResult, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT result_json FROM import_batches ORDER BY created_at DESC")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ImportResult{}
	for rows.Next() {
		var raw string
		var r ImportResult
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(raw), &r); e != nil {
			return nil, e
		}
		r.Items = nil
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) ImportByID(ctx context.Context, id string) (ImportResult, error) {
	var raw string
	var r ImportResult
	e := s.db.QueryRowContext(ctx, "SELECT result_json FROM import_batches WHERE id=?", id).Scan(&raw)
	if e == sql.ErrNoRows {
		return r, &APIError{404, "导入批次不存在"}
	}
	if e != nil {
		return r, e
	}
	e = json.Unmarshal([]byte(raw), &r)
	return r, e
}
func CheckFile(ctx context.Context, path string) error {
	p, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	u := url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro"}
	db, e := sql.Open("sqlite", u.String())
	if e != nil {
		return e
	}
	defer db.Close()
	var v int
	if e = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v != 1 {
		return fmt.Errorf("unsupported schema: %d", v)
	}
	var check string
	if e = db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check); e != nil {
		return e
	}
	if check != "ok" {
		return fmt.Errorf("integrity check: %s", check)
	}
	for _, t := range []string{"activities", "import_batches", "import_changes"} {
		var n int
		if e = db.QueryRowContext(ctx, "SELECT count(*) FROM "+t).Scan(&n); e != nil {
			return e
		}
	}
	var archiveTables int
	if e = db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='archive_schema'").Scan(&archiveTables); e != nil {
		return e
	}
	if archiveTables > 0 {
		var version int
		if e = db.QueryRowContext(ctx, "SELECT version FROM archive_schema").Scan(&version); e != nil || version != 1 {
			return fmt.Errorf("invalid archive schema: version %d, %v", version, e)
		}
		for _, table := range []string{"conversation_archives", "archive_parts", "archive_messages"} {
			var count int
			if e = db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); e != nil {
				return e
			}
		}
	}
	rows, e := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if e != nil {
		return e
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("foreign key check failed")
	}
	return rows.Err()
}
func newFile(path string) (string, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return "", e
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	return p, f.Close()
}

// Backup snapshots records, annotations and import audit. Conversation archives
// are deliberately excluded; the snapshot keeps their empty tables so a restored
// database still serves the archive API.
func (s *Store) Backup(ctx context.Context, path string) (err error) {
	p, err := newFile(path)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(p)
		}
	}()
	target, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: p, RawQuery: "mode=rw"}).String())
	if err != nil {
		return err
	}
	_, err = target.ExecContext(ctx, "BEGIN;"+schema+archiveSchema+"COMMIT;")
	if e := target.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "ATTACH DATABASE ? AS snapshot", p); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "DETACH DATABASE snapshot")
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"activities", "import_batches", "import_changes"} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO snapshot."+table+" SELECT * FROM main."+table); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "DETACH DATABASE snapshot"); err != nil {
		return err
	}
	return CheckFile(ctx, p)
}
func Restore(ctx context.Context, source, destination string) error {
	if e := CheckFile(ctx, source); e != nil {
		return e
	}
	p, e := filepath.Abs(source)
	if e != nil {
		return e
	}
	u := url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro"}
	db, e := sql.Open("sqlite", u.String())
	if e != nil {
		return e
	}
	defer db.Close()
	p, err := newFile(destination)
	if err != nil {
		return err
	}
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", p); err == nil {
		err = CheckFile(ctx, p)
	}
	if err != nil {
		os.Remove(p)
	}
	return err
}
