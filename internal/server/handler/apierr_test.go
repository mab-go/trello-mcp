package handler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mab-go/logging"
	"github.com/mab-go/trello-mcp/internal/trello"
)

func TestMapAPIErrorLogging(t *testing.T) {
	ctx, rec := testCtxRecording()
	log, _ := logging.FromContext(ctx)

	apiErr := &trello.APIError{StatusCode: 404, Body: "not found"}
	_, _ = mapAPIError(log, apiErr)

	if !rec.HasEntry("WARN", eventAPIError) {
		t.Fatal("expected WARN entry for eventAPIError")
	}
	entry := rec.Entry("WARN", eventAPIError)
	if !entry.HasField("status") {
		t.Error("expected field 'status' on eventAPIError entry")
	}
	if !entry.HasField("body") {
		t.Error("expected field 'body' on eventAPIError entry")
	}
}

func TestMapAPIError(t *testing.T) {
	log := logging.NewDefaultLogger()

	t.Run("non-APIError passes through", func(t *testing.T) {
		origErr := fmt.Errorf("some network error")
		result, err := mapAPIError(log, origErr)
		if result != nil {
			t.Fatalf("expected nil result, got %v", result)
		}
		if err != origErr {
			t.Fatalf("expected original error, got %v", err)
		}
	})

	statusTests := []struct {
		name      string
		status    int
		body      string
		wantTool  bool // true = tool error returned, false = passes through
		errSubstr string
	}{
		{
			name:      "400 bad request",
			status:    400,
			body:      "invalid value for field",
			wantTool:  true,
			errSubstr: "Trello rejected the request (400)",
		},
		{
			name:      "400 includes body",
			status:    400,
			body:      "invalid value for field",
			wantTool:  true,
			errSubstr: "invalid value for field",
		},
		{
			name:      "401 unauthorized",
			status:    401,
			body:      "unauthorized",
			wantTool:  true,
			errSubstr: "Authentication failed",
		},
		{
			name:      "403 forbidden",
			status:    403,
			body:      "forbidden",
			wantTool:  true,
			errSubstr: "Access denied (403)",
		},
		{
			name:      "404 not found",
			status:    404,
			body:      "not found",
			wantTool:  true,
			errSubstr: "Resource not found (404)",
		},
		{
			name:      "429 rate limit",
			status:    429,
			body:      "rate limit",
			wantTool:  true,
			errSubstr: "rate limit exceeded",
		},
		{
			name:     "500 passes through",
			status:   500,
			body:     "internal server error",
			wantTool: false,
		},
		{
			name:     "502 passes through",
			status:   502,
			body:     "bad gateway",
			wantTool: false,
		},
	}

	for _, tt := range statusTests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &trello.APIError{
				StatusCode: tt.status,
				Body:       tt.body,
			}

			result, err := mapAPIError(log, apiErr)

			if tt.wantTool {
				if result == nil {
					t.Fatal("expected tool error result, got nil")
				}
				if err != nil {
					t.Fatalf("expected nil error with tool result, got %v", err)
				}
				if !result.IsError {
					t.Fatal("expected IsError to be true")
				}
				text := resultText(result)
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error text %q does not contain %q", text, tt.errSubstr)
				}
			} else {
				if result != nil {
					t.Fatalf("expected nil result for passthrough, got %v", result)
				}
				if err != apiErr {
					t.Fatalf("expected original APIError to pass through, got %v", err)
				}
			}
		})
	}
}
