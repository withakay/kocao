package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/withakay/kocao/internal/symphony/workflow"
)

const (
	EventSessionStarted       = "session_started"
	EventTurnCompleted        = "turn_completed"
	EventTurnFailed           = "turn_failed"
	EventTurnCancelled        = "turn_cancelled"
	EventTurnInputRequired    = "turn_input_required"
	EventApprovalAutoApproved = "approval_auto_approved"
	EventUnsupportedToolCall  = "unsupported_tool_call"
	EventNotification         = "notification"
	EventOtherMessage         = "other_message"
	EventMalformed            = "malformed"
	EventStartupFailed        = "startup_failed"

	ErrCodeInvalidWorkspaceCWD = "invalid_workspace_cwd"
	ErrCodeResponseTimeout     = "response_timeout"
	ErrCodeTurnTimeout         = "turn_timeout"
	ErrCodePortExit            = "port_exit"
	ErrCodeResponseError       = "response_error"
	ErrCodeTurnFailed          = "turn_failed"
	ErrCodeTurnCancelled       = "turn_cancelled"
	ErrCodeTurnInputRequired   = "turn_input_required"
)

type Error struct {
	Code string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type Usage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type Event struct {
	Event             string         `json:"event"`
	Timestamp         time.Time      `json:"timestamp"`
	CodexAppServerPID int            `json:"codexAppServerPid,omitempty"`
	SessionID         string         `json:"sessionId,omitempty"`
	ThreadID          string         `json:"threadId,omitempty"`
	TurnID            string         `json:"turnId,omitempty"`
	Message           string         `json:"message,omitempty"`
	Method            string         `json:"method,omitempty"`
	Usage             Usage          `json:"usage,omitempty"`
	RateLimits        map[string]any `json:"rateLimits,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
}

type TurnResult struct {
	TurnID    string    `json:"turnId"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
	Usage     Usage     `json:"usage"`
}

type Result struct {
	ThreadID    string         `json:"threadId"`
	Turns       []TurnResult   `json:"turns"`
	Usage       Usage          `json:"usage"`
	RateLimits  map[string]any `json:"rateLimits,omitempty"`
	LastEvent   string         `json:"lastEvent,omitempty"`
	LastMessage string         `json:"lastMessage,omitempty"`
}

type Options struct {
	Workspace string
	Title     string
	Prompts   []string
	Config    workflow.CodexConfig
	OnEvent   func(Event)
}

type rpcMessage struct {
	JSONRPC string         `json:"jsonrpc,omitempty"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
	Result  map[string]any `json:"result,omitempty"`
	Error   map[string]any `json:"error,omitempty"`
}

type appServerClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	workdir string
	onEvent func(Event)

	mu      sync.Mutex
	waiters map[string]chan rpcMessage
	events  chan rpcMessage
	errs    chan error
	pid     int
	reqSeq  int

	threadID   string
	lastUsage  Usage
	rateLimits map[string]any
	lastEvent  string
	lastMsg    string
}

func Run(ctx context.Context, opts Options) (Result, error) {
	workdir := strings.TrimSpace(opts.Workspace)
	if workdir == "" {
		return Result{}, &Error{Code: ErrCodeInvalidWorkspaceCWD, Err: fmt.Errorf("workspace is required")}
	}
	if !filepath.IsAbs(workdir) {
		return Result{}, &Error{Code: ErrCodeInvalidWorkspaceCWD, Err: fmt.Errorf("workspace must be absolute")}
	}
	if len(opts.Prompts) == 0 {
		return Result{}, &Error{Code: ErrCodeTurnFailed, Err: fmt.Errorf("at least one prompt is required")}
	}
	client, err := startClient(ctx, workdir, opts.Config, opts.OnEvent)
	if err != nil {
		return Result{}, err
	}
	defer client.close()

	if err := client.initialize(ctx, opts.Config); err != nil {
		client.emit(Event{Event: EventStartupFailed, Timestamp: time.Now().UTC(), CodexAppServerPID: client.pid, Message: err.Error()})
		return Result{}, err
	}
	threadID, err := client.startThread(ctx, opts.Config)
	if err != nil {
		client.emit(Event{Event: EventStartupFailed, Timestamp: time.Now().UTC(), CodexAppServerPID: client.pid, Message: err.Error()})
		return Result{}, err
	}
	client.threadID = threadID

	result := Result{ThreadID: threadID}
	for _, prompt := range opts.Prompts {
		turn, err := client.runTurn(ctx, opts.Config, threadID, opts.Title, prompt)
		if err != nil {
			result.Usage = client.lastUsage
			result.RateLimits = cloneMap(client.rateLimits)
			result.LastEvent = client.lastEvent
			result.LastMessage = client.lastMsg
			return result, err
		}
		result.Turns = append(result.Turns, turn)
	}
	result.Usage = client.lastUsage
	result.RateLimits = cloneMap(client.rateLimits)
	result.LastEvent = client.lastEvent
	result.LastMessage = client.lastMsg
	return result, nil
}

func startClient(ctx context.Context, workdir string, cfg workflow.CodexConfig, onEvent func(Event)) (*appServerClient, error) {
	command := strings.TrimSpace(cfg.Command)
	if command == "" {
		command = "codex app-server"
	}
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = workdir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	client := &appServerClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		workdir: workdir,
		onEvent: onEvent,
		waiters: map[string]chan rpcMessage{},
		events:  make(chan rpcMessage, 64),
		errs:    make(chan error, 2),
		reqSeq:  2,
	}
	if err := cmd.Start(); err != nil {
		return nil, &Error{Code: ErrCodePortExit, Err: err}
	}
	client.pid = cmd.Process.Pid
	go client.readStdout()
	go client.readStderr()
	go func() {
		if err := cmd.Wait(); err != nil {
			client.errs <- &Error{Code: ErrCodePortExit, Err: err}
			return
		}
		client.errs <- io.EOF
	}()
	return client, nil
}

func (c *appServerClient) close() {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.stderr != nil {
		_ = c.stderr.Close()
	}
}

func (c *appServerClient) initialize(ctx context.Context, cfg workflow.CodexConfig) error {
	_, err := c.request(ctx, cfg.ReadTimeoutMS, rpcMessage{
		JSONRPC: "2.0",
		ID:      "initialize-1",
		Method:  "initialize",
		Params: map[string]any{
			"clientInfo": map[string]any{
				"name":    "kocao",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{},
		},
	})
	if err != nil {
		return err
	}
	return c.send(rpcMessage{JSONRPC: "2.0", Method: "initialized", Params: map[string]any{}})
}

func (c *appServerClient) startThread(ctx context.Context, cfg workflow.CodexConfig) (string, error) {
	params := map[string]any{"cwd": c.workdir}
	if cfg.ApprovalPolicy != "" {
		params["approvalPolicy"] = cfg.ApprovalPolicy
	}
	if cfg.ThreadSandbox != "" {
		params["sandbox"] = cfg.ThreadSandbox
	}
	resp, err := c.request(ctx, cfg.ReadTimeoutMS, rpcMessage{JSONRPC: "2.0", ID: "thread-start-1", Method: "thread/start", Params: params})
	if err != nil {
		return "", err
	}
	threadID := nestedString(resp.Result, "thread", "id")
	if threadID == "" {
		return "", &Error{Code: ErrCodeResponseError, Err: fmt.Errorf("thread/start missing thread id")}
	}
	return threadID, nil
}

func (c *appServerClient) runTurn(ctx context.Context, cfg workflow.CodexConfig, threadID, title, prompt string) (TurnResult, error) {
	startedAt := time.Now().UTC()
	params := map[string]any{
		"threadId": threadID,
		"input":    []map[string]any{{"type": "text", "text": prompt}},
		"cwd":      c.workdir,
		"title":    title,
	}
	if cfg.ApprovalPolicy != "" {
		params["approvalPolicy"] = cfg.ApprovalPolicy
	}
	if cfg.TurnSandboxPolicy != "" {
		params["sandboxPolicy"] = map[string]any{"type": cfg.TurnSandboxPolicy}
	}
	c.reqSeq++
	turnIDResp, err := c.request(ctx, cfg.ReadTimeoutMS, rpcMessage{JSONRPC: "2.0", ID: fmt.Sprintf("turn-start-%d", c.reqSeq), Method: "turn/start", Params: params})
	if err != nil {
		return TurnResult{}, err
	}
	turnID := nestedString(turnIDResp.Result, "turn", "id")
	if turnID == "" {
		return TurnResult{}, &Error{Code: ErrCodeResponseError, Err: fmt.Errorf("turn/start missing turn id")}
	}
	startedEvent := Event{Event: EventSessionStarted, Timestamp: startedAt, CodexAppServerPID: c.pid, ThreadID: threadID, TurnID: turnID, SessionID: sessionID(threadID, turnID)}
	c.emit(startedEvent)
	deadline := time.NewTimer(time.Duration(cfg.TurnTimeoutMS) * time.Millisecond)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return TurnResult{}, ctx.Err()
		case err := <-c.errs:
			if err == io.EOF {
				return TurnResult{}, &Error{Code: ErrCodePortExit, Err: io.EOF}
			}
			return TurnResult{}, err
		case msg := <-c.events:
			outcome, event, err := c.handleEvent(msg, threadID, turnID)
			if event.Event != "" {
				c.emit(event)
			}
			if err != nil {
				return TurnResult{}, err
			}
			if outcome != "" {
				return TurnResult{TurnID: turnID, Status: outcome, StartedAt: startedAt, EndedAt: time.Now().UTC(), Usage: c.lastUsage}, nil
			}
		case <-deadline.C:
			return TurnResult{}, &Error{Code: ErrCodeTurnTimeout, Err: fmt.Errorf("turn %s exceeded timeout", turnID)}
		}
	}
}

func (c *appServerClient) handleEvent(msg rpcMessage, threadID, turnID string) (string, Event, error) {
	now := time.Now().UTC()
	event := Event{Timestamp: now, CodexAppServerPID: c.pid, ThreadID: threadID, TurnID: turnID, SessionID: sessionID(threadID, turnID), Method: msg.Method, Payload: msg.Params}
	if usage, ok := extractUsage(msg.Params); ok {
		c.lastUsage = usage
		event.Usage = usage
	}
	if limits := extractRateLimits(msg.Params); limits != nil {
		c.rateLimits = limits
		event.RateLimits = cloneMap(limits)
	}
	event.Message = summarizeMessage(msg.Params)
	switch msg.Method {
	case "turn/completed":
		event.Event = EventTurnCompleted
		c.lastEvent = event.Event
		c.lastMsg = event.Message
		return "completed", event, nil
	case "turn/failed":
		event.Event = EventTurnFailed
		c.lastEvent = event.Event
		c.lastMsg = event.Message
		return "", event, &Error{Code: ErrCodeTurnFailed, Err: fmt.Errorf("%s", firstNonEmpty(event.Message, "turn failed"))}
	case "turn/cancelled":
		event.Event = EventTurnCancelled
		c.lastEvent = event.Event
		c.lastMsg = event.Message
		return "", event, &Error{Code: ErrCodeTurnCancelled, Err: fmt.Errorf("%s", firstNonEmpty(event.Message, "turn cancelled"))}
	}
	if msg.ID != nil && msg.Method != "" {
		if strings.Contains(strings.ToLower(msg.Method), "requestuserinput") {
			event.Event = EventTurnInputRequired
			c.lastEvent = event.Event
			c.lastMsg = event.Message
			_ = c.send(rpcMessage{JSONRPC: "2.0", ID: msg.ID, Error: map[string]any{"message": "user input unsupported"}})
			return "", event, &Error{Code: ErrCodeTurnInputRequired, Err: fmt.Errorf("user input required")}
		}
		if strings.Contains(strings.ToLower(msg.Method), "approve") || strings.Contains(strings.ToLower(msg.Method), "approval") {
			event.Event = EventApprovalAutoApproved
			_ = c.send(rpcMessage{JSONRPC: "2.0", ID: msg.ID, Result: map[string]any{"approved": true}})
			return "", event, nil
		}
		event.Event = EventUnsupportedToolCall
		_ = c.send(rpcMessage{JSONRPC: "2.0", ID: msg.ID, Result: map[string]any{"success": false, "error": "unsupported_tool_call"}})
		return "", event, nil
	}
	event.Event = EventNotification
	c.lastEvent = event.Event
	c.lastMsg = event.Message
	return "", event, nil
}

func (c *appServerClient) request(ctx context.Context, timeoutMS int, msg rpcMessage) (rpcMessage, error) {
	responseKey := idKey(msg.ID)
	if responseKey == "" {
		return rpcMessage{}, &Error{Code: ErrCodeResponseError, Err: fmt.Errorf("request id is required")}
	}
	ch := make(chan rpcMessage, 1)
	c.mu.Lock()
	c.waiters[responseKey] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.waiters, responseKey)
		c.mu.Unlock()
	}()
	if err := c.send(msg); err != nil {
		return rpcMessage{}, err
	}
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return rpcMessage{}, ctx.Err()
		case err := <-c.errs:
			if err == io.EOF {
				return rpcMessage{}, &Error{Code: ErrCodePortExit, Err: io.EOF}
			}
			return rpcMessage{}, err
		case resp := <-ch:
			if len(resp.Error) != 0 {
				return rpcMessage{}, &Error{Code: ErrCodeResponseError, Err: fmt.Errorf("%s", firstNonEmpty(stringValue(resp.Error["message"]), "request failed"))}
			}
			return resp, nil
		case msg := <-c.events:
			if _, event, err := c.handleEvent(msg, c.threadID, ""); event.Event != "" {
				c.emit(event)
				if err != nil {
					return rpcMessage{}, err
				}
			}
		case <-timer.C:
			return rpcMessage{}, &Error{Code: ErrCodeResponseTimeout, Err: fmt.Errorf("timed out waiting for response to %s", msg.Method)}
		}
	}
}

func (c *appServerClient) send(msg rpcMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = c.stdin.Write(append(b, '\n'))
	return err
}

func (c *appServerClient) readStdout() {
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		var msg rpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			c.emit(Event{Event: EventMalformed, Timestamp: time.Now().UTC(), CodexAppServerPID: c.pid, Message: string(line)})
			continue
		}
		if msg.ID != nil && (msg.Result != nil || msg.Error != nil) {
			key := idKey(msg.ID)
			c.mu.Lock()
			ch := c.waiters[key]
			c.mu.Unlock()
			if ch != nil {
				ch <- msg
				continue
			}
		}
		c.events <- msg
	}
	if err := scanner.Err(); err != nil {
		c.errs <- err
	}
}

func (c *appServerClient) readStderr() {
	scanner := bufio.NewScanner(c.stderr)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for scanner.Scan() {
		c.emit(Event{Event: EventOtherMessage, Timestamp: time.Now().UTC(), CodexAppServerPID: c.pid, Message: scanner.Text()})
	}
}

func (c *appServerClient) emit(event Event) {
	if event.Event == "" {
		return
	}
	if c.onEvent != nil {
		c.onEvent(event)
	}
}

func extractUsage(value any) (Usage, bool) {
	mapped, ok := value.(map[string]any)
	if !ok {
		return Usage{}, false
	}
	if total, ok := mapped["total_token_usage"].(map[string]any); ok {
		if usage := parseUsageMap(total); usage.TotalTokens > 0 || usage.InputTokens > 0 || usage.OutputTokens > 0 {
			return usage, true
		}
	}
	for _, key := range []string{"usage", "tokenUsage", "totalTokenUsage"} {
		if child, ok := mapped[key].(map[string]any); ok {
			if usage := parseUsageMap(child); usage.TotalTokens > 0 || usage.InputTokens > 0 || usage.OutputTokens > 0 {
				return usage, true
			}
		}
	}
	if usage := parseUsageMap(mapped); usage.TotalTokens > 0 || usage.InputTokens > 0 || usage.OutputTokens > 0 {
		return usage, true
	}
	for _, child := range mapped {
		if usage, ok := extractUsage(child); ok {
			return usage, true
		}
	}
	return Usage{}, false
}

func parseUsageMap(mapped map[string]any) Usage {
	return Usage{
		InputTokens:  intFromAny(firstNonNil(mapped["input_tokens"], mapped["inputTokens"])),
		OutputTokens: intFromAny(firstNonNil(mapped["output_tokens"], mapped["outputTokens"])),
		TotalTokens:  intFromAny(firstNonNil(mapped["total_tokens"], mapped["totalTokens"])),
	}
}

func extractRateLimits(value any) map[string]any {
	mapped, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"rate_limits", "rateLimits"} {
		if child, ok := mapped[key].(map[string]any); ok {
			return cloneMap(child)
		}
	}
	for _, child := range mapped {
		if limits := extractRateLimits(child); limits != nil {
			return limits
		}
	}
	return nil
}

func summarizeMessage(value any) string {
	mapped, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"message", "summary", "text", "reason"} {
		if text := strings.TrimSpace(stringValue(mapped[key])); text != "" {
			return text
		}
	}
	for _, child := range mapped {
		if text := summarizeMessage(child); text != "" {
			return text
		}
	}
	return ""
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func idKey(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func nestedString(root map[string]any, keys ...string) string {
	current := root
	for i, key := range keys {
		value, ok := current[key]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			return strings.TrimSpace(stringValue(value))
		}
		next, ok := value.(map[string]any)
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func sessionID(threadID, turnID string) string {
	if threadID == "" || turnID == "" {
		return ""
	}
	return threadID + "-" + turnID
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}
