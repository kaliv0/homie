package clipboard

import "os/exec"

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
		return "", err
	}
	return string(out), nil
}

// Write writes a given string to the clipboard
func Write(text string) error {
	cmd := exec.Command(xPaste.cmd, xPaste.args...)

	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	_, err = in.Write([]byte(text))
	if err != nil {
		return err
	}

	err = in.Close()
	if err != nil {
		return err
	}

	return cmd.Wait()
}
