package hardware

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	p "lautenbacher.net/goleds/producer"
)

func simulateLed(segmentID int, values []p.Led) {
	var buf strings.Builder
	buf.Grow(len(values))

	fmt.Print("[")
	for _, v := range values {
		if v.IsEmpty() {
			buf.WriteString(" ")
		} else {
			value := intensity(v)
			if value == 1 {
				buf.WriteString("▁")
			} else if value == 2 {
				buf.WriteString("▂")
			} else if value <= 4 {
				buf.WriteString("▃")
			} else if value <= 8 {
				buf.WriteString("▄")
			} else if value <= 16 {
				buf.WriteString("▅")
			} else if value <= 32 {
				buf.WriteString("▆")
			} else if value <= 64 {
				buf.WriteString("▇")
			} else {
				buf.WriteString("█")
			}
		}
	}
	fmt.Print(buf.String())
	if segmentID == 0 {
		fmt.Print("]       ")
	} else {
		fmt.Print("]\r")
	}
}

func intensity(s p.Led) byte {
	return byte(math.Round(float64(s.Red+s.Green+s.Blue) / 3.0))
}

func simulateSensors(sensorReader chan Trigger, sig chan bool) {
	for {
		sensorReader <- Trigger{"S0", 80, time.Now()}
		if !waitorbreak(12*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"S0", 80, time.Now()}
		if !waitorbreak(30*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"S3", 80, time.Now()}
		if !waitorbreak(20*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"S1", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"S2", 80, time.Now()}
		if !waitorbreak(30*time.Second, sig) {
			return
		}
	}
}

func waitorbreak(wait time.Duration, sig chan bool) bool {
	select {
	case <-time.After(wait):
		return true
	case <-sig:
		log.Println("Ending SensorDriver simulation go-routine")
		return false
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
