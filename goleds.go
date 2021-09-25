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

	c "lautenbacher.net/goleds/config"
	hw "lautenbacher.net/goleds/hardware"
	p "lautenbacher.net/goleds/producer"
)

const HOLD_LED_UID = "hold_led"
const FORCED_UPDATE_INTERVAL = 5 * time.Second

var ledproducers map[string]p.LedProducer
var sigchans [](chan bool)

func main() {
	cfile := flag.String("config", c.CONFILE, "Config file to use")
	realp := flag.Bool("real", false, "Set to true if program runs on real hardware")
	flag.Parse()
	osSig := make(chan os.Signal)
	reload := make(chan os.Signal)
	signal.Notify(osSig, os.Interrupt)
	signal.Notify(reload, syscall.SIGHUP)

	Initialise(*cfile, *realp, true)

	for {
		select {
		case <-osSig:
			fmt.Println("\nExiting...")
			os.Exit(0)
		case <-reload:
			fmt.Println("\nResetting...")
			ResetAll()
			fmt.Println("\nInitialising...")
			Initialise(*cfile, *realp, false)
		}
	}
}

func Initialise(cfile string, realhw bool, firsttime bool) {
	c.ReadConfig(cfile, realhw)
	log.Println(c.CONFIG)

	hw.InitGpioAndSensors(firsttime)
	ledproducers = make(map[string]p.LedProducer)
	sigchans = make([](chan bool), 0)
	ledReader := make(chan (p.LedProducer))
	ledWriter := make(chan []p.Led, c.CONFIG.Hardware.Display.LedsTotal)
	sensorReader := make(chan hw.Trigger)

	if c.CONFIG.SensorLED.Enabled {
		for uid := range hw.Sensors {
			ledproducers[uid] = p.NewSensorLedProducer(uid, hw.Sensors[uid].LedIndex, ledReader)
		}
	}
	if c.CONFIG.NightLED.Enabled {
		// The Nightlight producer makes a permanent red glow (by default) during night time
		prodnight := p.NewNightlightProducer("night_led", ledReader)
		ledproducers["night_led"] = prodnight
		prodnight.Fire()
	}

	if c.CONFIG.HoldLED.Enabled {
		// The HoldLight producer will be fired whenever a sensor produces for HOLD_TRIGGER_DELAY a signal > HOLD_TRIGGER_VALUE
		// It will generate a brighter, full lit LED stripe and keep it for FULL_HIGH_HOLD time, if not being triggered again
		// in this time - then it will shut off earlier
		prodhold := p.NewHoldProducer(HOLD_LED_UID, ledReader)
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

func combineAndupdateDisplay(r chan (p.LedProducer), w chan ([]p.Led), sig chan bool) {
	var oldSumLeds []p.Led
	allLedRanges := make(map[string][]p.Led)
	ticker := time.NewTicker(FORCED_UPDATE_INTERVAL)
	for uid := range hw.Sensors {
		allLedRanges[uid] = make([]p.Led, c.CONFIG.Hardware.Display.LedsTotal)
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
			// so we make sure to regularily update the Led stripe
			w <- combineLeds(allLedRanges)
		case <-sig:
			log.Println("Ending combineAndupdateDisplay go-routine")
			ticker.Stop()
			return
		}
	}
}

func combineLeds(allLedRanges map[string][]p.Led) []p.Led {
	sumLeds := make([]p.Led, c.CONFIG.Hardware.Display.LedsTotal)
	for _, currleds := range allLedRanges {
		for j, v := range currleds {
			sumLeds[j] = v.Max(sumLeds[j])
		}
	}
	return sumLeds
}

func fireController(sensor chan (hw.Trigger), producers map[string]p.LedProducer, sig chan bool) {
	var firstSameTrigger hw.Trigger
	var triggerDelay = c.CONFIG.HoldLED.TriggerSeconds * time.Second

	for {
		select {
		case trigger := <-sensor:
			oldStamp := firstSameTrigger.Timestamp
			newStamp := trigger.Timestamp

			if c.CONFIG.HoldLED.Enabled && (trigger.Value >= c.CONFIG.HoldLED.TriggerValue) {
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
