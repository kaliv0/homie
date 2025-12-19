package daemon

import (
	"os"

	"github.com/shirou/gopsutil/process"

	"github.com/kaliv0/homie/internal/log"
)

// StopAllInstances terminates other running homie processes.
func StopAllInstances() error {
	processes, err := process.Processes()
	if err != nil {
		return err
	}
	for _, p := range processes {
		pName, err := p.Name()
		if err != nil {
			continue // Skip inaccessible processes
		}
		if pName == "homie" && int32(os.Getpid()) != p.Pid {
			if err = p.Terminate(); err != nil {
				log.Logger().Println(err) // Log but continue trying other instances
			}
		}
	}
	return nil
}
