package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	hw "lautenbacher.net/goleds/hardware"
	c "lautenbacher.net/goleds/producer"
)

const HOLD_LED_UID = "hold_led"
const HOLD_TRIGGER_VALUE = 160
const FORCED_UPDATE_INTERVAL = 5 * time.Second

var ledproducers map[string]c.LedProducer
var sigchans [](chan bool)

func Initialise(cfile string, realhw bool) {
	c.ReadConfig(cfile, realhw)
	log.Println(c.CONFIG)

	hw.InitGpioAndSensors()
	ledproducers = make(map[string]c.LedProducer)
	sigchans = make([](chan bool), 0)
	ledReader := make(chan (c.LedProducer))
	ledWriter := make(chan []c.Led, hw.LEDS_TOTAL)
	sensorReader := make(chan hw.Trigger)

	if c.CONFIG.SensorLED.Enabled {
		for uid := range hw.Sensors {
			ledproducers[uid] = c.NewSensorLedProducer(uid, hw.LEDS_TOTAL, hw.Sensors[uid].LedIndex, ledReader)
		}
	}
	if c.CONFIG.NightLED.Enabled {
		// The Nightlight producer makes a permanent red glow (by default) during night time
		prodnight := c.NewNightlightProducer("night_led", hw.LEDS_TOTAL, ledReader)
		ledproducers["night_led"] = prodnight
		prodnight.Fire()
	}

	if c.CONFIG.HoldLED.Enabled {
		// The HoldLight producer will be fired whenever a sensor produces for HOLD_TRIGGER_DELAY a signal > HOLD_TRIGGER_VALUE
		// It will generate a brighter, full lit LED stripe and keep it for FULL_HIGH_HOLD time, if not being triggered again
		// in this time - then it will shut off earlier
		prodhold := c.NewHoldProducer(HOLD_LED_UID, hw.LEDS_TOTAL, ledReader)
		ledproducers[HOLD_LED_UID] = prodhold
	}

	// *FUTURE* init more types of ledproducers if needed/wanted

	cADsignal := make(chan bool)
	fCsignal := make(chan bool)
	DDsignal := make(chan bool)
	SDsignal := make(chan bool)
	sigchans = append(sigchans, cADsignal, fCsignal, DDsignal, SDsignal)
	go combineAndupdateDisplay(ledReader, ledWriter, cADsignal)
	go fireController(sensorReader, ledproducers, fCsignal)
	go hw.DisplayDriver(ledWriter, DDsignal)
	go hw.SensorDriver(sensorReader, hw.Sensors, SDsignal)
}

func ResetAll() {
	for _, prod := range ledproducers {
		prod.Stop()
	}
	time.Sleep(2 * time.Second)
	for _, sig := range sigchans {
		sig <- true
	}
	time.Sleep(2 * time.Second)
}

func main() {
	cfile := flag.String("config", c.CONFILE, "Config file to use")
	realp := flag.Bool("real", false, "Set to true if program runs on real hardware")
	flag.Parse()
	osSig := make(chan os.Signal)
	reload := make(chan os.Signal)
	signal.Notify(osSig, os.Interrupt)
	signal.Notify(reload, syscall.SIGHUP)

	Initialise(*cfile, *realp)

	for {
		select {
		case <-osSig:
			fmt.Println("\nExiting...")
			os.Exit(0)
		case <-reload:
			fmt.Println("\nResetting...")
			ResetAll()
			fmt.Println("\nInitialising...")
			Initialise(*cfile, *realp)
		}
	}
}

func combineAndupdateDisplay(r chan (c.LedProducer), w chan ([]c.Led), sig chan bool) {
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
		case <-sig:
			log.Println("Ending combineAndupdateDisplay go-routine")
			ticker.Stop()
			return
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

func fireController(sensor chan (hw.Trigger), producers map[string]c.LedProducer, sig chan bool) {
	var firstSameTrigger hw.Trigger
	var triggerDelay = c.CONFIG.HoldLED.TriggerSeconds * time.Second

	for {
		select {
		case trigger := <-sensor:
			oldStamp := firstSameTrigger.Timestamp
			newStamp := trigger.Timestamp

			if c.CONFIG.HoldLED.Enabled && (trigger.Value >= HOLD_TRIGGER_VALUE) {
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
		case <-sig:
			log.Println("Ending fireController go-routine")
			return
		}
	}
}

// Local Variables:
// compile-command: "go build"
// End:
