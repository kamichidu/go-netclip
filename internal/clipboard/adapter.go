package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// Adapter defines the interface for writing to the system clipboard.
type Adapter interface {
	Write(ctx context.Context, data []byte) error
}

// CommandAdapter executes a CLI tool to write to the clipboard.
type CommandAdapter struct {
	Command []string
}

// Write writes data to the clipboard by piping it to the configured command's stdin.
func (a *CommandAdapter) Write(ctx context.Context, data []byte) error {
	if len(a.Command) == 0 {
		return fmt.Errorf("no clipboard command configured")
	}

	cmd := exec.CommandContext(ctx, a.Command[0], a.Command[1:]...)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %v: %w (stderr: %q)", a.Command, err, stderr.String())
	}
	return nil
}

// GetAdapter resolves the correct clipboard adapter based on the OS or explicit override.
func GetAdapter(explicitCmd string) (Adapter, error) {
	if explicitCmd != "" {
		return &CommandAdapter{Command: []string{explicitCmd}}, nil
	}

	switch runtime.GOOS {
	case "darwin":
		return &CommandAdapter{Command: []string{"pbcopy"}}, nil
	case "windows":
		if _, err := exec.LookPath("clip.exe"); err == nil {
			return &CommandAdapter{Command: []string{"clip.exe"}}, nil
		}
		return &CommandAdapter{Command: []string{"powershell.exe", "-NoProfile", "-Command", "Set-Clipboard"}}, nil
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return &CommandAdapter{Command: []string{"wl-copy"}}, nil
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return &CommandAdapter{Command: []string{"xclip", "-selection", "clipboard"}}, nil
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return &CommandAdapter{Command: []string{"xsel", "--clipboard", "--input"}}, nil
		}
		return nil, fmt.Errorf("no clipboard command found on Linux (install wl-copy, xclip, or xsel)")
	default:
		return nil, fmt.Errorf("unsupported OS %q, please specify via --clipboard-command", runtime.GOOS)
	}
}
