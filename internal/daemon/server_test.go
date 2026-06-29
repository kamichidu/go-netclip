package daemon

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

func TestShellScriptClient(t *testing.T) {
	// Skip on Windows because Bash scripts typically require bash environment
	if runtime.GOOS == "windows" {
		t.Skip("skipping bash script test on windows")
	}

	// Create a temp directory for our unix socket
	tempDir := t.TempDir()
	socketPath := filepath.Join(tempDir, "netclip-test.sock")

	// Use 'cat' as a mock clipboard tool
	srv := NewServer("", "cat", "echo test-paste-data")

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	defer ln.Close()

	// Start the server in the background
	go func() {
		_ = srv.StartWithListener(ln)
	}()

	// Wait a moment for server to start
	time.Sleep(50 * time.Millisecond)

	scriptPath, err := filepath.Abs("../../example/netclip-client/netclip")
	if err != nil {
		t.Fatalf("failed to get absolute path to script: %v", err)
	}

	// Test case 1: Copy operation via stdin
	t.Run("copy", func(t *testing.T) {
		cmd := exec.Command("/bin/bash", scriptPath, "copy")
		cmd.Env = append(os.Environ(), "NETCLIP_SOCK="+socketPath)

		inputData := "hello from client script"
		cmd.Stdin = bytes.NewReader([]byte(inputData))

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("script failed: %v, stderr: %s", err, stderr.String())
		}
	})

	// Test case 2: Paste operation
	t.Run("paste", func(t *testing.T) {
		cmd := exec.Command("/bin/bash", scriptPath, "paste")
		cmd.Env = append(os.Environ(), "NETCLIP_SOCK="+socketPath)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("script failed: %v, stderr: %s", err, stderr.String())
		}

		expected := "test-paste-data\n"
		if stdout.String() != expected {
			t.Errorf("expected stdout %q, got %q", expected, stdout.String())
		}
	})

	// Test case 3: Default behavior without arguments (should act as copy)
	t.Run("default_copy", func(t *testing.T) {
		cmd := exec.Command("/bin/bash", scriptPath)
		cmd.Env = append(os.Environ(), "NETCLIP_SOCK="+socketPath)

		inputData := "default copy stdin test"
		cmd.Stdin = bytes.NewReader([]byte(inputData))

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("script failed: %v, stderr: %s", err, stderr.String())
		}
	})

	// Test case 4: Help command
	t.Run("help", func(t *testing.T) {
		cmd := exec.Command("/bin/bash", scriptPath, "help")
		cmd.Env = append(os.Environ(), "NETCLIP_SOCK="+socketPath)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("script failed: %v, stderr: %s", err, stderr.String())
		}

		if !bytes.Contains(stdout.Bytes(), []byte("Usage:")) {
			t.Errorf("expected usage in stdout, got %q", stdout.String())
		}
	})

	// Test case 5: Unknown command
	t.Run("unknown_command", func(t *testing.T) {
		cmd := exec.Command("/bin/bash", scriptPath, "invalid-cmd")
		cmd.Env = append(os.Environ(), "NETCLIP_SOCK="+socketPath)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected script to fail with unknown command, but it succeeded")
		}

		if !bytes.Contains(stderr.Bytes(), []byte("Error: unknown command 'invalid-cmd'")) {
			t.Errorf("expected error message in stderr, got %q", stderr.String())
		}
	})
}
