// A package to read infrared sensor data via MCP3008 and drive
// WS-2801 LED stripes.  This is configured according to a very
// special hardware layout of two MCPs handling 2 sensors each and two
// segments of LEDs. Multiplexing 2 MCPs for just 4 sensons is
// normally not needed, but the hardware was built to originally
// having to drive around 14 sensors spaced very closely together
// alongside the two LED stripes. This idea has later been abandoned
// because of heavy crosstalk of the sensors. Now there is only a
// sensor at both sides of each stripe (4 in total). The LED stripe
// layout is due to the special situation in my hallway with a door
// seperating the two stripes. All hardware related things are defined
// in the hardware/ directory (package hardware)
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	c "lautenbacher.net/goleds/config"
	hw "lautenbacher.net/goleds/hardware"
	p "lautenbacher.net/goleds/producer"
)

const (
	HOLD_LED_UID  = "__hold_producer"
	NIGHT_LED_UID = "__night_producer"
)

var (
	ledproducers map[string]p.LedProducer
	sigchans     [](chan bool)
)

// main driver loop to setup hardware, go routines etc., listen for signals
// to either exit or reload config or log sensor statistics
func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	cfile := flag.String("config", exPath+"/"+c.CONFILE, "Config file to use")
	realp := flag.Bool("real", false, "Set to true if program runs on real hardware")
	flag.Parse()

	c.ReadConfig(*cfile, *realp)
	initialise()

	osSig := make(chan os.Signal)
	reload := make(chan os.Signal)
	signal.Notify(osSig, os.Interrupt)
	signal.Notify(reload, syscall.SIGHUP)

	for {
		select {
		case <-osSig:
			log.Println("Exiting...")
			reset()
			os.Exit(0)
		case <-reload:
			reset()
			c.ReadConfig(*cfile, *realp)
			initialise()
		}
	}
}

func initialise() {
	log.Println("Initialising...")
	hw.InitGPIO()
	hw.Sensors = make(map[string]hw.Sensor)
	ledproducers = make(map[string]p.LedProducer)
	sigchans = make([](chan bool), 0, 4)
	ledReader := make(chan (p.LedProducer))
	ledWriter := make(chan []p.Led, c.CONFIG.Hardware.Display.LedsTotal)
	sensorReader := make(chan hw.Trigger)

	for uid, cfg := range c.CONFIG.Hardware.Sensors.SensorCfg {
		hw.Sensors[uid] = hw.NewSensor(cfg.LedIndex, cfg.Adc, cfg.AdcChannel, cfg.TriggerValue)
		if c.CONFIG.SensorLED.Enabled {
			ledproducers[uid] = p.NewSensorLedProducer(uid, cfg.LedIndex, ledReader)
		}
	}

	if c.CONFIG.NightLED.Enabled {
		// The Nightlight producer makes a permanent red (default)
		// glow during night time
		prodnight := p.NewNightlightProducer(NIGHT_LED_UID, ledReader)
		ledproducers[NIGHT_LED_UID] = prodnight
		prodnight.Fire()
	}

	if c.CONFIG.HoldLED.Enabled {
		// The HoldLight producer will be fired whenever a sensor
		// produces for longer than a configurable time a signal > a
		// configurable value (see config file for TriggerDelay and
		// TriggerValue) It will generate a brighter, full lit LED
		// stripe and keep it for FULL_HIGH_HOLD time, if not being
		// triggered again in this time - then it will shut off
		// earlier
		prodhold := p.NewHoldProducer(HOLD_LED_UID, ledReader)
		ledproducers[HOLD_LED_UID] = prodhold
	}

	if c.CONFIG.BlobLED.Enabled {
		for uid := range c.CONFIG.BlobLED.BlobCfg {
			prodblob := p.NewBlobProducer(uid, ledReader)
			ledproducers[uid] = prodblob
			prodblob.Fire()
		}
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

func reset() {
	log.Println("Resetting...")
	for _, prod := range ledproducers {
		prod.Exit()
	}
	time.Sleep(2 * time.Second)
	for _, sig := range sigchans {
		sig <- true
	}
	time.Sleep(2 * time.Second)
	hw.CloseGPIO()
}

func combineAndupdateDisplay(r chan (p.LedProducer), w chan ([]p.Led), sig chan bool) {
	var oldSumLeds []p.Led
	allLedRanges := make(map[string][]p.Led)
	ticker := time.NewTicker(c.CONFIG.Hardware.Display.ForceUpdateDelay)
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
	triggerDelay := c.CONFIG.HoldLED.TriggerDelay

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
