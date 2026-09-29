package suggest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
)

func TestAnthropicSuggestParsesTextAndDoesNotChangeTheRow(t *testing.T) {
	var sawKey, sawVersion, sawPath bool
	var leaked bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawKey = r.Header.Get("x-api-key") == "test-key"
		sawVersion = r.Header.Get("anthropic-version") == "2023-06-01"
		sawPath = r.URL.Path == "/v1/messages"
		body, _ := io.ReadAll(r.Body)
		leaked = strings.Contains(string(body), "test-key")
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if payload["model"] != "claude-sonnet-5" || payload["max_tokens"] == nil {
			http.Error(w, "unexpected payload", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"content":[{"type":"text","text":"{\"suggested\":\"160\",\"explanation\":\"Use a plain number of hours.\"}"}]}`)
	}))
	defer server.Close()

	raw := record.Raw{HoursWorked: "abc", EmployeeID: "E030"}
	client := &Anthropic{Key: "test-key", Model: defaultModel, BaseURL: server.URL, HTTP: server.Client()}
	got, err := client.Suggest(context.Background(), Request{Field: "hours_worked", Reason: "not a number", Raw: raw})
	if err != nil {
		t.Fatal(err)
	}
	if leaked {
		t.Fatal("request body contained the API key")
	}
	if !sawKey || !sawVersion || !sawPath {
		t.Fatalf("key=%v version=%v path=%v", sawKey, sawVersion, sawPath)
	}
	if got.Suggested != "160" || got.Field != "hours_worked" {
		t.Fatalf("%+v", got)
	}
	if raw.HoursWorked != "abc" {
		t.Fatalf("row was changed to %q", raw.HoursWorked)
	}
}

func TestAnthropicRejectsProse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"content":[{"type":"text","text":"I think 160 is fine."}]}`)
	}))
	defer server.Close()
	client := &Anthropic{Key: "test-key", Model: defaultModel, BaseURL: server.URL, HTTP: server.Client()}
	_, err := client.Suggest(context.Background(), Request{Field: "hours_worked", Raw: record.Raw{HoursWorked: "abc"}})
	if err == nil || !strings.Contains(err.Error(), "not a single JSON value") {
		t.Fatal(err)
	}
}

func TestFromEnvWithoutKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if FromEnv() != nil {
		t.Fatal("expected suggestions to stay off")
	}
}
