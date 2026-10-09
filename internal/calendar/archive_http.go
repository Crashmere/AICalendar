package calendar

import (
	"context"
	"github.com/Crashmere/AICalendar/api"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) archiveRoutes(mux *http.ServeMux, prefix string) {
	ready := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !s.Store.ArchivesEnabled(r.Context()) {
				writeJSON(w, 503, map[string]string{"error": "完整聊天存档尚未启用，请先完成数据库扩展"})
				return
			}
			handler(w, r)
		}
	}
	mux.HandleFunc("GET "+prefix+"/archives/schema", ready(func(w http.ResponseWriter, r *http.Request) {
		b, e := api.Files.ReadFile("archive.schema.json")
		if e != nil {
			fail(w, e)
			return
		}
		w.Header().Set("Content-Type", "application/schema+json")
		w.Write(b)
	}))
	mux.HandleFunc("POST "+prefix+"/archives/prepare", ready(func(w http.ResponseWriter, r *http.Request) {
		var m ArchiveManifest
		if e := decode(w, r, &m); e != nil {
			fail(w, e)
			return
		}
		a, e := s.Store.PrepareArchive(r.Context(), m)
		if e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, a)
	}))
	mux.HandleFunc("GET "+prefix+"/archives", ready(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if offset < 0 {
			offset = 0
		}
		if limit < 1 || limit > 100 {
			limit = 50
		}
		items, e := s.Store.Archives(r.Context(), q.Get("source"), q.Get("conversation_id"), q.Get("label"), q.Get("q"), offset, limit)
		if e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "next_offset": offset + len(items)})
	}))
	mux.HandleFunc("GET "+prefix+"/archives/{id}", ready(func(w http.ResponseWriter, r *http.Request) {
		a, e := s.Store.Archive(r.Context(), r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, a)
	}))
	mux.HandleFunc("POST "+prefix+"/archives/{id}/parts", ready(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Index  int    `json:"index"`
			Data   []byte `json:"data"`
			SHA256 string `json:"sha256"`
		}
		if e := decode(w, r, &body); e != nil {
			fail(w, e)
			return
		}
		if e := s.Store.PutArchivePart(r.Context(), r.PathValue("id"), body.Index, body.Data, body.SHA256); e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]bool{"stored": true})
	}))
	mux.HandleFunc("POST "+prefix+"/archives/{id}/commit", ready(func(w http.ResponseWriter, r *http.Request) {
		// An accepted, idempotent verification can finish after a network disconnect.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		a, e := s.Store.CommitArchive(ctx, r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, a)
	}))
	mux.HandleFunc("GET "+prefix+"/archives/{id}/messages", ready(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		after := -1
		if q.Has("after") {
			n, e := strconv.Atoi(q.Get("after"))
			if e != nil || n < -1 {
				fail(w, invalid("after 无效"))
				return
			}
			after = n
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit < 1 || limit > 200 {
			limit = 20
		}
		items, next, more, e := s.Store.ArchiveMessages(r.Context(), r.PathValue("id"), after, limit)
		if e != nil {
			fail(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "next_after": next, "has_more": more})
	}))
	mux.HandleFunc("GET "+prefix+"/archives/{id}/download", ready(func(w http.ResponseWriter, r *http.Request) {
		a, e := s.Store.Archive(r.Context(), r.PathValue("id"))
		if e != nil {
			fail(w, e)
			return
		}
		if a.State != "committed" {
			fail(w, conflict("存档尚未完成"))
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="conversation-`+a.ID[:16]+`.jsonl.gz"`)
		w.Header().Set("Content-Length", strconv.FormatInt(a.Manifest.Bytes, 10))
		if r.Method == "HEAD" {
			return
		}
		if e = s.Store.WriteArchive(r.Context(), a, archiveDownloadWriter{w}); e != nil {
			return
		}
	}))
}

type archiveDownloadWriter struct{ http.ResponseWriter }

func (w archiveDownloadWriter) Write(p []byte) (int, error) {
	_ = http.NewResponseController(w.ResponseWriter).SetWriteDeadline(time.Now().Add(30 * time.Second))
	return w.ResponseWriter.Write(p)
}
