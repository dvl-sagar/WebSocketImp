package processor

import (
	"time"
)

func Process(data any) string {
	time.Sleep(10 * time.Second) // Simulating processing time
	return "Request Processed"
}
