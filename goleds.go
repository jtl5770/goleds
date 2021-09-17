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

var NUMCONTROLLERS = len(hw.Sensors)

func main() {
	controllers := make([]*c.SensorLedController, NUMCONTROLLERS)
	ledReader := make(chan (c.LedProducer))
	ledWriter := make(chan []c.Led, hw.LEDS_TOTAL)
	sensorReader := make(chan int)
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	// NUMCONTROLLERS could be different from hw.Sensors as soon as we
	// implement other types of LedProduces that may not be associated
	// to a sensor. But here we only want to init the LedController
	// types.
	for i := range hw.Sensors {
		controllers[i] = c.NewSensorLedController(i, hw.LEDS_TOTAL, hw.Sensors[i].LedIndex,
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T)
	}
	// *FUTURE* init more controllers as needed

	go combineAndupdateDisplay(ledReader, ledWriter)
	go fireController(sensorReader, controllers)
	go hw.DisplayDriver(ledWriter)
	go hw.SensorDriver(sensorReader, hw.Sensors)

	<-sigchan
	fmt.Println("\nExiting...")
	os.Exit(0)
}

func combineAndupdateDisplay(r chan (c.LedProducer), w chan ([]c.Led)) {
	var oldSumLeds []c.Led
	var allLedRanges = make([][]c.Led, NUMCONTROLLERS)
	for i := range allLedRanges {
		allLedRanges[i] = make([]c.Led, hw.LEDS_TOTAL)
	}
	for {
		s := <-r
		allLedRanges[s.GetUID()] = s.GetLeds()

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

func fireController(sensor chan (int), controllers []*c.SensorLedController) {
	for {
		sensorIndex := <-sensor
		controllers[sensorIndex].Fire()
	}
}

// Local Variables:
// compile-command: "go build"
// End:
