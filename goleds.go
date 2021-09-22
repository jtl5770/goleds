package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"time"

	hw "lautenbacher.net/goleds/hardware"
	c "lautenbacher.net/goleds/producer"
)

const HOLD_T = 10 * time.Second
const RUN_UP_T = 5 * time.Millisecond
const RUN_DOWN_T = 30 * time.Millisecond

// how bright the SensorLedProducer makes the LEDs when on (will be
// used for all three compnents red, green, blue)
const LED_ON_SLP = 80

const LED_ON_HOLD = 200
const HOLD_LED_UID = "hold_led"
const FULL_HIGH_HOLD = 5 * time.Minute
const HOLD_TRIGGER_DELAY = 5 * time.Second
const HOLD_TRIGGER_VALUE = 160

// Karlsruhe
const LAT = 49.014
const LONG = 8.4043

var NIGHT_LED = c.Led{Red: 3, Green: 0, Blue: 0}

func main() {
	ledproducers := make(map[string]c.LedProducer)
	ledReader := make(chan (c.LedProducer))
	ledWriter := make(chan []c.Led, hw.LEDS_TOTAL)
	sensorReader := make(chan hw.Trigger)
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	for uid := range hw.Sensors {
		ledproducers[uid] = c.NewSensorLedProducer(uid, hw.LEDS_TOTAL, hw.Sensors[uid].LedIndex,
			ledReader, HOLD_T, RUN_UP_T, RUN_DOWN_T, c.Led{Red: LED_ON_SLP, Green: LED_ON_SLP, Blue: LED_ON_SLP})
	}
	// The Nightlight producer makes a permanent red glow (by default) during night time
	prodnight := c.NewNightlightProducter("night_led", hw.LEDS_TOTAL, ledReader, NIGHT_LED, LAT, LONG)
	ledproducers["night_led"] = prodnight
	prodnight.Fire()

	prodhold := c.NewHoldProducer(HOLD_LED_UID, hw.LEDS_TOTAL, ledReader,
		c.Led{Red: LED_ON_HOLD, Green: LED_ON_HOLD, Blue: LED_ON_HOLD}, FULL_HIGH_HOLD)
	ledproducers[HOLD_LED_UID] = prodhold

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
				sumLeds[j] = v.Max(sumLeds[j])
			}
		}
		if !reflect.DeepEqual(sumLeds, oldSumLeds) {
			w <- sumLeds
		}
		oldSumLeds = sumLeds
	}
}

func fireController(sensor chan (hw.Trigger), producers map[string]c.LedProducer) {
	var firstSameTrigger hw.Trigger
	for {
		trigger := <-sensor
		oldStamp := firstSameTrigger.Timestamp
		newStamp := trigger.Timestamp

		if trigger.Value >= HOLD_TRIGGER_VALUE {
			if trigger.ID != firstSameTrigger.ID {
				firstSameTrigger = trigger
			} else if newStamp.Sub(oldStamp) > HOLD_TRIGGER_DELAY {
				firstSameTrigger = hw.Trigger{}
				// Don't want to compare against too old timestamps
				if newStamp.Sub(oldStamp) < (HOLD_TRIGGER_DELAY + (1 * time.Second)) {
					producers[HOLD_LED_UID].Fire()
				}
				break
			}
		} else {
			firstSameTrigger = hw.Trigger{}
		}

		if producer, ok := producers[trigger.ID]; ok {
			producer.Fire()
		} else {
			log.Printf("Unknown UID %s", trigger.ID)
		}
	}
}

// Local Variables:
// compile-command: "go build"
// End:
