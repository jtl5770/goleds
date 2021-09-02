package main

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"time"

	c "goleds/controller"
)

const LEDS_TOTAL = 125
const HOLD_T = 5 * time.Second
const RUN_UP_T = 10 * time.Millisecond
const RUN_DOWN_T = 40 * time.Millisecond

var sensor_indices = []int{0, 69, 70, 124}

func main() {
	controllers := make([]c.LedController, len(sensor_indices))
	ledReader := make(chan (*c.LedController), 10)
	ledWriter := make(chan []byte, LEDS_TOTAL)
	sensorReader := make(chan int, 10)

	for i := range controllers {
		controllers[i] = c.NewLedController(i, LEDS_TOTAL, sensor_indices[i],
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T)
	}

	go updateDisplay(ledReader, ledWriter)
	go fireController(sensorReader, controllers)
	go displayDriver(ledWriter)
	go sensorDriver(sensorReader, sensor_indices)

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	fmt.Println("\nExiting...")
	os.Exit(0)
}

func updateDisplay(r chan (*c.LedController), w chan ([]byte)) {
	var oldSumLeds []byte
	var allLedRanges = make([][]byte, len(sensor_indices))
	for i := range allLedRanges {
		allLedRanges[i] = make([]byte, LEDS_TOTAL)
	}
	for {
		s := <-r
		allLedRanges[s.Name] = s.GetLeds()

		sumLeds := make([]byte, LEDS_TOTAL)
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

// ****
// TODO: real hardware implementation
// ****
func displayDriver(display chan ([]byte)) {
	for {
		sumLeds := <-display
		for _, v := range sumLeds {
			if v == 0 {
				fmt.Print(" ")
			} else {
				fmt.Print("☼")
			}
		}
		fmt.Print("\r")
	}
}

// ****
// TODO: real hardware implementation
// ****
func sensorDriver(sensorReader chan int, sensorIndices []int) {
	sensorReader <- 3
	sensorReader <- 0
	time.Sleep(9 * time.Second)
	sensorReader <- 1
	time.Sleep(7 * time.Second)
	sensorReader <- 1
	time.Sleep(2 * time.Second)
	sensorReader <- 0
}
