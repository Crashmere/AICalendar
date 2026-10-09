package calendar

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Archive storage is an additive extension: the original schema and old binary
// remain readable, including the entire database through the native backup command.
//
//go:embed archives.sql
var archiveSchema string

const archivePartSize = 384 * 1024
const archiveMaxBytes = 128 * 1024 * 1024
const archiveMaxExpanded = 512 * 1024 * 1024

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type ArchiveManifest struct {
	SourceLabel    string `json:"source_label,omitempty"`
	SchemaVersion  int    `json:"schema_version"`
	Source         string `json:"source"`
	ConversationID string `json:"conversation_id"`
	Title          string `json:"title"`
	CapturedAt     string `json:"captured_at"`
	Coverage       string `json:"coverage"`
	Note           string `json:"note"`
	SHA256         string `json:"sha256"`
	Bytes          int64  `json:"bytes"`
	Parts          int    `json:"parts"`
	ExpandedSHA256 string `json:"expanded_sha256"`
	ExpandedBytes  int64  `json:"expanded_bytes"`
	MessageCount   int    `json:"message_count"`
	SourceCount    int    `json:"source_count"`
}
type Archive struct {
	ID            string          `json:"id"`
	Manifest      ArchiveManifest `json:"manifest"`
	State         string          `json:"state"`
	CreatedAt     string          `json:"created_at"`
	CommittedAt   *string         `json:"committed_at"`
	ReceivedParts []int           `json:"received_parts,omitempty"`
}
type ArchiveSource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
	Parts     int    `json:"parts"`
}
type ArchiveHeader struct {
	Type           string          `json:"type"`
	Format         string          `json:"format"`
	Source         string          `json:"source"`
	ConversationID string          `json:"conversation_id"`
	Title          string          `json:"title"`
	Sources        []ArchiveSource `json:"sources"`
}
type ArchiveMessage struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	At         *string         `json:"at"`
	Content    string          `json:"content"`
	Part       int             `json:"part"`
	Parts      int             `json:"parts"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

func (s *Store) ArchivesEnabled(ctx context.Context) bool {
	var n int
	return s.db.QueryRowContext(ctx, "SELECT version FROM archive_schema").Scan(&n) == nil && n == 1
}
func (s *Store) EnableArchives(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, archiveSchema); e != nil {
		return e
	}
	var v int
	if e = tx.QueryRowContext(ctx, "SELECT version FROM archive_schema").Scan(&v); e != nil {
		return e
	}
	if v != 1 {
		return fmt.Errorf("unsupported archive schema %d", v)
	}
	return tx.Commit()
}
func validManifest(m *ArchiveManifest) error {
	m.SourceLabel = strings.TrimSpace(m.SourceLabel)
	if len([]rune(m.SourceLabel)) > 80 {
		return invalid("来源标签最长 80 字")
	}
	if m.SchemaVersion != 1 || !sourcePattern.MatchString(m.Source) || len(m.ConversationID) == 0 || len(m.ConversationID) > 300 {
		return invalid("存档版本、来源或会话标识无效")
	}
	if len([]rune(m.Title)) > 160 || len([]rune(m.Note)) > 4000 || !oneOf(m.Coverage, "complete", "partial", "unknown") {
		return invalid("存档标题、覆盖说明无效")
	}
	t, e := time.Parse(time.RFC3339Nano, m.CapturedAt)
	if e != nil {
		return invalid("captured_at 需包含时区")
	}
	m.CapturedAt = t.UTC().Format(time.RFC3339Nano)
	if !shaPattern.MatchString(m.SHA256) || !shaPattern.MatchString(m.ExpandedSHA256) || m.Bytes < 1 || m.Bytes > archiveMaxBytes || m.ExpandedBytes < 1 || m.ExpandedBytes > archiveMaxExpanded {
		return invalid("存档哈希或大小无效：压缩后最多 128 MiB，展开后最多 512 MiB")
	}
	if m.Parts != int((m.Bytes+archivePartSize-1)/archivePartSize) || m.MessageCount < 0 || m.MessageCount > 200000 || m.SourceCount < 1 || m.SourceCount > 100 {
		return invalid("存档分块数或消息/来源数量无效")
	}
	return nil
}
func archiveManifestHash(m ArchiveManifest) string {
	m.CapturedAt = ""
	m.SourceLabel = ""
	return digest(m)
}
func scanArchive(row scanner) (Archive, error) {
	var a Archive
	var raw string
	e := row.Scan(&a.ID, &raw, &a.State, &a.CreatedAt, &a.CommittedAt)
	if e == nil {
		e = json.Unmarshal([]byte(raw), &a.Manifest)
	}
	return a, e
}

const archiveColumns = "id,manifest_json,state,created_at,committed_at"

func (s *Store) Archive(ctx context.Context, id string) (Archive, error) {
	a, e := scanArchive(s.db.QueryRowContext(ctx, "SELECT "+archiveColumns+" FROM conversation_archives WHERE id=?", id))
	if e == sql.ErrNoRows {
		return a, &APIError{404, "存档不存在"}
	}
	if e != nil {
		return a, e
	}
	if a.State != "committed" {
		rows, e := s.db.QueryContext(ctx, "SELECT part_index FROM archive_parts WHERE archive_id=? ORDER BY part_index", id)
		if e != nil {
			return a, e
		}
		defer rows.Close()
		a.ReceivedParts = []int{}
		for rows.Next() {
			var i int
			if e = rows.Scan(&i); e != nil {
				return a, e
			}
			a.ReceivedParts = append(a.ReceivedParts, i)
		}
		e = rows.Err()
	}
	return a, e
}
func (s *Store) PrepareArchive(ctx context.Context, m ArchiveManifest) (Archive, error) {
	if e := validManifest(&m); e != nil {
		return Archive{}, e
	}
	id := digest([]string{m.Source, m.ConversationID, m.SHA256})
	s.mu.Lock()
	defer s.mu.Unlock()
	old, e := s.Archive(ctx, id)
	if e == nil {
		if archiveManifestHash(old.Manifest) != archiveManifestHash(m) {
			return old, conflict("相同内容的存档元数据不同")
		}
		if m.SourceLabel != "" && old.Manifest.SourceLabel != m.SourceLabel {
			old.Manifest.SourceLabel = m.SourceLabel
			raw, _ := json.Marshal(old.Manifest)
			if _, e = s.db.ExecContext(ctx, "UPDATE conversation_archives SET manifest_json=? WHERE id=?", string(raw), id); e != nil {
				return old, e
			}
		}
		return old, nil
	}
	if ae, ok := e.(*APIError); !ok || ae.Status != 404 {
		return Archive{}, e
	}
	raw, _ := json.Marshal(m)
	_, e = s.db.ExecContext(ctx, "INSERT INTO conversation_archives(id,source,conversation_id,manifest_json,manifest_hash,state,created_at) VALUES(?,?,?,?,?,'uploading',?)", id, m.Source, m.ConversationID, string(raw), archiveManifestHash(m), time.Now().UTC().Format(time.RFC3339Nano))
	if e != nil {
		return Archive{}, e
	}
	return s.Archive(ctx, id)
}
func (s *Store) PutArchivePart(ctx context.Context, id string, index int, data []byte, expected string) error {
	if len(data) < 1 || len(data) > archivePartSize || !shaPattern.MatchString(expected) {
		return invalid("存档分块无效")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != expected {
		return invalid("分块校验和不符")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, e := s.Archive(ctx, id)
	if e != nil {
		return e
	}
	m := a.Manifest
	if index < 0 || index >= m.Parts {
		return invalid("分块编号越界")
	}
	size := int64(archivePartSize)
	if index == m.Parts-1 {
		size = m.Bytes - int64(index)*archivePartSize
	}
	if int64(len(data)) != size {
		return invalid("分块长度不符")
	}
	var old string
	e = s.db.QueryRowContext(ctx, "SELECT sha256 FROM archive_parts WHERE archive_id=? AND part_index=?", id, index).Scan(&old)
	if e == nil {
		if old != expected {
			return conflict("已上传分块内容不同")
		}
		return nil
	}
	if e != sql.ErrNoRows {
		return e
	}
	if a.State == "committed" {
		return conflict("完整存档不可修改")
	}
	_, e = s.db.ExecContext(ctx, "INSERT INTO archive_parts VALUES(?,?,?,?)", id, index, expected, data)
	return e
}

type archiveReader struct {
	ctx   context.Context
	query interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	id          string
	parts, next int
	current     *bytes.Reader
}

func (r *archiveReader) Read(p []byte) (int, error) {
	for {
		if r.current != nil && r.current.Len() > 0 {
			return r.current.Read(p)
		}
		if r.next >= r.parts {
			return 0, io.EOF
		}
		var data []byte
		if e := r.query.QueryRowContext(r.ctx, "SELECT data FROM archive_parts WHERE archive_id=? AND part_index=?", r.id, r.next).Scan(&data); e != nil {
			return 0, e
		}
		r.next++
		r.current = bytes.NewReader(data)
	}
}

type sourceProgress struct {
	spec  ArchiveSource
	next  int
	bytes int64
	hash  hash.Hash
}

func strictJSON(raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return invalid("存档行不符合统一格式")
	}
	if d.Decode(new(any)) != io.EOF {
		return invalid("存档行包含多段 JSON")
	}
	return nil
}
func gzipData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, e := w.Write(data); e != nil {
		return nil, e
	}
	if e := w.Close(); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}
func (s *Store) CommitArchive(ctx context.Context, id string) (Archive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, e := s.Archive(ctx, id)
	if e != nil || a.State == "committed" {
		return a, e
	}
	m := a.Manifest
	if len(a.ReceivedParts) != m.Parts {
		return a, conflict("存档分块尚未上传完整")
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return a, e
	}
	defer tx.Rollback()
	encodedHash := sha256.New()
	expandedHash := sha256.New()
	input := io.TeeReader(&archiveReader{ctx: ctx, query: tx, id: id, parts: m.Parts}, encodedHash)
	gz, e := gzip.NewReader(input)
	if e != nil {
		return a, invalid("存档不是有效 gzip")
	}
	defer gz.Close()
	reader := io.LimitReader(io.TeeReader(gz, expandedHash), m.ExpandedBytes+1)
	scan := bufio.NewScanner(reader)
	scan.Buffer(make([]byte, 65536), 1048576)
	var header ArchiveHeader
	var expanded int64
	var ordinal, messages int
	sources := map[string]*sourceProgress{}
	seen := map[string]bool{}
	var previous ArchiveMessage
	for scan.Scan() {
		line := append([]byte(nil), scan.Bytes()...)
		expanded += int64(len(line) + 1)
		if expanded > m.ExpandedBytes || !utf8.Valid(line) {
			return a, invalid("展开大小不符或不是 UTF-8")
		}
		if ordinal == 0 {
			if e = strictJSON(line, &header); e != nil {
				return a, e
			}
			if header.Type != "header" || header.Format != "aicalendar.conversation.v1" || header.Source != m.Source || header.ConversationID != m.ConversationID || header.Title != m.Title || len(header.Sources) != m.SourceCount {
				return a, invalid("存档头与清单不符")
			}
			for _, spec := range header.Sources {
				if spec.ID == "" || len(spec.ID) > 300 || sources[spec.ID] != nil || !shaPattern.MatchString(spec.SHA256) || spec.Bytes < 0 || spec.Bytes > archiveMaxExpanded || spec.Parts < 1 {
					return a, invalid("原始资料清单无效")
				}
				sources[spec.ID] = &sourceProgress{spec: spec, hash: sha256.New()}
			}
			ordinal++
			continue
		}
		var kind struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(line, &kind) != nil {
			return a, invalid("存档行 JSON 无效")
		}
		switch kind.Type {
		case "message":
			var message ArchiveMessage
			if e = strictJSON(line, &message); e != nil {
				return a, e
			}
			if message.ID == "" || len(message.ID) > 300 || !oneOf(message.Role, "user", "assistant", "system", "developer", "tool", "unknown") || len(message.Content) > 65536 || len(message.Attributes) > 16384 || message.Parts < 1 || message.Part < 0 || message.Part >= message.Parts {
				return a, invalid("消息字段无效或超长；需分段，不能截断")
			}
			if message.At != nil {
				if _, e = time.Parse(time.RFC3339Nano, *message.At); e != nil {
					return a, invalid("消息时间无效")
				}
			}
			if message.Part == 0 {
				if previous.Parts > 0 && previous.Part+1 != previous.Parts {
					return a, invalid("上一条消息分段不完整")
				}
				if seen[message.ID] {
					return a, invalid("消息 ID 重复")
				}
				seen[message.ID] = true
				messages++
			} else if message.ID != previous.ID || message.Part != previous.Part+1 || message.Parts != previous.Parts || message.Role != previous.Role || !bytes.Equal(message.Attributes, previous.Attributes) || !sameTimestamp(message.At, previous.At) {
				return a, invalid("消息分段顺序或元数据不符")
			}
			if messages > m.MessageCount {
				return a, invalid("消息数量超出清单")
			}
			previous = message
			packed, e := gzipData(line)
			if e != nil {
				return a, e
			}
			if _, e = tx.ExecContext(ctx, "INSERT INTO archive_messages VALUES(?,?,?)", id, ordinal-1, packed); e != nil {
				return a, e
			}
		case "source_chunk":
			var part struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Part int    `json:"part"`
				Data []byte `json:"data"`
			}
			if e = strictJSON(line, &part); e != nil {
				return a, e
			}
			p := sources[part.ID]
			if p == nil || part.Part != p.next || p.next >= p.spec.Parts || len(part.Data) > archivePartSize {
				return a, invalid("原始资料分块顺序无效")
			}
			p.next++
			p.bytes += int64(len(part.Data))
			if p.bytes > p.spec.Bytes {
				return a, invalid("原始资料大小超出清单")
			}
			p.hash.Write(part.Data)
		default:
			return a, invalid("未知存档行类型")
		}
		ordinal++
	}
	if e = scan.Err(); e != nil {
		return a, invalid("存档损坏或单行超过 1 MiB")
	}
	if previous.Parts > 0 && previous.Part+1 != previous.Parts {
		return a, invalid("最后一条消息分段不完整")
	}
	if ordinal == 0 || expanded != m.ExpandedBytes || hex.EncodeToString(expandedHash.Sum(nil)) != m.ExpandedSHA256 || hex.EncodeToString(encodedHash.Sum(nil)) != m.SHA256 || messages != m.MessageCount {
		return a, invalid("存档长度、消息数量或整体校验和不符")
	}
	for _, p := range sources {
		if p.next != p.spec.Parts || p.bytes != p.spec.Bytes || hex.EncodeToString(p.hash.Sum(nil)) != p.spec.SHA256 {
			return a, invalid("原始资料未完整保存或校验和不符")
		}
	}
	if _, e = tx.ExecContext(ctx, "UPDATE conversation_archives SET state='committed',committed_at=? WHERE id=?", time.Now().UTC().Format(time.RFC3339Nano), id); e != nil {
		return a, e
	}
	if e = tx.Commit(); e != nil {
		return a, e
	}
	return s.Archive(ctx, id)
}
func sameTimestamp(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func (s *Store) Archives(ctx context.Context, source, conversation, label, query string, offset, limit int) ([]Archive, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT "+archiveColumns+" FROM conversation_archives WHERE state='committed' AND (?='' OR source=?) AND (?='' OR conversation_id=?) AND (?='' OR COALESCE(json_extract(manifest_json,'$.source_label'),'') LIKE '%'||?||'%') AND (?='' OR (COALESCE(json_extract(manifest_json,'$.title'),'')||' '||COALESCE(json_extract(manifest_json,'$.source_label'),'')) LIKE '%'||?||'%') ORDER BY committed_at DESC,id LIMIT ? OFFSET ?", source, source, conversation, conversation, label, label, query, query, limit, offset)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Archive{}
	for rows.Next() {
		a, e := scanArchive(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
// archiveMessagesBudget bounds one response; at least one fragment is always returned.
const archiveMessagesBudget = 2 * 1024 * 1024

func (s *Store) ArchiveMessages(ctx context.Context, id string, after, limit int) ([]json.RawMessage, int, bool, error) {
	a, e := s.Archive(ctx, id)
	if e != nil {
		return nil, after, false, e
	}
	if a.State != "committed" {
		return nil, after, false, conflict("存档尚未完成")
	}
	rows, e := s.db.QueryContext(ctx, "SELECT ordinal,message_json FROM archive_messages WHERE archive_id=? AND ordinal>? ORDER BY ordinal LIMIT ?", id, after, limit+1)
	if e != nil {
		return nil, after, false, e
	}
	defer rows.Close()
	out := []json.RawMessage{}
	next, size := after, 0
	for rows.Next() {
		if len(out) == limit || (len(out) > 0 && size >= archiveMessagesBudget) {
			return out, next, true, nil
		}
		var raw []byte
		if e = rows.Scan(&next, &raw); e != nil {
			return nil, next, false, e
		}
		gz, e := gzip.NewReader(bytes.NewReader(raw))
		if e != nil {
			return nil, next, false, e
		}
		data, e := io.ReadAll(io.LimitReader(gz, 1048577))
		gz.Close()
		if e != nil {
			return nil, next, false, e
		}
		size += len(data)
		out = append(out, json.RawMessage(data))
	}
	return out, next, false, rows.Err()
}
func (s *Store) WriteArchive(ctx context.Context, a Archive, w io.Writer) error {
	if a.State != "committed" {
		return conflict("存档尚未完成")
	}
	_, e := io.Copy(w, &archiveReader{ctx: ctx, query: s.db, id: a.ID, parts: a.Manifest.Parts})
	return e
}

// MigrateArchives only adds archive tables. It never rewrites existing records.
func MigrateArchives(ctx context.Context, path string) error {
	if _, e := os.Stat(path); e != nil {
		return e
	}
	s, e := Open(path, false)
	if e != nil {
		return e
	}
	defer s.Close()
	return s.EnableArchives(ctx)
}
