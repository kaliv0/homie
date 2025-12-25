package clipboard

import (
	"fmt"
	"os/exec"
)

type xclip struct {
	cmd  string
	args []string
}

var (
	xCopy = xclip{
		cmd:  "xclip",
		args: []string{"-out", "-selection", "clipboard"},
	}
	xPaste = xclip{
		cmd:  "xclip",
		args: []string{"-in", "-selection", "clipboard"},
	}
)

// Read reads whatever is in the clipboard, and returns it as a string.
func Read() (string, error) {
	cmd := exec.Command(xCopy.cmd, xCopy.args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read from clipboard using xclip (cmd=%s %v): %w", xCopy.cmd, xCopy.args, err)
	}
	return string(out), nil
}

// Write writes a given string to the clipboard
func Write(text string) error {
	cmd := exec.Command(xPaste.cmd, xPaste.args...)

	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for xclip write (cmd=%s %v): %w", xPaste.cmd, xPaste.args, err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start xclip command for write (cmd=%s %v): %w", xPaste.cmd, xPaste.args, err)
	}

	_, err = in.Write([]byte(text))
	if err != nil {
		return fmt.Errorf("failed to write text to xclip stdin (length=%d): %w", len(text), err)
	}

	err = in.Close()
	if err != nil {
		return fmt.Errorf("failed to close xclip stdin pipe: %w", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("xclip command failed during write: %w", err)
	}
	return nil
}
