package clipboard

import (
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

	adapter := &CommandAdapter{Command: testCmd}
	input := []byte("hello clipboard")

	if err := adapter.Write(ctx, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdapter_Explicit(t *testing.T) {
	adapter, err := GetAdapter("my-custom-copy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmdAdapter, ok := adapter.(*CommandAdapter)
	if !ok {
		t.Fatalf("expected *CommandAdapter, got %T", adapter)
	}

	if len(cmdAdapter.Command) != 1 || cmdAdapter.Command[0] != "my-custom-copy" {
		t.Errorf("expected command [my-custom-copy], got %v", cmdAdapter.Command)
	}
}

func TestGetAdapter_Default(t *testing.T) {
	adapter, err := GetAdapter("")
	if err != nil {
		// Depending on the OS/environment, LookPath might fail on headless Linux CI, so we allow errors on unsupported/headless systems, but it shouldn't panic.
		t.Logf("GetAdapter default returned error (this is normal on some headless/unsupported systems): %v", err)
		return
	}

	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}
