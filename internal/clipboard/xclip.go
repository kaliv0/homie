package clipboard

import (
	"fmt"
	"os/exec"
)

// Write writes a given string to the clipboard using the specified tool.
func Write(text, tool string) error {
	var cmdName string
	var args []string
	switch tool {
	case "xclip":
		// although tool and cmdName point to the same string value, we keep them separated (loosely coupled)
		cmdName, args = "xclip", []string{"-in", "-selection", "clipboard"}
	case "xsel":
		cmdName, args = "xsel", []string{"--input", "--clipboard"}
	default:
		return fmt.Errorf("unsupported clipboard tool: %q", tool)
	}

	cmd := exec.Command(cmdName, args...)

	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for clip write (cmd=%s %v): %w", cmdName, args, err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start clip command for write (cmd=%s %v): %w", cmdName, args, err)
	}

	_, err = in.Write([]byte(text))
	if err != nil {
		return fmt.Errorf("failed to write text to clip stdin (length=%d): %w", len(text), err)
	}

	err = in.Close()
	if err != nil {
		return fmt.Errorf("failed to close clip stdin pipe: %w", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("clip command failed during write: %w", err)
	}
	return nil
}
