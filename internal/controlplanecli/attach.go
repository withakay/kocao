package controlplanecli

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type attachMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Cols    int    `json:"cols,omitempty"`
	Rows    int    `json:"rows,omitempty"`
	Message string `json:"message,omitempty"`
	Role    string `json:"role,omitempty"`
}

func runSessionAttachCommand(cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao sessions attach <workspace-session-id> [--driver]")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" || strings.HasPrefix(sessionID, "-") {
		return fmt.Errorf("usage: kocao sessions attach <workspace-session-id> [--driver]")
	}

	fs := flag.NewFlagSet("kocao sessions attach", flag.ContinueOnError)
	fs.SetOutput(stderr)
	driver := fs.Bool("driver", false, "request driver role")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	role := "viewer"
	if *driver {
		role = "driver"
	}
	return attachSession(ctx, client, sessionID, role, stdout, stderr)
}

func attachSession(ctx context.Context, client *Client, workspaceSessionID string, role string, stdout io.Writer, stderr io.Writer) error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("attach requires an interactive terminal (TTY)")
	}

	tok, err := client.CreateAttachToken(ctx, workspaceSessionID, role)
	if err != nil {
		return err
	}

	wsURL, origin, err := attachWSURL(client.baseURL, workspaceSessionID)
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+tok.Token)
	headers.Set("Origin", origin)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return fmt.Errorf("connect attach websocket: %w", err)
	}
	defer func() { _ = conn.Close() }()

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("set terminal raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 4)
	sendCh := make(chan attachMessage, 64)

	go attachWriter(ctx, conn, sendCh, errCh)
	go attachReader(ctx, conn, stdout, stderr, errCh)
	go attachKeepalive(ctx, sendCh)
	go attachResize(ctx, fd, sendCh)
	if role == "driver" {
		go attachStdin(ctx, sendCh, errCh)
		sendCh <- attachMessage{Type: "take_control"}
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		cancel()
		if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
}

func attachWSURL(baseURL *url.URL, workspaceSessionID string) (string, string, error) {
	u := *baseURL
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", "", fmt.Errorf("api url must use http or https")
	}
	u.Path = path.Join("/", strings.TrimPrefix(baseURL.Path, "/"), "api/v1/workspace-sessions", url.PathEscape(strings.TrimSpace(workspaceSessionID)), "attach")
	u.RawQuery = ""
	u.Fragment = ""

	originURL := &url.URL{Scheme: baseURL.Scheme, Host: baseURL.Host}
	return u.String(), originURL.String(), nil
}

func attachWriter(ctx context.Context, conn *websocket.Conn, sendCh <-chan attachMessage, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errCh <- nil
			return
		case msg := <-sendCh:
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(msg); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func attachReader(ctx context.Context, conn *websocket.Conn, stdout io.Writer, stderr io.Writer, errCh chan<- error) {
	for {
		var m attachMessage
		if err := conn.ReadJSON(&m); err != nil {
			errCh <- err
			return
		}
		switch m.Type {
		case "stdout":
			b, err := base64.StdEncoding.DecodeString(m.Data)
			if err == nil {
				_, _ = stdout.Write(b)
			}
		case "error":
			_, _ = fmt.Fprintf(stderr, "\r\nattach error: %s\r\n", strings.TrimSpace(m.Message))
		case "backend_closed":
			_, _ = fmt.Fprintln(stderr, "\r\nattach backend closed")
			errCh <- io.EOF
			return
		case "hello":
			if strings.EqualFold(m.Role, "viewer") {
				_, _ = fmt.Fprintln(stderr, "\r\nconnected in viewer mode (read-only)")
			}
		}

		select {
		case <-ctx.Done():
			errCh <- nil
			return
		default:
		}
	}
}

func attachKeepalive(ctx context.Context, sendCh chan<- attachMessage) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendCh <- attachMessage{Type: "keepalive"}
		}
	}
}

func attachResize(ctx context.Context, fd int, sendCh chan<- attachMessage) {
	if cols, rows, err := term.GetSize(fd); err == nil && cols > 0 && rows > 0 {
		sendCh <- attachMessage{Type: "resize", Cols: cols, Rows: rows}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			cols, rows, err := term.GetSize(fd)
			if err == nil && cols > 0 && rows > 0 {
				sendCh <- attachMessage{Type: "resize", Cols: cols, Rows: rows}
			}
		}
	}
}

func attachStdin(ctx context.Context, sendCh chan<- attachMessage, errCh chan<- error) {
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			payload := base64.StdEncoding.EncodeToString(buf[:n])
			select {
			case <-ctx.Done():
				return
			case sendCh <- attachMessage{Type: "stdin", Data: payload}:
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			errCh <- err
			return
		}
	}
}
