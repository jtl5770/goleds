package hardware

import (
	"log"
	"math"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

const SPLIT_AT = 70

func DisplayDriver(display chan ([]p.Led), sig chan bool) {
	if !c.CONFIG.RealHW {
		SetupDebugUI()
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
				simulateLedDisplay(led1, led2)
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
		display[3*idx] = byte(math.Min(led.Red*c.CONFIG.Hardware.Display.ColorCorrection[0], 255))
		display[(3*idx)+1] = byte(math.Min(led.Green*c.CONFIG.Hardware.Display.ColorCorrection[1], 255))
		display[(3*idx)+2] = byte(math.Min(led.Blue*c.CONFIG.Hardware.Display.ColorCorrection[2], 255))
	}
	selectLed(segmentID)
	SPIExchange(display)
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
