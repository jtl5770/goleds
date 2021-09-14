package main

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"time"

	c "lautenbacher.net/goleds/controller"
	hw "lautenbacher.net/goleds/hardware"
)

const HOLD_T = 5 * time.Second
const RUN_UP_T = 5 * time.Millisecond
const RUN_DOWN_T = 50 * time.Millisecond

func main() {
	controllers := make([]*c.LedController, len(hw.Sensors))
	ledReader := make(chan (*c.LedController))
	ledWriter := make(chan []c.Led, hw.LEDS_TOTAL)
	sensorReader := make(chan int)
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	for i := range controllers {
		controllers[i] = c.NewLedController(i, hw.LEDS_TOTAL, hw.Sensors[i].LedIndex,
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T)
	}

	go combineAndupdateDisplay(ledReader, ledWriter)
	go fireController(sensorReader, controllers)
	go hw.DisplayDriver(ledWriter)
	go hw.SensorDriver(sensorReader, hw.Sensors)

	<-sigchan
	fmt.Println("\nExiting...")
	os.Exit(0)
}

func combineAndupdateDisplay(r chan (*c.LedController), w chan ([]c.Led)) {
	var oldSumLeds []c.Led
	var allLedRanges = make([][]c.Led, len(hw.Sensors))
	for i := range allLedRanges {
		allLedRanges[i] = make([]c.Led, hw.LEDS_TOTAL)
	}
	for {
		s := <-r
		allLedRanges[s.UID] = s.GetLeds()

		sumLeds := make([]c.Led, hw.LEDS_TOTAL)
		for _, currleds := range allLedRanges {
			for j, v := range currleds {
				if v > sumLeds[j] {
					sumLeds[j] = v
				}
			}
		}
		if !reflect.DeepEqual(sumLeds, oldSumLeds) {
			w <- sumLeds
		}
		oldSumLeds = sumLeds
	}
}

func fireController(sensor chan (int), controllers []*c.LedController) {
	for {
		sensorIndex := <-sensor
		controllers[sensorIndex].Fire()
	}
}

// Local Variables:
// compile-command: "go build"
// End:
