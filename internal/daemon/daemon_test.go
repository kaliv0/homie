package daemon

import (
	"errors"
	"testing"
)

type mockProcess struct {
	name       string
	nameErr    error
	args       []string
	argsErr    error
	pid        int32
	terminated bool
	termErr    error
}

func (m *mockProcess) GetName() (string, error) { return m.name, m.nameErr }
func (m *mockProcess) GetPid() int32             { return m.pid }
func (m *mockProcess) GetCliArgs() ([]string, error) {
	if m.argsErr != nil {
		return nil, m.argsErr
	}
	if m.args != nil {
		return m.args, nil
	}
	return []string{"homie", "run"}, nil
}
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

// assertDaemonStopped runs ProcessDaemons(lister, true) and fails the test on error or ok==false.
func assertDaemonStopped(t *testing.T, lister *mockProcessLister) {
	t.Helper()
	ok, err := ProcessDaemons(lister, true)
	if err != nil {
		t.Fatalf("ProcessDaemons(stop=true) failed: %v", err)
	}
	if !ok {
		t.Fatal("ProcessDaemons(stop=true): expected ok=true, got false")
	}
}

func TestProcessDaemons_Stop_TerminatesOtherHomieRun(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "homie", pid: 200}
	assertDaemonStopped(t, newLister(other))

	if !other.terminated {
		t.Error("expected other homie run process to be terminated")
	}
}

func TestProcessDaemons_Stop_SkipsSelf(t *testing.T) {
	t.Parallel()
	self := &mockProcess{name: "homie", pid: 100}
	assertDaemonStopped(t, newLister(self))

	if self.terminated {
		t.Error("should not terminate own process")
	}
}

func TestProcessDaemons_Stop_SkipsNonHomie(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "firefox", pid: 200}
	assertDaemonStopped(t, newLister(other))

	if other.terminated {
		t.Error("should not terminate non-homie processes")
	}
}

func TestProcessDaemons_Stop_SkipsHomieWithoutRunArg(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "homie", pid: 200, args: []string{"homie", "start"}}
	assertDaemonStopped(t, newLister(other))

	if other.terminated {
		t.Error("should not terminate homie without run subcommand in argv")
	}
}

func TestProcessDaemons_Stop_SkipsHomieShortArgv(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "homie", pid: 200, args: []string{"homie"}}
	assertDaemonStopped(t, newLister(other))

	if other.terminated {
		t.Error("should not terminate homie with argv too short for run check")
	}
}

func TestProcessDaemons_ProcessEnumError(t *testing.T) {
	t.Parallel()
	enumErr := errors.New("permission denied")
	lister := &mockProcessLister{procsErr: enumErr, currentPid: 100}

	_, err := ProcessDaemons(lister, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, enumErr) {
		t.Fatalf("expected wrapped enum error, got %v", err)
	}
}

func TestProcessDaemons_Stop_NameError(t *testing.T) {
	t.Parallel()
	proc := &mockProcess{nameErr: errors.New("access denied"), pid: 200}
	assertDaemonStopped(t, newLister(proc))

	if proc.terminated {
		t.Error("should not terminate process with inaccessible name")
	}
}

func TestProcessDaemons_Stop_CliArgsError(t *testing.T) {
	t.Parallel()
	proc := &mockProcess{name: "homie", pid: 200, argsErr: errors.New("eaccess")}
	assertDaemonStopped(t, newLister(proc))

	if proc.terminated {
		t.Error("should not terminate process when argv cannot be read")
	}
}

func TestProcessDaemons_Stop_NoProcesses(t *testing.T) {
	t.Parallel()
	assertDaemonStopped(t, newLister())
}

func TestProcessDaemons_Stop_MultipleHomieRun(t *testing.T) {
	t.Parallel()
	p1 := &mockProcess{name: "homie", pid: 200}
	p2 := &mockProcess{name: "homie", pid: 300}
	self := &mockProcess{name: "homie", pid: 100}

	assertDaemonStopped(t, newLister(p1, self, p2))

	if !p1.terminated || !p2.terminated {
		t.Error("expected other homie run processes to be terminated")
	}
	if self.terminated {
		t.Error("should not terminate self")
	}
}

func TestProcessDaemons_Stop_TerminateError(t *testing.T) {
	t.Parallel()
	proc := &mockProcess{name: "homie", pid: 200, termErr: errors.New("kill failed")}
	lister := newLister(proc)

	ok, err := ProcessDaemons(lister, true)
	if err == nil {
		t.Fatal("expected terminate error, got nil")
	}
	if ok {
		t.Fatal("expected ok=false on terminate error")
	}
	if !proc.terminated {
		t.Error("terminate should have been attempted")
	}
}

func TestProcessDaemons_Stop_StopsAfterFirstTerminateError(t *testing.T) {
	t.Parallel()
	first := &mockProcess{name: "homie", pid: 200, termErr: errors.New("err1")}
	second := &mockProcess{name: "homie", pid: 300}

	_, err := ProcessDaemons(newLister(first, second), true)
	if err == nil {
		t.Fatal("expected error from first terminate")
	}
	if !first.terminated {
		t.Error("first process should have terminate attempted")
	}
	if second.terminated {
		t.Error("should not attempt second terminate after first error")
	}
}

func TestProcessDaemons_Stop_MixedProcesses(t *testing.T) {
	t.Parallel()
	homie1 := &mockProcess{name: "homie", pid: 200}
	firefox := &mockProcess{name: "firefox", pid: 300}
	homie2 := &mockProcess{name: "homie", pid: 400}
	chrome := &mockProcess{name: "chrome", pid: 500}
	self := &mockProcess{name: "homie", pid: 100}

	assertDaemonStopped(t, newLister(homie1, firefox, self, homie2, chrome))

	if !homie1.terminated || !homie2.terminated {
		t.Error("expected homie run processes to be terminated")
	}
	if firefox.terminated || chrome.terminated {
		t.Error("should not terminate non-homie processes")
	}
	if self.terminated {
		t.Error("should not terminate self")
	}
}

func TestProcessDaemons_Stop_NameMatching(t *testing.T) {
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
			assertDaemonStopped(t, newLister(proc))

			if proc.terminated != tt.wantKilled {
				t.Errorf("process %q: terminated=%v, want %v", tt.procName, proc.terminated, tt.wantKilled)
			}
		})
	}
}

func TestProcessDaemons_Stop_AllProcessesNameError(t *testing.T) {
	t.Parallel()
	procs := []Process{
		&mockProcess{nameErr: errors.New("err1"), pid: 200},
		&mockProcess{nameErr: errors.New("err2"), pid: 300},
		&mockProcess{nameErr: errors.New("err3"), pid: 400},
	}
	assertDaemonStopped(t, newLister(procs...))

	for _, p := range procs {
		if p.(*mockProcess).terminated {
			t.Error("should not terminate any process with name error")
		}
	}
}

func TestProcessDaemons_Stop_OnlySelf(t *testing.T) {
	t.Parallel()
	self := &mockProcess{name: "homie", pid: 100}
	assertDaemonStopped(t, newLister(self))

	if self.terminated {
		t.Error("should not terminate self when it's the only process")
	}
}

func TestProcessDaemons_Check_NoOtherDaemon(t *testing.T) {
	t.Parallel()
	self := &mockProcess{name: "homie", pid: 100}
	ok, err := ProcessDaemons(newLister(self), false)
	if err != nil {
		t.Fatalf("ProcessDaemons(stop=false): %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true when no other homie run exists")
	}
}

func TestProcessDaemons_Check_OtherDaemonRunning(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "homie", pid: 200}
	ok, err := ProcessDaemons(newLister(other), false)
	if err != nil {
		t.Fatalf("ProcessDaemons(stop=false): %v", err)
	}
	if ok {
		t.Fatal("expected ok=false when another homie run is present")
	}
}

func TestProcessDaemons_Check_SkipsNonRunHomie(t *testing.T) {
	t.Parallel()
	other := &mockProcess{name: "homie", pid: 200, args: []string{"homie", "start"}}
	ok, err := ProcessDaemons(newLister(other), false)
	if err != nil {
		t.Fatalf("ProcessDaemons(stop=false): %v", err)
	}
	if !ok {
		t.Fatal("homie without run should not block check")
	}
}

func TestProcessDaemons_Check_FirstConflictShortCircuits(t *testing.T) {
	t.Parallel()
	a := &mockProcess{name: "homie", pid: 200}
	b := &mockProcess{name: "homie", pid: 300}
	ok, err := ProcessDaemons(newLister(a, b), false)
	if err != nil {
		t.Fatalf("ProcessDaemons(stop=false): %v", err)
	}
	if ok {
		t.Fatal("expected conflict when multiple other daemons")
	}
}
