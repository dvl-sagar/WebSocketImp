package processor

import (
	"time"
)

func Process(data any) string {
	time.Sleep(8 * time.Second) // Simulating processing time
	return "Request Processed"
}
