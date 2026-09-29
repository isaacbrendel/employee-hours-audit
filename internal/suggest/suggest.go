package suggest

import (
	"context"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
)

// Request is one exception a person asked about.
// A suggestion is never applied by this package.
type Request struct {
	Field  string     `json:"field"`
	Reason string     `json:"reason"`
	Raw    record.Raw `json:"raw"`
}

// Suggestion is a proposed value. The caller decides whether to use it.
type Suggestion struct {
	Field       string `json:"field"`
	Suggested   string `json:"suggested"`
	Explanation string `json:"explanation"`
}

// Client produces a suggestion. The API hides the endpoint when no client is configured.
type Client interface {
	Suggest(ctx context.Context, req Request) (Suggestion, error)
}
