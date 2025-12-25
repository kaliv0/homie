package daemon

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/process"

	"github.com/kaliv0/homie/internal/log"
)

// StopAllInstances terminates other running homie processes.
func StopAllInstances() error {
	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to enumerate processes: %w", err)
	}

	currentPid := int32(os.Getpid())
	for _, p := range processes {
		pName, err := p.Name()
		if err != nil {
			continue // Skip inaccessible processes
		}
		if pName == "homie" && currentPid != p.Pid {
			if err = p.Terminate(); err != nil {
				log.Logger().Printf("failed to terminate homie process (pid=%d): %v\n", p.Pid, err)
			}
		}
	}
	return nil
}
