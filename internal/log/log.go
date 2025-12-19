package log

import (
	"log"
	"os"
	"sync"
)

// Logger is the shared log used across the application.
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
