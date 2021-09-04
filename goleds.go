package main

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"time"

	c "goleds/controller"
	hw "goleds/hardware"
)

const HOLD_T = 5 * time.Second
const RUN_UP_T = 15 * time.Millisecond
const RUN_DOWN_T = 40 * time.Millisecond

func main() {
	controllers := make([]c.LedController, len(hw.Sensors))
	ledReader := make(chan (*c.LedController))
	ledWriter := make(chan []byte, hw.LEDS_TOTAL)
	sensorReader := make(chan int)
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	for i := range controllers {
		controllers[i] = c.NewLedController(i, hw.LEDS_TOTAL, hw.Sensors[i].LedIndex,
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T)
	}

	go updateDisplay(ledReader, ledWriter)
	go fireController(sensorReader, controllers)
	go hw.DisplayDriver(ledWriter)
	go hw.SensorDriver(sensorReader, hw.Sensors)

	<-sigchan
	fmt.Println("\nExiting...")
	os.Exit(0)
}

func updateDisplay(r chan (*c.LedController), w chan ([]byte)) {
	var oldSumLeds []byte
	var allLedRanges = make([][]byte, len(hw.Sensors))
	for i := range allLedRanges {
		allLedRanges[i] = make([]byte, hw.LEDS_TOTAL)
	}
	for {
		s := <-r
		allLedRanges[s.GetUID()] = s.GetLeds()

		sumLeds := make([]byte, hw.LEDS_TOTAL)
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

func fireController(sensor chan (int), controllers []c.LedController) {
	for {
		sensorIndex := <-sensor
		controllers[sensorIndex].Fire()
	}
}

// Local Variables:
// compile-command: "go build"
// End:
