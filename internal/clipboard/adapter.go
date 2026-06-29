package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Adapter defines the interface for writing to and reading from the system clipboard.
type Adapter interface {
	Write(ctx context.Context, data []byte) error
	Read(ctx context.Context) ([]byte, error)
}

// CommandAdapter executes a CLI tool to write to or read from the clipboard.
type CommandAdapter struct {
	CopyCommand  []string
	PasteCommand []string
}

// Write writes data to the clipboard by piping it to the configured command's stdin.
func (a *CommandAdapter) Write(ctx context.Context, data []byte) error {
	if len(a.CopyCommand) == 0 {
		return fmt.Errorf("no clipboard copy command configured")
	}

	cmd := exec.CommandContext(ctx, a.CopyCommand[0], a.CopyCommand[1:]...)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %v: %w (stderr: %q)", a.CopyCommand, err, stderr.String())
	}
	return nil
}

// Read reads data from the clipboard by executing the configured command and capturing its stdout.
func (a *CommandAdapter) Read(ctx context.Context) ([]byte, error) {
	if len(a.PasteCommand) == 0 {
		return nil, fmt.Errorf("no clipboard paste command configured")
	}

	cmd := exec.CommandContext(ctx, a.PasteCommand[0], a.PasteCommand[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr: %q)", a.PasteCommand, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// GetAdapter resolves the correct clipboard adapter based on the OS or explicit overrides.
func GetAdapter(explicitCopyCmd string, explicitPasteCmd string) (Adapter, error) {
	if explicitCopyCmd != "" || explicitPasteCmd != "" {
		adapter := &CommandAdapter{}
		if explicitCopyCmd != "" {
			adapter.CopyCommand = strings.Fields(explicitCopyCmd)
		} else {
			defaultAdapter, err := getDefaultAdapter()
			if err != nil {
				return nil, err
			}
			adapter.CopyCommand = defaultAdapter.CopyCommand
		}
		if explicitPasteCmd != "" {
			adapter.PasteCommand = strings.Fields(explicitPasteCmd)
		} else {
			defaultAdapter, err := getDefaultAdapter()
			if err != nil {
				return nil, err
			}
			adapter.PasteCommand = defaultAdapter.PasteCommand
		}
		return adapter, nil
	}

	return getDefaultAdapter()
}

func getDefaultAdapter() (*CommandAdapter, error) {
	switch runtime.GOOS {
	case "darwin":
		return &CommandAdapter{
			CopyCommand:  []string{"pbcopy"},
			PasteCommand: []string{"pbpaste"},
		}, nil
	case "windows":
		var copyCmd []string
		if _, err := exec.LookPath("clip.exe"); err == nil {
			copyCmd = []string{"clip.exe"}
		} else {
			copyCmd = []string{"powershell.exe", "-NoProfile", "-Command", "Set-Clipboard"}
		}
		return &CommandAdapter{
			CopyCommand:  copyCmd,
			PasteCommand: []string{"powershell.exe", "-NoProfile", "-Command", "Get-Clipboard"},
		}, nil
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return &CommandAdapter{
				CopyCommand:  []string{"wl-copy"},
				PasteCommand: []string{"wl-paste"},
			}, nil
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return &CommandAdapter{
				CopyCommand:  []string{"xclip", "-selection", "clipboard"},
				PasteCommand: []string{"xclip", "-selection", "clipboard", "-o"},
			}, nil
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return &CommandAdapter{
				CopyCommand:  []string{"xsel", "--clipboard", "--input"},
				PasteCommand: []string{"xsel", "--clipboard", "--output"},
			}, nil
		}
		return nil, fmt.Errorf("no clipboard command found on Linux (install wl-copy, xclip, or xsel)")
	default:
		return nil, fmt.Errorf("unsupported OS %q, please specify via --clipboard-command and --paste-command", runtime.GOOS)
	}
}
