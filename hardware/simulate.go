package hardware

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	p "lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/tui"
)

func scaledColor(led p.Led) string {
	var factor float64
	red := led.Red
	// magic numbers to account for different intensities in led stripe to get a warm white
	green := byte(math.Min(float64(led.Green)*5.7, 255))
	blue := byte(math.Min(float64(led.Blue)*28.3, 255))
	if red >= green && red >= blue {
		// red biggest
		factor = float64(255 / red)
	} else if green >= red && green >= blue {
		// green biggest
		factor = float64(255 / green)
	} else if blue >= red && blue >= green {
		// blue biggest
		factor = float64(255 / blue)
	}
	red = byte(math.Min(float64(red)*factor, 255))
	green = byte(math.Min(float64(green)*factor, 255))
	blue = byte(math.Min(float64(blue)*factor, 255))
	color := fmt.Sprintf("[#%02x%02x%02x]", red, green, blue)
	// log.Printf("%v scaledColor: %s", factor, color)
	return color
}

func simulateLed(segmentID int, values []p.Led) string {
	var buf strings.Builder
	buf.Grow(len(values))
	for _, v := range values {
		if v.IsEmpty() {
			buf.WriteString(" ")
		} else {
			value := byte(math.Round(float64(v.Red+v.Green+v.Blue) / 3.0))
			buf.WriteString(scaledColor(v))
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
			} else if value <= 24 {
				buf.WriteString("▆")
			} else if value <= 32 {
				buf.WriteString("▇")
			} else {
				buf.WriteString("█")
			}
			buf.WriteString("[-]")
		}
	}
	return buf.String()
}

func simulateSensors(sensorReader chan Trigger, sig chan bool) {
	tui.KEYCHAN = make(chan rune)

	for {
		select {
		case <-sig:
			log.Println("Ending SensorDriver simulation go-routine")
			return
		case r := <-tui.KEYCHAN:
			if r == '1' {
				sensorReader <- Trigger{"S0", 80, time.Now()}
			} else if r == '2' {
				sensorReader <- Trigger{"S1", 80, time.Now()}
			} else if r == '3' {
				sensorReader <- Trigger{"S2", 80, time.Now()}
			} else if r == '4' {
				sensorReader <- Trigger{"S3", 80, time.Now()}
			}
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
