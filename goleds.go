package main

import (
	"fmt"
	"time"

	lc "goleds/controller"
)

const LEDS_TOTAL = 125
const SENSOR_TOTAL = 4

var controllers [SENSOR_TOTAL]lc.LedController

func main() {
	sensor_indices := [SENSOR_TOTAL]int{0, 69, 70, 124}
	ledReader := make(chan (*lc.LedController), 10)
	ledWriter := make(chan [LEDS_TOTAL]int)
	sensorReader := make(chan int, 10)

	for i := 0; i < SENSOR_TOTAL; i++ {
		controllers[i] = lc.NewLedController(i, sensor_indices[i], ledReader)
	}

	go updateDisplay(ledReader, ledWriter)
	go hardwareDriver(ledWriter, sensorReader)

	sensorReader <- 3
	sensorReader <- 0
	time.Sleep(9 * time.Second)
	sensorReader <- 1
	time.Sleep(7 * time.Second)
	sensorReader <- 1
	time.Sleep(2 * time.Second)
	sensorReader <- 0

	for {
		time.Sleep(24 * time.Hour)
	}
}

func updateDisplay(r chan (*lc.LedController), w chan ([LEDS_TOTAL]int)) {
	var oldSumLeds [LEDS_TOTAL]int
	var leds [SENSOR_TOTAL][LEDS_TOTAL]int
	for {
		var sumLeds [LEDS_TOTAL]int
		select {
		case s := <-r:
			leds[s.Name] = s.GetLeds()
		}
		for i := 0; i < SENSOR_TOTAL; i++ {
			currleds := leds[i]
			for j := 0; j < LEDS_TOTAL; j++ {
				if currleds[j] == 1 {
					sumLeds[j] = 1
				}
			}
		}
		if sumLeds != oldSumLeds {
			w <- sumLeds
		}
		oldSumLeds = sumLeds
	}
}

func hardwareDriver(display chan ([LEDS_TOTAL]int), sensor chan (int)) {
	for {
		select {
		case sumLeds := <-display:
			for i := 0; i < LEDS_TOTAL; i++ {
				if sumLeds[i] == 0 {
					fmt.Print(" ")
				} else {
					fmt.Print("☼")
				}
			}
			fmt.Print("\r")
		case sensorIndex := <-sensor:
			controllers[sensorIndex].Fire()
		}
	}
}
