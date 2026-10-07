package calendar

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Span struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
type Quality struct {
	Count string `json:"count"`
	Time  string `json:"time"`
	Topic string `json:"topic"`
}
type Provenance struct {
	Method     string `json:"method"`
	Producer   string `json:"producer,omitempty"`
	SourceRef  string `json:"source_ref,omitempty"`
	ProducedAt string `json:"produced_at,omitempty"`
}
type Record struct {
	Source           string     `json:"source"`
	ExternalID       string     `json:"external_id"`
	ConversationID   string     `json:"conversation_id,omitempty"`
	ActivityDate     string     `json:"activity_date"`
	Timezone         string     `json:"timezone"`
	FirstActivityAt  *string    `json:"first_activity_at"`
	LastActivityAt   *string    `json:"last_activity_at"`
	UserMessageCount *int       `json:"user_message_count"`
	Title            string     `json:"title"`
	Summary          string     `json:"summary"`
	Tags             []string   `json:"tags"`
	Spans            []Span     `json:"spans"`
	Quality          Quality    `json:"quality"`
	Provenance       Provenance `json:"provenance"`
	RecordState      string     `json:"record_state"`
	ExpectedVersion  *int       `json:"expected_version,omitempty"`
}
type Coverage struct {
	Source string `json:"source"`
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}
type ImportRequest struct {
	SchemaVersion  int        `json:"schema_version"`
	IdempotencyKey string     `json:"idempotency_key"`
	Mode           string     `json:"mode"`
	Records        []Record   `json:"records"`
	Coverage       []Coverage `json:"coverage,omitempty"`
}
type Annotation struct {
	Title   *string   `json:"title,omitempty"`
	Summary *string   `json:"summary,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
	Hidden  bool      `json:"hidden"`
}
type Activity struct {
	ID                string     `json:"id"`
	Version           int        `json:"version"`
	AnnotationVersion int        `json:"annotation_version"`
	Record            Record     `json:"record"`
	Annotation        Annotation `json:"annotation"`
	UpdatedAt         string     `json:"updated_at"`
}
type ImportItem struct {
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
	ID         string `json:"id"`
	Action     string `json:"action"`
	Version    int    `json:"version"`
	Reason     string `json:"reason,omitempty"`
}
type ImportResult struct {
	ID        string       `json:"id,omitempty"`
	CreatedAt string       `json:"created_at,omitempty"`
	Committed bool         `json:"committed"`
	Replay    bool         `json:"replay"`
	Inserted  int          `json:"inserted"`
	Updated   int          `json:"updated"`
	Unchanged int          `json:"unchanged"`
	Conflicts int          `json:"conflicts"`
	Items     []ImportItem `json:"items"`
	Coverage  []Coverage   `json:"coverage"`
}
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }
func invalid(s string) error      { return &APIError{400, s} }
func conflict(s string) error     { return &APIError{409, s} }

var sourcePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,79}$`)

func oneOf(value string, options ...string) bool {
	for _, o := range options {
		if value == o {
			return true
		}
	}
	return false
}
func validDate(s string) bool {
	t, e := time.Parse(time.DateOnly, s)
	return e == nil && t.Format(time.DateOnly) == s
}
func cleanTags(tags []string) ([]string, error) {
	if len(tags) > 12 {
		return nil, invalid("每条记录最多 12 个标签")
	}
	result := []string{}
	seen := map[string]bool{}
	for _, v := range tags {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if len([]rune(v)) > 40 {
			return nil, invalid("标签最长 40 字")
		}
		if !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}
	sort.Strings(result)
	return result, nil
}
func normalizeRecord(r *Record) error {
	if !sourcePattern.MatchString(r.Source) {
		return invalid("source 需为 1–80 位字母、数字、点、下划线或短横线")
	}
	if len(r.ExternalID) == 0 || len(r.ExternalID) > 300 || strings.ContainsAny(r.ExternalID, "\x00\r\n") {
		return invalid("external_id 必填，最长 300 字节")
	}
	if len(r.ConversationID) > 300 {
		return invalid("conversation_id 过长")
	}
	if !validDate(r.ActivityDate) {
		return invalid("activity_date 必须为 YYYY-MM-DD")
	}
	loc, e := time.LoadLocation(r.Timezone)
	if e != nil || r.Timezone == "" {
		return invalid("timezone 必须为有效时区")
	}
	r.Title = strings.TrimSpace(r.Title)
	r.Summary = strings.TrimSpace(r.Summary)
	if r.Title == "" || len([]rune(r.Title)) > 160 || len([]rune(r.Summary)) > 4000 {
		return invalid("标题必填且最多 160 字；摘要最多 4000 字")
	}
	if r.UserMessageCount != nil && (*r.UserMessageCount < 0 || *r.UserMessageCount > 1000000) {
		return invalid("user_message_count 必须为非负整数或 null")
	}
	if !oneOf(r.RecordState, "partial", "final") {
		return invalid("record_state 必须为 partial 或 final")
	}
	if !oneOf(r.Quality.Count, "observed", "estimated", "unknown") || !oneOf(r.Quality.Time, "observed_interval", "observed_timestamps", "estimated_interaction_span", "date_only", "unknown") || !oneOf(r.Quality.Topic, "source_title", "agent_summary", "manual") {
		return invalid("quality 值不在支持范围内")
	}
	if (r.UserMessageCount == nil) != (r.Quality.Count == "unknown") {
		return invalid("未知次数必须为 null 且 quality.count=unknown")
	}
	if len(r.Provenance.Method) == 0 || len(r.Provenance.Method) > 80 || len(r.Provenance.Producer) > 160 || len(r.Provenance.SourceRef) > 300 {
		return invalid("provenance.method 必填；来源说明过长")
	}
	if r.Provenance.ProducedAt != "" {
		t, e := time.Parse(time.RFC3339Nano, r.Provenance.ProducedAt)
		if e != nil {
			return invalid("produced_at 必须包含时区")
		}
		r.Provenance.ProducedAt = t.UTC().Format(time.RFC3339Nano)
	}
	if r.ExpectedVersion != nil && *r.ExpectedVersion < 1 {
		return invalid("expected_version 必须大于 0")
	}
	r.Tags, e = cleanTags(r.Tags)
	if e != nil {
		return e
	}
	var first, last time.Time
	if (r.FirstActivityAt == nil) != (r.LastActivityAt == nil) {
		return invalid("首尾时间需同时存在或同时为 null")
	}
	day, _ := time.ParseInLocation(time.DateOnly, r.ActivityDate, loc)
	next := day.AddDate(0, 0, 1)
	parse := func(value string) (time.Time, error) {
		t, e := time.Parse(time.RFC3339Nano, value)
		if e != nil || t.Before(day) || t.After(next) {
			return t, invalid("活动时间需包含时区，并处于所属日期内（结束可为次日零点）")
		}
		return t, nil
	}
	if r.FirstActivityAt != nil {
		first, e = parse(*r.FirstActivityAt)
		if e != nil {
			return e
		}
		last, e = parse(*r.LastActivityAt)
		if e != nil {
			return e
		}
		if first.After(last) || !first.Before(next) {
			return invalid("活动首尾时间无效")
		}
		a, b := first.UTC().Format(time.RFC3339Nano), last.UTC().Format(time.RFC3339Nano)
		r.FirstActivityAt = &a
		r.LastActivityAt = &b
	}
	if len(r.Spans) > 200 {
		return invalid("每条记录最多 200 个时段")
	}
	if r.Spans == nil {
		r.Spans = []Span{}
	}
	if len(r.Spans) > 0 && (r.FirstActivityAt == nil || oneOf(r.Quality.Time, "date_only", "unknown")) {
		return invalid("有时段时需填写首尾时间和对应的时间质量")
	}
	for i := range r.Spans {
		a, e := parse(r.Spans[i].Start)
		if e != nil {
			return e
		}
		b, e := parse(r.Spans[i].End)
		if e != nil {
			return e
		}
		if !a.Before(b) || a.Before(first) || b.After(last) {
			return invalid("时段必须为正时长且包含在首尾时间内")
		}
		r.Spans[i] = Span{a.UTC().Format(time.RFC3339Nano), b.UTC().Format(time.RFC3339Nano)}
	}
	sort.Slice(r.Spans, func(i, j int) bool { return r.Spans[i].Start < r.Spans[j].Start })
	return nil
}
func Normalize(req *ImportRequest) error {
	if req.SchemaVersion != 1 {
		return invalid("只支持 schema_version=1")
	}
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 160 {
		return invalid("idempotency_key 必填，最多 160 字节")
	}
	if !oneOf(req.Mode, "insert_only", "upsert") {
		return invalid("mode 必须为 insert_only 或 upsert")
	}
	if len(req.Records) > 500 || len(req.Coverage) > 50 || (len(req.Records) == 0 && len(req.Coverage) == 0) {
		return invalid("每批需包含记录或覆盖声明，最多 500 条记录 / 50 个覆盖声明")
	}
	seen := map[string]bool{}
	for i := range req.Records {
		r := &req.Records[i]
		if e := normalizeRecord(r); e != nil {
			return invalid(fmt.Sprintf("第 %d 条：%s", i+1, e.Error()))
		}
		key := r.Source + "\x00" + r.ExternalID
		if seen[key] {
			return invalid("同批包含重复 source/external_id")
		}
		seen[key] = true
	}
	for _, c := range req.Coverage {
		if !sourcePattern.MatchString(c.Source) || !validDate(c.From) || !validDate(c.To) || c.From > c.To || !oneOf(c.Status, "complete", "partial", "unknown") || len(c.Note) > 2000 {
			return invalid("覆盖范围声明无效")
		}
	}
	return nil
}
func digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func recordDigest(r Record) string {
	r.ExpectedVersion = nil
	r.Provenance.ProducedAt = ""
	return digest(r)
}
func recordID(r Record) string { return digest([]string{r.Source, r.ExternalID})[:32] }
