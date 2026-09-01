package clipboard

import (
	"runtime"
	"strings"
	"testing"
)

func TestToolCommand_unknownTool(t *testing.T) {
	_, _, err := toolCommand("pbcopy")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported clipboard tool") {
		t.Fatalf("error = %q, want unsupported clipboard tool", err.Error())
	}
}

func TestWriteSelection_emptyToolOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	err := WriteSelection("hello", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "clipboard tool not set") {
		t.Fatalf("error = %q, want clipboard tool not set", err.Error())
	}
}
