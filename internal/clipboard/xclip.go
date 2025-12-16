package clipboard

import "os/exec"

// Read reads whatever is in the clipboard, and returns it as a string.
func Read() (string, error) {
	cmd := exec.Command("xclip", "-out", "-selection", "clipboard")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// WriteText writes a given string to the clipboard
func WriteText(text string) error {
	cmd := exec.Command("xclip", "-in", "-selection", "clipboard")

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
