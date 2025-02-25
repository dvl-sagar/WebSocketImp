package processor

import (
	"fmt"
	"time"
)

func Process(data string) string {
	time.Sleep(2 * time.Second) // Simulating processing time
	return fmt.Sprintf("Processed: %s", data)
}
