package main

import (
	"fmt"
	"log"
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
	ledproducers := make(map[string]c.LedProducer)
	ledReader := make(chan (c.LedProducer))
	ledWriter := make(chan []c.Led, hw.LEDS_TOTAL)
	sensorReader := make(chan string)
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	for uid := range hw.Sensors {
		ledproducers[uid] = c.NewSensorLedProducer(uid, hw.LEDS_TOTAL, hw.Sensors[uid].LedIndex,
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T)
	}
	// *FUTURE* init more types of ledproducers if needed/wanted

	go combineAndupdateDisplay(ledReader, ledWriter)
	go fireController(sensorReader, ledproducers)
	go hw.DisplayDriver(ledWriter)
	go hw.SensorDriver(sensorReader, hw.Sensors)

	<-sigchan
	fmt.Println("\nExiting...")
	os.Exit(0)
}

func combineAndupdateDisplay(r chan (c.LedProducer), w chan ([]c.Led)) {
	var oldSumLeds []c.Led
	var allLedRanges = make(map[string][]c.Led)
	for uid := range hw.Sensors {
		allLedRanges[uid] = make([]c.Led, hw.LEDS_TOTAL)
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

func fireController(sensor chan (string), producers map[string]c.LedProducer) {
	for {
		sensorUid := <-sensor
		if producer, ok := producers[sensorUid]; ok {
			producer.Fire()
		} else {
			log.Printf("Unknown UID %s", sensorUid)
		}
	}
}

// Local Variables:
// compile-command: "go build"
// End:
