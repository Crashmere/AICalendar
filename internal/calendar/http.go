package calendar

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Crashmere/AICalendar/api"
	"github.com/Crashmere/AICalendar/web"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Store     *Store
	Origin    string
	BasePath  string
	TokenHash string
	Demo      bool
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, e error) {
	var a *APIError
	if errors.As(e, &a) {
		writeJSON(w, a.Status, map[string]any{"error": a.Message})
		return
	}
	slog.Error("request failed", "error", e)
	writeJSON(w, 500, map[string]any{"error": "服务暂时无法完成请求"})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if e := dec.Decode(v); e != nil {
		return invalid("无效的 JSON、未知字段或请求超过 1 MiB")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return invalid("请求只能包含一个 JSON 对象")
	}
	return nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := s.Store.Ping(r.Context()); e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	for _, prefix := range []string{"/api/v1", "/ingest/v1"} {
		s.archiveRoutes(mux, prefix)
		mux.HandleFunc("GET "+prefix+"/schema", func(w http.ResponseWriter, r *http.Request) {
			b, e := api.Files.ReadFile("import.schema.json")
			if e != nil {
				fail(w, e)
				return
			}
			w.Header().Set("Content-Type", "application/schema+json")
			w.Write(b)
		})
		mux.HandleFunc("POST "+prefix+"/imports/preview", s.importHandler(false))
		mux.HandleFunc("POST "+prefix+"/imports", s.importHandler(true))
		mux.HandleFunc("GET "+prefix+"/imports/{id}", func(w http.ResponseWriter, r *http.Request) {
			v, e := s.Store.ImportByID(r.Context(), r.PathValue("id"))
			if e != nil {
				fail(w, e)
				return
			}
			writeJSON(w, 200, v)
		})
		mux.HandleFunc("GET "+prefix+"/imports", func(w http.ResponseWriter, r *http.Request) {
			v, e := s.Store.Imports(r.Context())
			if e != nil {
				fail(w, e)
				return
			}
			writeJSON(w, 200, map[string]any{"items": v})
		})
		mux.HandleFunc("GET "+prefix+"/activities", s.activities)
	}
	mux.HandleFunc("GET /api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		writeJSON(w, 200, map[string]any{"name": "AICalendar", "timezone": "Asia/Shanghai", "today": time.Now().In(loc).Format(time.DateOnly), "demo": s.Demo})
	})
	mux.HandleFunc("PATCH /api/v1/activities/{id}/annotation", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Annotation      Annotation `json:"annotation"`
			ExpectedVersion *int       `json:"expected_version"`
		}
		if e := decode(w, r, &req); e != nil {
			fail(w, e)
			return
		}
		if req.ExpectedVersion == nil {
			fail(w, invalid("expected_version 必填"))
			return
		}
		v, e := s.Store.Annotate(r.Context(), r.PathValue("id"), req.Annotation, *req.ExpectedVersion)
		if e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("GET /api/v1/export", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.Store.Activities(r.Context(), Filter{IncludeHidden: true})
		if e != nil {
			fail(w, e)
			return
		}
		b, e := s.Store.Imports(r.Context())
		if e != nil {
			fail(w, e)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="aicalendar-export.json"`)
		writeJSON(w, 200, map[string]any{"schema_version": 1, "exported_at": time.Now().UTC().Format(time.RFC3339), "activities": v, "imports": b})
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]string{"error": "接口不存在"})
	})
	mux.HandleFunc("/ingest/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]string{"error": "接口不存在"})
	})
	assets, _ := fs.Sub(web.Files, "dist")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "." {
			name = "index.html"
		}
		f, e := assets.Open(name)
		if e == nil {
			info, _ := f.Stat()
			f.Close()
			if info != nil && !info.IsDir() {
				http.FileServer(http.FS(assets)).ServeHTTP(w, r)
				return
			}
		}
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		b, e := fs.ReadFile(assets, "index.html")
		if e != nil {
			fail(w, e)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		if s.BasePath != "/" && strings.HasPrefix(r.URL.Path, s.BasePath) {
			clone := r.Clone(r.Context())
			u := *r.URL
			clone.URL = &u
			clone.URL.Path = "/" + strings.TrimPrefix(r.URL.Path, s.BasePath)
			r = clone
		}
		if strings.HasPrefix(r.URL.Path, "/ingest/") {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			sum := sha256.Sum256([]byte(token))
			actual := hex.EncodeToString(sum[:])
			if s.TokenHash == "" || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || subtle.ConstantTimeCompare([]byte(actual), []byte(s.TokenHash)) != 1 {
				writeJSON(w, 401, map[string]string{"error": "导入凭据无效或尚未配置"})
				return
			}
		} else if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			origin := r.Header.Get("Origin")
			allowed := s.Origin
			if allowed == "" && s.Demo {
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				allowed = scheme + "://" + r.Host
			}
			if origin == "" || origin != allowed {
				writeJSON(w, 403, map[string]string{"error": "写入请求需来自本站页面"})
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (s *Server) importHandler(commit bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ImportRequest
		if e := decode(w, r, &req); e != nil {
			fail(w, e)
			return
		}
		v, e := s.Store.Import(r.Context(), req, commit)
		if e != nil {
			fail(w, e)
			return
		}
		status := 200
		if commit && v.Conflicts > 0 {
			status = 409
		}
		writeJSON(w, status, v)
	}
}
func (s *Server) activities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := Filter{From: q.Get("from"), To: q.Get("to"), Source: q.Get("source"), Query: q.Get("q"), Tag: q.Get("tag"), IncludeHidden: q.Get("include_hidden") == "true"}
	if (f.From != "" && !validDate(f.From)) || (f.To != "" && !validDate(f.To)) || (f.From != "" && f.To != "" && f.From > f.To) {
		fail(w, invalid("查询日期无效"))
		return
	}
	items, e := s.Store.Activities(r.Context(), f)
	if e != nil {
		fail(w, e)
		return
	}
	offset, limit := 0, 500
	for key, dst := range map[string]*int{"offset": &offset, "limit": &limit} {
		if v := q.Get(key); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil || n < 0 {
				fail(w, invalid("分页参数无效"))
				return
			}
			*dst = n
		}
	}
	if limit < 1 || limit > 5000 {
		fail(w, invalid("limit 需为 1–5000"))
		return
	}
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	writeJSON(w, 200, map[string]any{"items": items[offset:end], "total": total, "offset": offset, "limit": limit})
}
func ValidOrigin(s string) bool {
	u, e := url.Parse(s)
	return e == nil && oneOf(u.Scheme, "http", "https") && u.Host != "" && u.Path == "" && u.RawQuery == "" && u.User == nil && u.Fragment == ""
}
