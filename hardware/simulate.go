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

// magic numbers to account for different intensities of color
// components in led stripe to get a warm white. Needed because
// terminal output doesn't have such a huge color cast
var (
	magic_factor_green float64 = 5.7
	magic_factor_blue  float64 = 28.3
)

func scaledColor(led p.Led) string {
	var factor float64
	red := float64(led.Red)
	green := math.Min(float64(led.Green)*magic_factor_green, 255)
	blue := math.Min(float64(led.Blue)*magic_factor_blue, 255)

	factor = float64(255 / math.Max(red, math.Max(green, blue)))
	red = math.Min(red*factor, 255)
	green = math.Min(green*factor, 255)
	blue = math.Min(blue*factor, 255)
	color := fmt.Sprintf("[#%02x%02x%02x]", byte(red), byte(green), byte(blue))
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
