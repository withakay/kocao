package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/withakay/kocao/internal/symphony/workflow"
)

func TestRunSupportsContinuationAndTelemetry(t *testing.T) {
	var events []Event
	result, err := Run(context.Background(), Options{
		Workspace: t.TempDir(),
		Title:     "ABC-123: Example",
		Prompts:   []string{"first prompt", "continue working"},
		Config: workflow.CodexConfig{
			Command:        helperCommand(t, "success"),
			ReadTimeoutMS:  1000,
			TurnTimeoutMS:  1000,
			ApprovalPolicy: "auto",
		},
		OnEvent: func(event Event) { events = append(events, event) },
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.ThreadID != "thread-1" {
		t.Fatalf("thread id = %q", result.ThreadID)
	}
	if len(result.Turns) != 2 {
		t.Fatalf("turn count = %d", len(result.Turns))
	}
	if result.Turns[0].TurnID != "turn-1" || result.Turns[1].TurnID != "turn-2" {
		t.Fatalf("turn ids = %#v", result.Turns)
	}
	if result.Usage.TotalTokens != 30 {
		t.Fatalf("usage = %#v", result.Usage)
	}
	if result.RateLimits["remaining"] != float64(99) {
		t.Fatalf("rate limits = %#v", result.RateLimits)
	}
	if !hasEvent(events, EventSessionStarted) || !hasEvent(events, EventTurnCompleted) {
		t.Fatalf("events = %#v", events)
	}
}

func TestRunAutoApprovesServerApprovalRequests(t *testing.T) {
	var events []Event
	_, err := Run(context.Background(), Options{
		Workspace: t.TempDir(),
		Title:     "ABC-123: Example",
		Prompts:   []string{"first prompt"},
		Config: workflow.CodexConfig{
			Command:       helperCommand(t, "approval"),
			ReadTimeoutMS: 1000,
			TurnTimeoutMS: 1000,
		},
		OnEvent: func(event Event) { events = append(events, event) },
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasEvent(events, EventApprovalAutoApproved) {
		t.Fatalf("events = %#v", events)
	}
}

func TestRunFailsOnUserInputRequirement(t *testing.T) {
	_, err := Run(context.Background(), Options{
		Workspace: t.TempDir(),
		Title:     "ABC-123: Example",
		Prompts:   []string{"first prompt"},
		Config: workflow.CodexConfig{
			Command:       helperCommand(t, "userinput"),
			ReadTimeoutMS: 1000,
			TurnTimeoutMS: 1000,
		},
	})
	var runnerErr *Error
	if !errorsAs(err, &runnerErr) {
		t.Fatalf("expected runner error, got %v", err)
	}
	if runnerErr.Code != ErrCodeTurnInputRequired {
		t.Fatalf("error code = %q", runnerErr.Code)
	}
}

func TestRunTimesOutTurn(t *testing.T) {
	_, err := Run(context.Background(), Options{
		Workspace: t.TempDir(),
		Title:     "ABC-123: Example",
		Prompts:   []string{"first prompt"},
		Config: workflow.CodexConfig{
			Command:       helperCommand(t, "hang"),
			ReadTimeoutMS: 1000,
			TurnTimeoutMS: 50,
		},
	})
	var runnerErr *Error
	if !errorsAs(err, &runnerErr) {
		t.Fatalf("expected runner error, got %v", err)
	}
	if runnerErr.Code != ErrCodeTurnTimeout {
		t.Fatalf("error code = %q", runnerErr.Code)
	}
}

func TestRunStartupTimeout(t *testing.T) {
	_, err := Run(context.Background(), Options{
		Workspace: t.TempDir(),
		Title:     "ABC-123: Example",
		Prompts:   []string{"first prompt"},
		Config: workflow.CodexConfig{
			Command:       helperCommand(t, "startup-timeout"),
			ReadTimeoutMS: 50,
			TurnTimeoutMS: 1000,
		},
	})
	var runnerErr *Error
	if !errorsAs(err, &runnerErr) {
		t.Fatalf("expected runner error, got %v", err)
	}
	if runnerErr.Code != ErrCodeResponseTimeout {
		t.Fatalf("error code = %q", runnerErr.Code)
	}
}

func helperCommand(t *testing.T, mode string) string {
	t.Helper()
	return fmt.Sprintf("GO_WANT_HELPER_PROCESS=1 %s -test.run=TestHelperProcess -- %s", os.Args[0], mode)
}

func hasEvent(events []Event, name string) bool {
	for _, event := range events {
		if event.Event == name {
			return true
		}
	}
	return false
}

func errorsAs(err error, target **Error) bool {
	return errors.As(err, target)
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || len(os.Args) < 4 {
		return
	}
	mode := os.Args[3]
	reader := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	turnCount := 0
	for reader.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(reader.Bytes(), &msg); err != nil {
			continue
		}
		method := strings.TrimSpace(stringValue(msg["method"]))
		switch method {
		case "initialize":
			if mode == "startup-timeout" {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"serverInfo": map[string]any{"name": "fake"}}})
		case "initialized":
		case "thread/start":
			writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "turn/start":
			turnCount++
			turnID := fmt.Sprintf("turn-%d", turnCount)
			writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"turn": map[string]any{"id": turnID}}})
			switch mode {
			case "success":
				writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "method": "thread/tokenUsage/updated", "params": map[string]any{"threadId": "thread-1", "total_token_usage": map[string]any{"input_tokens": 10 * turnCount, "output_tokens": 5 * turnCount, "total_tokens": 15 * turnCount}, "rate_limits": map[string]any{"remaining": 99}}})
				writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "method": "turn/completed", "params": map[string]any{"turnId": turnID, "summary": "completed"}})
			case "approval":
				writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": 99, "method": "approval/request", "params": map[string]any{"kind": "command"}})
				if !reader.Scan() {
					os.Exit(1)
				}
				var approvalResp map[string]any
				_ = json.Unmarshal(reader.Bytes(), &approvalResp)
				if result, ok := approvalResp["result"].(map[string]any); !ok || result["approved"] != true {
					os.Exit(1)
				}
				writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "method": "turn/completed", "params": map[string]any{"turnId": turnID, "summary": "completed after approval"}})
			case "userinput":
				writeHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": 100, "method": "item/tool/requestUserInput", "params": map[string]any{"prompt": "Need input"}})
				return
			case "hang":
				time.Sleep(5 * time.Second)
				return
			}
		}
	}
	os.Exit(0)
}

func writeHelperJSON(w *bufio.Writer, payload map[string]any) {
	b, _ := json.Marshal(payload)
	_, _ = w.Write(append(b, '\n'))
	_ = w.Flush()
}
