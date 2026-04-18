package daemon

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/process"
)

// Process represents a system process.
type Process interface {
	GetName() (string, error)
	GetPid() int32
	GetCliArgs() ([]string, error)
	Terminate() error
}

// ProcessLister enumerates running processes.
type ProcessLister interface {
	Processes() ([]Process, error)
	CurrentPid() int32
}

// osProcess wraps a gopsutil process to satisfy the Process interface.
type osProcess struct {
	p *process.Process
}

func (o osProcess) GetName() (string, error)      { return o.p.Name() }
func (o osProcess) GetPid() int32                 { return o.p.Pid }
func (o osProcess) GetCliArgs() ([]string, error) { return o.p.CmdlineSlice() }
func (o osProcess) Terminate() error              { return o.p.Terminate() }

// osProcessLister uses gopsutil to enumerate system processes.
type osProcessLister struct{}

func (o osProcessLister) Processes() ([]Process, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}
	result := make([]Process, len(procs))
	for i, p := range procs {
		result[i] = osProcess{p: p}
	}
	return result, nil
}

func (o osProcessLister) CurrentPid() int32 {
	return int32(os.Getpid())
}

// CheckAll returns true when no other homie "run" process exists (self excluded).
func CheckAll() (bool, error) {
	return ProcessDaemons(osProcessLister{}, false)
}

// StopAll terminates other homie "run" processes only.
func StopAll() (bool, error) {
	return ProcessDaemons(osProcessLister{}, true)
}

// ProcessDaemons finds homie processes with argv[1]=="run", excluding pl.CurrentPid().
// If stop is false, returns false when any match exists; if true, terminates matches.
func ProcessDaemons(pl ProcessLister, stop bool) (bool, error) {
	processes, err := pl.Processes()
	if err != nil {
		return false, fmt.Errorf("failed to enumerate processes: %w", err)
	}

	currentPid := pl.CurrentPid()
	for _, p := range processes {
		pName, err := p.GetName()
		if err != nil {
			continue // Skip inaccessible processes
		}

		if pName == "homie" {
			args, err := p.GetCliArgs()
			if err != nil {
				continue
			}
			if len(args) > 1 && args[1] == "run" && currentPid != p.GetPid() {
				if !stop {
					return false, nil
				}
				if err = p.Terminate(); err != nil {
					return false, fmt.Errorf("failed to terminate homie process (pid=%d): %v", p.GetPid(), err)
				}
			}
		}
	}

	return true, nil
}
