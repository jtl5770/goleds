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
const RUN_UP_T = 10 * time.Millisecond
const RUN_DOWN_T = 50 * time.Millisecond

func main() {
	controllers := make([]c.LedController, len(hw.Sensors))
	ledReader := make(chan (*c.LedController))
	ledWriter := make(chan []byte, hw.LEDS_TOTAL)
	sensorReader := make(chan int)

	for i := range controllers {
		controllers[i] = c.NewLedController(i, hw.LEDS_TOTAL, hw.Sensors[i].LedIndex,
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T)
	}

	go updateDisplay(ledReader, ledWriter)
	go fireController(sensorReader, controllers)
	go hw.DisplayDriver(ledWriter)
	go hw.SensorDriver(sensorReader, hw.Sensors)

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)
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
		allLedRanges[s.Name] = s.GetLeds()

		sumLeds := make([]byte, hw.LEDS_TOTAL)
		for _, currleds := range allLedRanges {
			for j, v := range currleds {
				sumLeds[j] = max(sumLeds[j], v)
			}
		}
		if !reflect.DeepEqual(sumLeds, oldSumLeds) {
			w <- sumLeds
		}
		oldSumLeds = sumLeds
	}
}

func max(x byte, y byte) byte {
	if x > y {
		return x
	} else {
		return y
	}
}

func fireController(sensor chan (int), controllers []c.LedController) {
	for {
		sensorIndex := <-sensor
		controllers[sensorIndex].Fire()
	}
}
