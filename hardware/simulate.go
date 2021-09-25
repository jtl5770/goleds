package hardware

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	c "lautenbacher.net/goleds/producer"
)

func simulateLed(segmentID int, values []c.Led) {
	var buf strings.Builder
	buf.Grow(len(values))

	fmt.Print("[")
	for _, v := range values {
		if v.IsEmpty() {
			buf.WriteString(" ")
		} else if intensity(v) > 50 {
			buf.WriteString("*")
		} else {
			buf.WriteString("_")
		}
	}
	fmt.Print(buf.String())
	if segmentID == 0 {
		fmt.Print("]       ")
	} else {
		fmt.Print("]\r")
	}
}

func intensity(s c.Led) byte {
	return byte(math.Round(float64(s.Red+s.Green+s.Blue) / 3.0))
}

func simulateSensors(sensorReader chan Trigger, sig chan bool) {
	for {
		sensorReader <- Trigger{"_s0", 80, time.Now()}
		if !waitorbreak(12*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s0", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s3", 80, time.Now()}
		if !waitorbreak(20*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s1", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s2", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
	}
}

func waitorbreak(wait time.Duration, sig chan bool) bool {
	select {
	case <-time.After(wait):
		return true
	case <-sig:
		log.Println("Ending SensorDriver go-routine")
		return false
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
