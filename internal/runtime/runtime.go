package runtime

import (
	"log"
	"os"
	"sync"

	"github.com/shirou/gopsutil/process"
)

// Logger is the shared logger used across the application.
var Logger = sync.OnceValue(getLogger)

func getLogger() *log.Logger {
	l := log.New(os.Stderr, "ERROR: ", log.Llongfile)

	// Add log config below:

	//logFile, err := os.OpenFile("homie.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	//if err != nil {
	//	fmt.Println(err)
	//	return l
	//}
	//log.SetOutput(logFile)

	return l
}

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
				Logger().Println(err) // Log but continue trying other instances
			}
		}
	}
	return nil
}
