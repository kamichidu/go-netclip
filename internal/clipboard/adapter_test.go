package clipboard

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"
)

func TestCommandAdapter_Write(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use 'cat' on non-Windows systems, and 'findstr' or similar on Windows, as a mock clipboard tool
	var testCmd []string
	if runtime.GOOS == "windows" {
		testCmd = []string{"findstr", "^"}
	} else {
		testCmd = []string{"cat"}
	}

	adapter := &CommandAdapter{CopyCommand: testCmd}
	input := []byte("hello clipboard")

	if err := adapter.Write(ctx, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandAdapter_Read(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var testCmd []string
	var expected []byte
	if runtime.GOOS == "windows" {
		testCmd = []string{"cmd", "/c", "echo hello"}
		expected = []byte("hello\r\n")
	} else {
		testCmd = []string{"echo", "hello"}
		expected = []byte("hello\n")
	}

	adapter := &CommandAdapter{PasteCommand: testCmd}
	output, err := adapter.Read(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(output, expected) {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestGetAdapter_Explicit(t *testing.T) {
	adapter, err := GetAdapter("my-custom-copy", "my-custom-paste")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmdAdapter, ok := adapter.(*CommandAdapter)
	if !ok {
		t.Fatalf("expected *CommandAdapter, got %T", adapter)
	}

	if len(cmdAdapter.CopyCommand) != 1 || cmdAdapter.CopyCommand[0] != "my-custom-copy" {
		t.Errorf("expected copy command [my-custom-copy], got %v", cmdAdapter.CopyCommand)
	}

	if len(cmdAdapter.PasteCommand) != 1 || cmdAdapter.PasteCommand[0] != "my-custom-paste" {
		t.Errorf("expected paste command [my-custom-paste], got %v", cmdAdapter.PasteCommand)
	}
}

func TestGetAdapter_Default(t *testing.T) {
	adapter, err := GetAdapter("", "")
	if err != nil {
		// Depending on the OS/environment, LookPath might fail on headless Linux CI, so we allow errors on unsupported/headless systems, but it shouldn't panic.
		t.Logf("GetAdapter default returned error (this is normal on some headless/unsupported systems): %v", err)
		return
	}

	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}
