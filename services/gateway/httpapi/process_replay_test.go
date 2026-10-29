package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// TestGatewayChildProcess is only executed as a spawned child OS process.
func TestGatewayChildProcess(t *testing.T) {
	if os.Getenv("AGENTHUB_GATEWAY_CHILD") != "1" {
		t.Skip("child process helper")
	}
	addr := os.Getenv("CHILD_ADDR")
	dsn := os.Getenv("DATABASE_URL")
	if addr == "" || dsn == "" {
		t.Fatal("child addr/dsn missing")
	}
	st, err := store.OpenMySQL(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(st, &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Addr: addr, Handler: srv.Handler()}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	return addr
}

func TestTwoGatewayProcessesReplaySameRun(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP multi-process integration: set AGENTHUB_MYSQL_TEST_DSN")
	}
	ctx := context.Background()
	st, err := store.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	runID := fmt.Sprintf("run-proc-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = st.DB().Exec("DELETE FROM run_public_events WHERE run_id = ?", runID)
	})
	for i := 1; i <= 3; i++ {
		if err := st.AppendPublicEvent(ctx, store.PublicEvent{
			RunID: runID, Seq: int64(i), Type: "TEXT_MESSAGE_CONTENT",
			Payload: json.RawMessage(fmt.Sprintf(`{"type":"TEXT_MESSAGE_CONTENT","runId":%q,"seq":%d,"delta":"x"}`, runID, i)),
		}); err != nil {
			t.Fatal(err)
		}
	}

	addrs := []string{freePort(t), freePort(t)}
	children := make([]*exec.Cmd, 0, len(addrs))
	for _, addr := range addrs {
		cmd := exec.Command(os.Args[0], "-test.run=^TestGatewayChildProcess$", "-test.v")
		cmd.Env = append(os.Environ(),
			"AGENTHUB_GATEWAY_CHILD=1",
			"CHILD_ADDR="+addr,
			"DATABASE_URL="+dsn,
		)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		children = append(children, cmd)
		t.Cleanup(func() { _ = cmd.Process.Kill() })
	}
	for _, addr := range addrs {
		deadline := time.Now().Add(15 * time.Second)
		ready := false
		for time.Now().Before(deadline) {
			resp, err := http.Get("http://" + addr + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					ready = true
					break
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !ready {
			t.Fatalf("gateway child %s did not become ready", addr)
		}
	}
	for _, addr := range addrs {
		resp, err := http.Get("http://" + addr + "/api/runs/" + runID + "/events?after=0&limit=100")
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Events []map[string]any `json:"events"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
		resp.Body.Close()
		if len(body.Events) != 3 {
			t.Fatalf("process %s received %d events, want 3", addr, len(body.Events))
		}
		for i, event := range body.Events {
			if event["seq"] != float64(i+1) {
				t.Fatalf("process %s event %d has seq %v", addr, i, event["seq"])
			}
		}
	}
}
