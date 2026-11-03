package httpapi

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeStreamErrorRedactsTransportDetails(t *testing.T) {
	cases := []string{
		`Post "http://127.0.0.1:19080/internal/orchestrator/runs/stream": dial tcp 127.0.0.1:19080: connect: connection refused`,
		`dial tcp: lookup orchestrator on 127.0.0.11:53: no such host`,
		`read tcp 10.0.0.2:1234->10.0.0.3:9000: i/o timeout`,
		`dial tcp 10.0.0.3:9000: connectex: No connection could be made because the target machine actively refused it.`,
	}
	for _, raw := range cases {
		got := sanitizeStreamError(errors.New(raw))
		for _, leaked := range []string{"connection refused", "dial tcp", "no such host", "i/o timeout", "127.0.0.1", "10.0.0", "connectex"} {
			if strings.Contains(strings.ToLower(got), strings.ToLower(leaked)) {
				t.Fatalf("sanitized error leaked %q: %q", leaked, got)
			}
		}
		if got != "orchestrator communication error" {
			t.Fatalf("got %q, want generic communication error", got)
		}
	}
}
