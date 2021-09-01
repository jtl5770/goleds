package main

import (
	"fmt"
	"reflect"
	"time"

	c "goleds/controller"
)

var LEDS_TOTAL = 125
var sensor_indices = []int{0, 69, 70, 124}
var controllers = make([]c.LedController, len(sensor_indices))

func main() {
	ledReader := make(chan (*c.LedController), 10)
	ledWriter := make(chan []int, LEDS_TOTAL)
	sensorReader := make(chan int, 10)

	for i := range controllers {
		controllers[i] = c.NewLedController(i, LEDS_TOTAL, sensor_indices[i], ledReader)
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

func updateDisplay(r chan (*c.LedController), w chan ([]int)) {
	var oldSumLeds []int
	var allLedRanges = make([][]int, len(sensor_indices))
	for i := range allLedRanges {
		allLedRanges[i] = make([]int, LEDS_TOTAL)
	}
	for {
		sumLeds := make([]int, LEDS_TOTAL)
		select {
		case s := <-r:
			allLedRanges[s.Name] = s.GetLeds()
		}
		for _, currleds := range allLedRanges {
			for j, v := range currleds {
				if v == 1 {
					sumLeds[j] = 1
				}
			}
		}
		if !reflect.DeepEqual(sumLeds, oldSumLeds) {
			w <- sumLeds
		}
		oldSumLeds = sumLeds
	}
}

func hardwareDriver(display chan ([]int), sensor chan (int)) {
	for {
		select {
		case sumLeds := <-display:
			for _, v := range sumLeds {
				if v == 0 {
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
