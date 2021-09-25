package hardware

import (
	"log"
	"math"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

const SPLIT_AT = 70

func DisplayDriver(display chan ([]p.Led), sig chan bool) {
	for {
		select {
		case <-sig:
			log.Println("Ending DisplayDriver go-routine")
			return
		case sumLeds := <-display:
			led1 := sumLeds[:SPLIT_AT]
			led2 := sumLeds[SPLIT_AT:]
			if !c.CONFIG.RealHW {
				simulateLed(0, led1)
				simulateLed(1, led2)
			} else {
				setLedSegment(0, led1)
				setLedSegment(1, led2)
			}
		}
	}
}

func setLedSegment(segmentID int, values []p.Led) {
	display := make([]byte, 3*len(values))
	for idx, led := range values {
		display[3*idx] = byte(math.Round(c.CONFIG.Hardware.Display.ColorCorrRed * float64(led.Red)))
		display[(3*idx)+1] = byte(math.Round(c.CONFIG.Hardware.Display.ColorCorrGreen * float64(led.Green)))
		display[(3*idx)+2] = byte(math.Round(c.CONFIG.Hardware.Display.ColorCorrBlue * float64(led.Blue)))
	}
	spiMutex.Lock()
	selectLed(segmentID)
	rpio.SpiExchange(display)
	spiMutex.Unlock()
}

func selectLed(index int) {
	if index == 0 {
		pin17.Low()
		pin22.High()
		pin23.High()
		pin24.High()
	} else if index == 1 {
		pin17.High()
		pin22.Low()
		pin23.High()
		pin24.High()
	} else {
		panic("No LED")
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
