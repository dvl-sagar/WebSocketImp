package processor

import (
	"time"
)

func Process(data interface{}) string {
	time.Sleep(3 * time.Second) // Simulating processing time
	return "Processed Request"
}
