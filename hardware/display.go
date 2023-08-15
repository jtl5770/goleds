package hardware

import (
	"log"
	"strings"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

const SPLIT_AT = 70

func DisplayDriver(display chan ([]p.Led), sig chan bool) {
	if !c.CONFIG.RealHW {
		go SetupDebugUI()
	}
	for {
		select {
		case <-sig:
			log.Println("Ending DisplayDriver go-routine")
			return
		case sumLeds := <-display:
			led1 := sumLeds[:SPLIT_AT]
			led2 := sumLeds[SPLIT_AT:]
			if !c.CONFIG.RealHW {
				var buf strings.Builder
				buf.WriteString(simulateLed(0, led1))
				buf.WriteString("                  ")
				buf.WriteString(simulateLed(1, led2))
				CONTENT.SetText(buf.String())
			} else {
				spiMutex.Lock()
				setLedSegment(0, led1)
				setLedSegment(1, led2)
				spiMutex.Unlock()
			}
		}
	}
}

func setLedSegment(segmentID int, values []p.Led) {
	display := make([]byte, 3*len(values))
	for idx, led := range values {
		display[3*idx] = led.Red
		display[(3*idx)+1] = led.Green
		display[(3*idx)+2] = led.Blue
	}
	selectLed(segmentID)
	SPIExchange(display)
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
