package clipboard

import (
	"fmt"
	"os/exec"

	"github.com/kaliv0/homie/internal/config"
)

const (
	BinXclip  = "xclip"
	BinXsel   = "xsel"
	BinWLCopy = "wl-copy"
)

// Binary returns the executable name for a config clipboard tool identifier.
func Binary(tool string) (string, error) {
	switch tool {
	case config.ClipboardXclip:
		return BinXclip, nil
	case config.ClipboardXsel:
		return BinXsel, nil
	case config.ClipboardWLClipboard:
		return BinWLCopy, nil
	default:
		return "", fmt.Errorf("unsupported clipboard tool: %q", tool)
	}
}

// Write writes a given string to the clipboard using the specified tool.
func Write(text, tool string) error {
	cmdName, err := Binary(tool)
	if err != nil {
		return err
	}

	var args []string
	switch tool {
	case config.ClipboardXclip:
		args = []string{"-in", "-selection", "clipboard"}
	case config.ClipboardXsel:
		args = []string{"--input", "--clipboard"}
	case config.ClipboardWLClipboard:
		args = []string{}
	}

	cmd := exec.Command(cmdName, args...)

	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for clip write (cmd=%s %v): %w", cmdName, args, err)
	}

	if err = cmd.Start(); err != nil {
		_ = in.Close()
		return fmt.Errorf("failed to start clip command for write (cmd=%s %v): %w", cmdName, args, err)
	}

	// close pipe before reaping subprocess to avoid deadlock
	// waiting for stdin to close (e.g. if in.Write fails mid-way)
	defer func() {
		_ = in.Close()
		_ = cmd.Wait()
	}()

	if _, err = in.Write([]byte(text)); err != nil {
		return fmt.Errorf("failed to write text to clip stdin (length=%d): %w", len(text), err)
	}

	if err = in.Close(); err != nil {
		return fmt.Errorf("failed to close clip stdin pipe: %w", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("clip command failed during write: %w", err)
	}
	return nil
}
