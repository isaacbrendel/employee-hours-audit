// Package api is the HTTP surface of the hours audit.
// It stores nothing. The caller holds the rows.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
	"github.com/isaacbrendel/employee-hours-audit/internal/report"
	"github.com/isaacbrendel/employee-hours-audit/internal/suggest"
)

// MaxUploadBytes is the request body limit. A larger body is rejected,
// not truncated.
const MaxUploadBytes = 1 << 20

const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Server serves the audit routes and, when Static contains index.html, the built UI.
type Server struct {
	Suggest suggest.Client
	Static  fs.FS
}

// Handler returns the routes. Patterns are the Go 1.22 ServeMux form.
// The UI is mounted on exact "/" and "/assets/" so a GET of an API path
// does not get swallowed by the file server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /api/validate", s.validate)
	mux.HandleFunc("POST /api/report", s.report)
	mux.HandleFunc("POST /api/suggest", s.suggest)
	s.mountUI(mux)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"suggest": s.Suggest != nil,
	})
}

func (s *Server) validate(w http.ResponseWriter, r *http.Request) {
	res, ok := s.auditRequest(w, r, true)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	res, ok := s.auditRequest(w, r, false)
	if !ok {
		return
	}
	var body bytes.Buffer
	if err := report.Write(&body, res); err != nil {
		writeError(w, http.StatusInternalServerError, "could not build the workbook")
		return
	}
	w.Header().Set("Content-Type", xlsxType)
	w.Header().Set("Content-Disposition", `attachment; filename="hours-audit.xlsx"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
}

func (s *Server) suggest(w http.ResponseWriter, r *http.Request) {
	if s.Suggest == nil {
		writeError(w, http.StatusNotFound, "suggestions are off because ANTHROPIC_API_KEY is not set")
		return
	}
	body, err := readLimited(w, r)
	if err != nil {
		writeReadError(w, err)
		return
	}
	var req suggest.Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "request body is not valid JSON")
		return
	}
	if strings.TrimSpace(req.Field) == "" {
		writeError(w, http.StatusBadRequest, "field is required")
		return
	}
	got, err := s.Suggest.Suggest(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "suggestion service failed")
		return
	}
	if got.Field == "" {
		got.Field = req.Field
	}
	writeJSON(w, http.StatusOK, got)
}

// auditRequest reads either a CSV upload or a JSON rows array.
// csvOK is false for the report route, which accepts JSON only.
func (s *Server) auditRequest(w http.ResponseWriter, r *http.Request, csvOK bool) (record.Result, bool) {
	body, err := readLimited(w, r)
	if err != nil {
		writeReadError(w, err)
		return record.Result{}, false
	}
	media := mediaType(r.Header.Get("Content-Type"))
	switch {
	case media == "application/json":
		return s.auditJSON(w, body)
	case csvOK && strings.HasPrefix(media, "multipart/"):
		return s.auditCSV(w, r, body)
	default:
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json or multipart/form-data")
		return record.Result{}, false
	}
}

func (s *Server) auditJSON(w http.ResponseWriter, body []byte) (record.Result, bool) {
	var req struct {
		Rows []record.Input `json:"rows"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "request body is not valid JSON")
		return record.Result{}, false
	}
	if req.Rows == nil {
		writeError(w, http.StatusBadRequest, "JSON body needs a rows array")
		return record.Result{}, false
	}
	if len(req.Rows) > record.MaxDataRows {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("a request can contain at most %d rows", record.MaxDataRows))
		return record.Result{}, false
	}
	return record.AuditRows(req.Rows), true
}

func (s *Server) auditCSV(w http.ResponseWriter, r *http.Request, body []byte) (record.Result, bool) {
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err := r.ParseMultipartForm(MaxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "could not read the upload")
		return record.Result{}, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, `upload the CSV as the form field "file"`)
		return record.Result{}, false
	}
	defer file.Close()
	if name := header.Filename; name != "" && !strings.EqualFold(filepath.Ext(name), ".csv") {
		writeError(w, http.StatusUnsupportedMediaType, "upload a .csv file")
		return record.Result{}, false
	}
	if ct := mediaType(header.Header.Get("Content-Type")); ct != "" && !csvType(ct) {
		writeError(w, http.StatusUnsupportedMediaType, "upload a CSV content type")
		return record.Result{}, false
	}
	res, err := record.AuditCSV(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return record.Result{}, false
	}
	return res, true
}

func (s *Server) mountUI(mux *http.ServeMux) {
	if s.Static != nil {
		if _, err := fs.Stat(s.Static, "index.html"); err == nil {
			files := http.FileServer(http.FS(s.Static))
			mux.Handle("GET /{$}", files)
			mux.Handle("GET /assets/", files)
			return
		}
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, fallbackPage)
	})
}

func readLimited(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.ContentLength > MaxUploadBytes {
		return nil, errTooLarge
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			return nil, errTooLarge
		}
		return nil, err
	}
	return body, nil
}

var errTooLarge = errors.New("file is larger than 1 MiB")

func writeReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, errTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, errTooLarge.Error())
		return
	}
	writeError(w, http.StatusBadRequest, "could not read the request body")
}

func mediaType(header string) string {
	media, _, err := mime.ParseMediaType(header)
	if err != nil {
		return strings.TrimSpace(strings.ToLower(header))
	}
	return strings.ToLower(media)
}

func csvType(media string) bool {
	switch media {
	case "text/csv", "application/csv", "application/vnd.ms-excel", "text/plain", "application/octet-stream":
		return true
	default:
		return false
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

const fallbackPage = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Hours audit</title></head>
<body>
<p>The API is running. Build the UI with <code>npm run build</code> inside <code>web/</code>, then restart from the repository root.</p>
</body>
</html>
`
