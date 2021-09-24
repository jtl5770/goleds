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

const HOLD_LED_UID = "hold_led"
const HOLD_TRIGGER_VALUE = 160

const FORCED_UPDATE_INTERVAL = 5 * time.Second

func main() {
	c.ReadConfig()
	hw.InitSensors()
	ledproducers := make(map[string]c.LedProducer)
	ledReader := make(chan (c.LedProducer))
	ledWriter := make(chan []c.Led, hw.LEDS_TOTAL)
	sensorReader := make(chan hw.Trigger)
	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	for uid := range hw.Sensors {
		ledproducers[uid] = c.NewSensorLedProducer(uid, hw.LEDS_TOTAL, hw.Sensors[uid].LedIndex, ledReader)
	}
	// The Nightlight producer makes a permanent red glow (by default) during night time
	prodnight := c.NewNightlightProducter("night_led", hw.LEDS_TOTAL, ledReader)
	ledproducers["night_led"] = prodnight
	prodnight.Fire()

	// The HoldLight producer will be fired whenever a sensor produces for HOLD_TRIGGER_DELAY a signal > HOLD_TRIGGER_VALUE
	// It will generate a brighter, full lit LED stripe and keep it for FULL_HIGH_HOLD time, if not being triggered again
	// in this time - then it will shut off earlier
	prodhold := c.NewHoldProducer(HOLD_LED_UID, hw.LEDS_TOTAL, ledReader)
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
	allLedRanges := make(map[string][]c.Led)
	ticker := time.NewTicker(FORCED_UPDATE_INTERVAL)
	for uid := range hw.Sensors {
		allLedRanges[uid] = make([]c.Led, hw.LEDS_TOTAL)
	}
	for {
		select {
		case s := <-r:
			allLedRanges[s.GetUID()] = s.GetLeds()
			sumLeds := combineLeds(allLedRanges)
			if !reflect.DeepEqual(sumLeds, oldSumLeds) {
				w <- sumLeds
			}
			oldSumLeds = sumLeds
		case <-ticker.C:
			// We do this purely because there occasionally come
			// artifacts from - maybe/somehow - electrical distortions
			// So we make sure to regularily update the Led stripe
			w <- combineLeds(allLedRanges)
		}
	}
}

func combineLeds(allLedRanges map[string][]c.Led) []c.Led {
	sumLeds := make([]c.Led, hw.LEDS_TOTAL)
	for _, currleds := range allLedRanges {
		for j, v := range currleds {
			sumLeds[j] = v.Max(sumLeds[j])
		}
	}
	return sumLeds
}

func fireController(sensor chan (hw.Trigger), producers map[string]c.LedProducer) {
	var firstSameTrigger hw.Trigger
	var triggerDelay = c.CONFIG.HoldLED.TriggerSeconds * time.Second
	for {
		trigger := <-sensor
		oldStamp := firstSameTrigger.Timestamp
		newStamp := trigger.Timestamp

		if trigger.Value >= HOLD_TRIGGER_VALUE {
			if trigger.ID != firstSameTrigger.ID {
				firstSameTrigger = trigger
			} else if newStamp.Sub(oldStamp) > triggerDelay {
				firstSameTrigger = hw.Trigger{}
				// Don't want to compare against too old timestamps
				if newStamp.Sub(oldStamp) < (triggerDelay + (1 * time.Second)) {
					producers[HOLD_LED_UID].Fire()
				}
			}
		} else {
			firstSameTrigger = hw.Trigger{}
			if producer, ok := producers[trigger.ID]; ok {
				producer.Fire()
			} else {
				log.Printf("Unknown UID %s", trigger.ID)
			}
		}
	}
}

// Local Variables:
// compile-command: "go build"
// End:
