package daemon

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"runtime"
	"testing"
	"time"
)

func TestServer_handleCopy(t *testing.T) {
	// Let's create a listener on an ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	// Use 'cat' or 'findstr' as a mock clipboard tool
	var mockCmd string
	if runtime.GOOS == "windows" {
		mockCmd = "findstr ^"
	} else {
		mockCmd = "cat"
	}

	srv := NewServer(ln.Addr().String(), mockCmd, "")

	// Start the server in the background
	go func() {
		_ = srv.StartWithListener(ln)
	}()

	// Wait a moment for server to start
	time.Sleep(50 * time.Millisecond)

	url := "http://" + ln.Addr().String() + "/copy"

	// 1. Send valid POST request
	resp, err := http.Post(url, "application/octet-stream", bytes.NewReader([]byte("test data")))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 No Content, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Send invalid GET request (should return 405)
	respGet, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	if respGet.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", respGet.StatusCode)
	}
	respGet.Body.Close()

	// 3. Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown server: %v", err)
	}
}

func TestServer_handlePaste(t *testing.T) {
	// Let's create a listener on an ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	var mockCopyCmd string
	var mockPasteCmd string
	if runtime.GOOS == "windows" {
		mockCopyCmd = "findstr ^"
		mockPasteCmd = "cmd /c echo test paste"
	} else {
		mockCopyCmd = "cat"
		mockPasteCmd = "echo test paste"
	}

	srv := NewServer(ln.Addr().String(), mockCopyCmd, mockPasteCmd)

	// Start the server in the background
	go func() {
		_ = srv.StartWithListener(ln)
	}()

	// Wait a moment for server to start
	time.Sleep(50 * time.Millisecond)

	url := "http://" + ln.Addr().String() + "/paste"

	// 1. Send valid GET request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var expected []byte
	if runtime.GOOS == "windows" {
		expected = []byte("test paste\r\n")
	} else {
		expected = []byte("test paste\n")
	}

	if !bytes.Equal(body, expected) {
		t.Errorf("expected %q, got %q", expected, body)
	}

	// 2. Send invalid POST request (should return 405)
	respPost, err := http.Post(url, "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer respPost.Body.Close()

	if respPost.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", respPost.StatusCode)
	}

	// 3. Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown server: %v", err)
	}
}
