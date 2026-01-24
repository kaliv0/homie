package daemon

import (
	"errors"
	"testing"
)

type mockProcess struct {
	name       string
	nameErr    error
	pid        int32
	terminated bool
	termErr    error
}

func (m *mockProcess) Name() (string, error) { return m.name, m.nameErr }
func (m *mockProcess) GetPid() int32         { return m.pid }
func (m *mockProcess) Terminate() error {
	m.terminated = true
	return m.termErr
}

type mockProcessLister struct {
	procs      []Process
	procsErr   error
	currentPid int32
}

func (m *mockProcessLister) Processes() ([]Process, error) {
	return m.procs, m.procsErr
}

func (m *mockProcessLister) CurrentPid() int32 {
	return m.currentPid
}

// newLister creates a mockProcessLister with currentPid=100 and the given processes.
func newLister(procs ...Process) *mockProcessLister {
	return &mockProcessLister{procs: procs, currentPid: 100}
}

// runStop calls StopAllInstances with the given lister and asserts no error.
func runStop(t *testing.T, lister *mockProcessLister) {
	t.Helper()
	if err := StopAllInstances(lister); err != nil {
		t.Fatalf("StopAllInstances() failed: %v", err)
	}
}

func TestStopAllInstances_TerminatesOtherHomie(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "homie", pid: 200}
	runStop(t, newLister(other))

	if !other.terminated {
		t.Error("expected other homie process to be terminated")
	}
}

func TestStopAllInstances_SkipsSelf(t *testing.T) {
	t.Parallel()
	self := &mockProcess{name: "homie", pid: 100}
	runStop(t, newLister(self))

	if self.terminated {
		t.Error("should not terminate own process")
	}
}

func TestStopAllInstances_SkipsNonHomie(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "firefox", pid: 200}
	runStop(t, newLister(other))

	if other.terminated {
		t.Error("should not terminate non-homie processes")
	}
}

func TestStopAllInstances_ProcessEnumError(t *testing.T) {
	t.Parallel()
	enumErr := errors.New("permission denied")
	lister := &mockProcessLister{procsErr: enumErr, currentPid: 100}

	err := StopAllInstances(lister)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, enumErr) {
		t.Fatalf("expected wrapped enum error, got %v", err)
	}
}

func TestStopAllInstances_NameError(t *testing.T) {
	t.Parallel()
	proc := &mockProcess{nameErr: errors.New("access denied"), pid: 200}
	runStop(t, newLister(proc))

	if proc.terminated {
		t.Error("should not terminate process with inaccessible name")
	}
}

func TestStopAllInstances_NoProcesses(t *testing.T) {
	t.Parallel()
	runStop(t, newLister())
}

func TestStopAllInstances_MultipleHomie(t *testing.T) {
	t.Parallel()
	p1 := &mockProcess{name: "homie", pid: 200}
	p2 := &mockProcess{name: "homie", pid: 300}
	self := &mockProcess{name: "homie", pid: 100}

	runStop(t, newLister(p1, self, p2))

	if !p1.terminated || !p2.terminated {
		t.Error("expected other homie processes to be terminated")
	}
	if self.terminated {
		t.Error("should not terminate self")
	}
}

func TestStopAllInstances_TerminateErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		procs []*mockProcess
	}{
		{
			"single terminate error",
			[]*mockProcess{{name: "homie", pid: 200, termErr: errors.New("err")}},
		},
		{
			"multiple terminate errors",
			[]*mockProcess{
				{name: "homie", pid: 200, termErr: errors.New("err1")},
				{name: "homie", pid: 300, termErr: errors.New("err2")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			procs := make([]Process, len(tt.procs))
			for i, p := range tt.procs {
				procs[i] = p
			}
			runStop(t, newLister(procs...))

			for _, p := range tt.procs {
				if !p.terminated {
					t.Errorf("terminate should be attempted on pid=%d", p.pid)
				}
			}
		})
	}
}

func TestStopAllInstances_MixedProcesses(t *testing.T) {
	t.Parallel()
	homie1 := &mockProcess{name: "homie", pid: 200}
	firefox := &mockProcess{name: "firefox", pid: 300}
	homie2 := &mockProcess{name: "homie", pid: 400}
	chrome := &mockProcess{name: "chrome", pid: 500}
	self := &mockProcess{name: "homie", pid: 100}

	runStop(t, newLister(homie1, firefox, self, homie2, chrome))

	if !homie1.terminated || !homie2.terminated {
		t.Error("expected homie processes to be terminated")
	}
	if firefox.terminated || chrome.terminated {
		t.Error("should not terminate non-homie processes")
	}
	if self.terminated {
		t.Error("should not terminate self")
	}
}

func TestStopAllInstances_NameMatching(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		procName   string
		wantKilled bool
	}{
		{"exact match", "homie", true},
		{"prefix", "homie-helper", false},
		{"suffix", "myhomie", false},
		{"extended", "homiex", false},
		{"case different", "HOMIE", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proc := &mockProcess{name: tt.procName, pid: 200}
			runStop(t, newLister(proc))

			if proc.terminated != tt.wantKilled {
				t.Errorf("process %q: terminated=%v, want %v", tt.procName, proc.terminated, tt.wantKilled)
			}
		})
	}
}

func TestStopAllInstances_AllProcessesNameError(t *testing.T) {
	t.Parallel()
	procs := []Process{
		&mockProcess{nameErr: errors.New("err1"), pid: 200},
		&mockProcess{nameErr: errors.New("err2"), pid: 300},
		&mockProcess{nameErr: errors.New("err3"), pid: 400},
	}
	runStop(t, newLister(procs...))

	for _, p := range procs {
		if p.(*mockProcess).terminated {
			t.Error("should not terminate any process with name error")
		}
	}
}

func TestStopAllInstances_OnlySelf(t *testing.T) {
	t.Parallel()
	self := &mockProcess{name: "homie", pid: 100}
	runStop(t, newLister(self))

	if self.terminated {
		t.Error("should not terminate self when it's the only process")
	}
}
