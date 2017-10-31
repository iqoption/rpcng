package log

import (
	"log"

	"github.com/iqoption/rpcng/logger"
)

// default logger struct
type l struct{}

// Default logger constructor
func New() logger.Logger {
	return &l{}
}

// Debug message
func (l *l) Debug(message string, vars ...interface{}) {
	var arguments = append([]interface{}{"RPCNG DEBUG:", message}, vars...)
	log.Println(arguments...)
}

// Info message
func (l *l) Info(message string, vars ...interface{}) {
	var arguments = append([]interface{}{"RPCNG INFO:", message}, vars...)
	log.Println(arguments...)
}

// Error message
func (l *l) Error(message string, vars ...interface{}) {
	var arguments = append([]interface{}{"RPCNG ERROR:", message}, vars...)
	log.Println(arguments...)
}
