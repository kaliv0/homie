package daemon

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/process"

	"github.com/kaliv0/homie/internal/log"
)

// Process represents a system process.
type Process interface {
	Name() (string, error)
	GetPid() int32
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

func (o osProcess) Name() (string, error) { return o.p.Name() }
func (o osProcess) GetPid() int32         { return o.p.Pid }
func (o osProcess) Terminate() error      { return o.p.Terminate() }

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

// StopAll terminates other running homie processes using the system process list.
func StopAll() error {
	return StopAllInstances(osProcessLister{})
}

// StopAllInstances terminates other running homie processes.
func StopAllInstances(pl ProcessLister) error {
	processes, err := pl.Processes()
	if err != nil {
		return fmt.Errorf("failed to enumerate processes: %w", err)
	}

	currentPid := pl.CurrentPid()
	for _, p := range processes {
		pName, err := p.Name()
		if err != nil {
			continue // Skip inaccessible processes
		}
		if pName == "homie" && currentPid != p.GetPid() {
			if err = p.Terminate(); err != nil {
				log.Logger().Printf("failed to terminate homie process (pid=%d): %v\n", p.GetPid(), err)
			}
		}
	}
	return nil
}
