package internal

import (
	"log"
	"os"

	"github.com/shirou/gopsutil/process"
)

// Logger is the shared logger used across the application.
var Logger = log.New(os.Stderr, "Homie error: ", log.Llongfile)

// StopAllInstances terminates other running homie processes.
func StopAllInstances() {
	processes, err := process.Processes()
	if err != nil {
		Logger.Fatal(err)
	}
	for _, p := range processes {
		pName, err := p.Name()
		if err != nil {
			continue // Skip inaccessible processes
		}
		if pName == "homie" && int32(os.Getpid()) != p.Pid {
			if err := p.Terminate(); err != nil {
				Logger.Print(err) // Log but continue trying other instances
			}
		}
	}
}
