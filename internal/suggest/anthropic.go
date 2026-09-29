package suggest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// defaultModel is listed on the Messages API model enum.
	// https://docs.anthropic.com/en/api/messages
	// The same page's curl example uses claude-opus-5. This default is the
	// smaller model id from that enum. ANTHROPIC_MODEL overrides it.
	defaultModel   = "claude-sonnet-5"
	apiVersion     = "2023-06-01"
	defaultBaseURL = "https://api.anthropic.com"
)

// Anthropic calls the Messages API. It returns a suggestion and does not
// change the row it was shown.
type Anthropic struct {
	Key     string
	Model   string
	BaseURL string
	HTTP    *http.Client
}

// FromEnv returns nil when ANTHROPIC_API_KEY is unset. The API then hides suggestions.
func FromEnv() Client {
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if key == "" {
		return nil
	}
	model := strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL"))
	if model == "" {
		model = defaultModel
	}
	return &Anthropic{
		Key:     key,
		Model:   model,
		BaseURL: defaultBaseURL,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (a *Anthropic) Suggest(ctx context.Context, req Request) (Suggestion, error) {
	if strings.TrimSpace(a.Key) == "" {
		return Suggestion{}, fmt.Errorf("suggestions are off because ANTHROPIC_API_KEY is not set")
	}
	payload, err := json.Marshal(messageBody{
		Model:     a.Model,
		MaxTokens: 300,
		Messages: []message{{
			Role:    "user",
			Content: prompt(req),
		}},
	})
	if err != nil {
		return Suggestion{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.BaseURL, "/")+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return Suggestion{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.Key)
	httpReq.Header.Set("anthropic-version", apiVersion)

	client := a.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(httpReq)
	if err != nil {
		return Suggestion{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Suggestion{}, err
	}
	if res.StatusCode != http.StatusOK {
		return Suggestion{}, fmt.Errorf("suggestion service returned %d", res.StatusCode)
	}
	var parsed messagesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Suggestion{}, fmt.Errorf("suggestion service returned unreadable JSON")
	}
	var text string
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	suggested, err := parseSuggestion(text)
	if err != nil {
		return Suggestion{}, err
	}
	return Suggestion{
		Field:       req.Field,
		Suggested:   suggested.Suggested,
		Explanation: suggested.Explanation,
	}, nil
}

type messageBody struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type modelJSON struct {
	Suggested   string `json:"suggested"`
	Explanation string `json:"explanation"`
}

func prompt(req Request) string {
	raw, _ := json.Marshal(req.Raw)
	return fmt.Sprintf(`Suggest one replacement for a single field in an employee hours row. Do not change the other fields.
Field: %s
Current value: %s
Why it was held: %s
Row JSON: %s

Reply with JSON only, no markdown, in this shape:
{"suggested":"one replacement value","explanation":"one sentence"}
If the row does not contain enough to choose a value, use an empty suggested string.`,
		req.Field, rawValue(req), req.Reason, raw)
}

func rawValue(req Request) string {
	switch req.Field {
	case "employee_id":
		return req.Raw.EmployeeID
	case "name":
		return req.Raw.Name
	case "hire_date":
		return req.Raw.HireDate
	case "month":
		return req.Raw.Month
	case "hours_worked":
		return req.Raw.HoursWorked
	case "coverage_offered":
		return req.Raw.CoverageOffered
	default:
		return ""
	}
}

func parseSuggestion(text string) (modelJSON, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	var got modelJSON
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		return modelJSON{}, fmt.Errorf("suggestion was not a single JSON value")
	}
	got.Suggested = strings.TrimSpace(got.Suggested)
	got.Explanation = strings.TrimSpace(got.Explanation)
	if got.Suggested == "" {
		return modelJSON{}, fmt.Errorf("no suggestion was returned")
	}
	if len(got.Suggested) > 200 {
		return modelJSON{}, fmt.Errorf("suggestion was too long to apply")
	}
	return got, nil
}
