package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
	"github.com/isaacbrendel/employee-hours-audit/internal/suggest"
	"github.com/xuri/excelize/v2"
)

func TestHealthAndFallback(t *testing.T) {
	srv := (&Server{}).Handler()
	res := get(t, srv, "/healthz")
	if res.Code != 200 || !strings.Contains(res.Body.String(), `"suggest":false`) {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
	page := get(t, srv, "/")
	if page.Code != 200 || !strings.Contains(page.Body.String(), "The API is running") {
		t.Fatalf("%d %s", page.Code, page.Body.String())
	}
	missing := get(t, srv, "/missing")
	if missing.Code != 404 {
		t.Fatalf("status %d", missing.Code)
	}
}

func TestStaticUI(t *testing.T) {
	srv := (&Server{Static: fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<h1>audit</h1>")},
	}}).Handler()
	res := get(t, srv, "/")
	if res.Code != 200 || !strings.Contains(res.Body.String(), "audit") {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
}

func TestValidateJSON(t *testing.T) {
	body := `{"rows":[
		{"row":2,"employee_id":"E001","name":"Ada","hire_date":"2020-01-15","month":"2024-01","hours_worked":"160","coverage_offered":"yes"},
		{"row":3,"employee_id":"E002","name":"Grace","hire_date":"nope","month":"2024-01","hours_worked":"10","coverage_offered":"yes"}
	]}`
	res := postJSON(t, (&Server{}).Handler(), "/api/validate", body)
	if res.Code != 200 {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
	var got record.Result
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Clean) != 1 || len(got.Exceptions) != 1 || !got.Clean[0].FullTime {
		t.Fatalf("%+v", got)
	}
}

func TestValidateRejectsBadJSONAndMissingRows(t *testing.T) {
	srv := (&Server{}).Handler()
	if res := postJSON(t, srv, "/api/validate", "{"); res.Code != 400 {
		t.Fatalf("bad json %d %s", res.Code, res.Body.String())
	}
	if res := postJSON(t, srv, "/api/validate", `{"clean":[]}`); res.Code != 400 || !strings.Contains(res.Body.String(), "rows") {
		t.Fatalf("missing rows %d %s", res.Code, res.Body.String())
	}
	if res := postJSON(t, srv, "/api/report", "not-json"); res.Code != 400 {
		t.Fatalf("report json %d", res.Code)
	}
}

func TestValidateCSVAndRejections(t *testing.T) {
	srv := (&Server{}).Handler()
	csvBody := "employee_id,name,hire_date,month,hours_worked,coverage_offered\nE001,Ada,2020-01-15,2024-01,10,yes\n"
	res := postFile(t, srv, "hours.csv", "text/csv", csvBody)
	if res.Code != 200 || !strings.Contains(res.Body.String(), `"clean_rows":1`) {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}

	badName := postFile(t, srv, "hours.xlsx", "text/csv", csvBody)
	if badName.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("extension %d %s", badName.Code, badName.Body.String())
	}
	badType := postFile(t, srv, "hours.csv", "image/png", csvBody)
	if badType.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("type %d %s", badType.Code, badType.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/validate", strings.NewReader("hi"))
	req.Header.Set("Content-Type", "text/plain")
	plain := httptest.NewRecorder()
	srv.ServeHTTP(plain, req)
	if plain.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("plain %d", plain.Code)
	}
}

func TestOversizedUpload(t *testing.T) {
	srv := (&Server{}).Handler()
	payload := bytes.Repeat([]byte("a"), MaxUploadBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/api/validate", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("content-length path %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/validate", io.NopCloser(bytes.NewReader(payload)))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("maxbytes path %d %s", res.Code, res.Body.String())
	}
}

func TestReportWorkbook(t *testing.T) {
	body := `{"rows":[{"row":2,"employee_id":"E009","name":"=1+1","hire_date":"2020-01-15","month":"2024-01","hours_worked":"20","coverage_offered":"no"}]}`
	res := postJSON(t, (&Server{}).Handler(), "/api/report", body)
	if res.Code != 200 || !strings.Contains(res.Header().Get("Content-Type"), "spreadsheetml") {
		t.Fatalf("%d %s %s", res.Code, res.Header().Get("Content-Type"), res.Body.String())
	}
	f, err := excelize.OpenReader(bytes.NewReader(res.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	name, err := f.GetCellValue("Clean Data", "B2")
	if err != nil || name != "'=1+1" {
		t.Fatalf("name %q %v", name, err)
	}
	formula, err := f.GetCellFormula("Summary", "B7")
	if err != nil || !strings.Contains(formula, "COUNTIF") {
		t.Fatalf("formula %q %v", formula, err)
	}
	coverage, err := f.GetCellValue("Clean Data", "F2")
	if err != nil || coverage != "FALSE" {
		t.Fatalf("coverage %q", coverage)
	}
}

func TestSuggestIsHiddenUntilConfigured(t *testing.T) {
	res := postJSON(t, (&Server{}).Handler(), "/api/suggest", `{"field":"hours_worked"}`)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "ANTHROPIC_API_KEY") {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
}

type fakeSuggest struct{}

func (fakeSuggest) Suggest(_ context.Context, req suggest.Request) (suggest.Suggestion, error) {
	return suggest.Suggestion{
		Field:       req.Field,
		Suggested:   "160",
		Explanation: "The cell is not a number.",
	}, nil
}

func TestSuggestDoesNotApplyTheValue(t *testing.T) {
	srv := (&Server{Suggest: fakeSuggest{}}).Handler()
	res := postJSON(t, srv, "/api/suggest", `{"field":"hours_worked","reason":"not a number","raw":{"hours_worked":"abc"}}`)
	if res.Code != 200 {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
	var got suggest.Suggestion
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Suggested != "160" || got.Field != "hours_worked" {
		t.Fatalf("%+v", got)
	}
	health := get(t, srv, "/healthz")
	if !strings.Contains(health.Body.String(), `"suggest":true`) {
		t.Fatalf("%s", health.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/validate", nil)
	res := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", res.Code)
	}
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func postJSON(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func postFile(t *testing.T, h http.Handler, name, contentType, csv string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="` + name + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(csv)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/validate", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}
