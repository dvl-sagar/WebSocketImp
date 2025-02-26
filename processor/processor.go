package processor

import (
	"fmt"
	"math/rand"
	"time"
)

func Process(data any) string {
	time.Sleep(8 * time.Second)
	num := rand.Int() // Simulating processing time
	return fmt.Sprintf("Request Processed %d", num)
}
