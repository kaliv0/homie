package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	gclip "golang.design/x/clipboard"

	"github.com/kaliv0/homie/internal/config"
)

const (
	binXclip  = "xclip"
	binXsel   = "xsel"
	binWLCopy = "wl-copy"
)

// WriteSelection writes text to the system clipboard.
// On Linux, tool must be one of xclip, xsel, or wl-clipboard (from .homierc).
// On other platforms, tool is ignored and the native clipboard API is used.
func WriteSelection(text, tool string) error {
	if runtime.GOOS == "linux" {
		if tool == "" {
			return fmt.Errorf(
				"clipboard tool not set: add tool: %s, %s, or %s to ~/.homierc",
				config.ClipboardXclip, config.ClipboardXsel, config.ClipboardWLClipboard,
			)
		}
		return writeExternal(text, tool)
	}
	return writeNative(text)
}

func writeExternal(text, tool string) error {
	bin, args, err := toolCommand(tool)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%s not found: %w", tool, err)
	}

	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clip command failed during write: %w", err)
	}
	return nil
}

func writeNative(text string) error {
	if err := gclip.Init(); err != nil {
		return fmt.Errorf("failed to initialize clipboard: %w", err)
	}
	gclip.Write(gclip.FmtText, []byte(text))
	return nil
}

func toolCommand(tool string) (bin string, args []string, err error) {
	switch tool {
	case config.ClipboardXclip:
		return binXclip, []string{"-in", "-selection", "clipboard"}, nil
	case config.ClipboardXsel:
		return binXsel, []string{"--input", "--clipboard"}, nil
	case config.ClipboardWLClipboard:
		return binWLCopy, nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported clipboard tool: %q", tool)
	}
}
