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
//
// The software is designed to be configurable via a config file
// (default: config.yml) and to be able to react to signals to
// reload the config file or to exit. The config file is read by the
// config package (package config) and the config is stored in a
// global variable CONFIG. The config file is read on startup and
// whenever a SIGHUP signal is received.
//
// The main functionality is to read the sensors and to drive the LED
// stripes accordingly. The sensor data is read by the hardware packag
// and the LED stripes are driven by the producer package. The
// producer package is designed to be able to handle different types
// of producers, e.g.  the HoldProducer which is triggered by a sensor
// and keeps the stripes lit for a configurable time.
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
	HOLD_LED_UID   = "__hold_producer"
	NIGHT_LED_UID  = "__night_producer"
	MULTI_BLOB_UID = "__multiblob_producer"
	CYLON_LED_UID  = "__cylon_producer"
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

	exit := make(chan os.Signal)
	reload := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)
	signal.Notify(reload, syscall.SIGHUP)

	for {
		select {
		case <-exit:
			log.Println("Exiting...")
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
	hw.Sensors = make(map[string]*hw.Sensor)
	ledproducers = make(map[string]p.LedProducer)
	sigchans = make([](chan bool), 0, 4)
	ledReader := make(chan (p.LedProducer))
	ledWriter := make(chan []p.Led, c.CONFIG.Hardware.Display.LedsTotal)
	sensorReader := make(chan hw.Trigger)

	// This is the main producer: reacting to a sensor trigger to light the stripes
	for uid, cfg := range c.CONFIG.Hardware.Sensors.SensorCfg {
		hw.Sensors[uid] = hw.NewSensor(uid, cfg.LedIndex, cfg.Adc, cfg.AdcChannel, cfg.TriggerValue)
		if c.CONFIG.SensorLED.Enabled {
			ledproducers[uid] = p.NewSensorLedProducer(uid, cfg.LedIndex, ledReader)
		}
	}

	if c.CONFIG.HoldLED.Enabled {
		// The HoldLight producer will be started whenever a sensor
		// produces for longer than a configurable time a signal > a
		// configurable value (see config file for TriggerDelay and
		// TriggerValue) It will generate a brighter, full lit LED
		// stripe and keep it for HoldTime time, if not being
		// triggered again in this time - then it will shut off
		// earlier
		prodhold := p.NewHoldProducer(HOLD_LED_UID, ledReader)
		ledproducers[HOLD_LED_UID] = prodhold
	}

	var prodnight *p.NightlightProducer = nil
	if c.CONFIG.NightLED.Enabled {
		// The Nightlight producer creates a permanent glow during night time
		prodnight = p.NewNightlightProducer(NIGHT_LED_UID, ledReader)
		ledproducers[NIGHT_LED_UID] = prodnight
		prodnight.Start()
	}

	if c.CONFIG.MultiBlobLED.Enabled {
		// multiblobproducer gets the - maybe nil - prodnight instance to control it
		multiblob := p.NewMultiBlobProducer(MULTI_BLOB_UID, ledReader, prodnight)
		ledproducers[MULTI_BLOB_UID] = multiblob
	}

	if c.CONFIG.CylonLED.Enabled {
		cylon := p.NewCylonProducer(CYLON_LED_UID, ledReader)
		ledproducers[CYLON_LED_UID] = cylon
	}

	// *FUTURE* init more types of ledproducers if needed/wanted

	cAUDsignal := make(chan bool)
	fCsignal := make(chan bool)
	DDsignal := make(chan bool)
	SDsignal := make(chan bool)
	sigchans = append(sigchans, cAUDsignal, fCsignal, DDsignal, SDsignal)

	go combineAndUpdateDisplay(ledReader, ledWriter, cAUDsignal)
	go fireController(sensorReader, fCsignal)
	go hw.DisplayDriver(ledWriter, DDsignal)
	go hw.SensorDriver(sensorReader, SDsignal)
}

func reset() {
	log.Println("Resetting...")
	for _, prod := range ledproducers {
		log.Println("Exiting producer: ", prod.GetUID())
		prod.Exit()
	}
	time.Sleep(1 * time.Second)
	for _, sig := range sigchans {
		sig <- true
	}
	time.Sleep(1 * time.Second)
	hw.CloseGPIO()
}

func combineAndUpdateDisplay(r chan (p.LedProducer), w chan ([]p.Led), sig chan bool) {
	var oldSumLeds []p.Led
	allLedRanges := make(map[string][]p.Led)
	ticker := time.NewTicker(c.CONFIG.Hardware.Display.ForceUpdateDelay)
	old_sensorledsrunning := false
	for {
		select {
		case s := <-r:
			if c.CONFIG.MultiBlobLED.Enabled || c.CONFIG.CylonLED.Enabled {
				isrunning := false
				for uid := range hw.Sensors {
					isrunning = (isrunning || ledproducers[uid].GetIsRunning())
				}
				if c.CONFIG.HoldLED.Enabled {
					isrunning = (isrunning || ledproducers[HOLD_LED_UID].GetIsRunning())
				}
				// Now we know if any of the sensor driven producers
				// is still running (aka: has any LED on) if NOT (aka:
				// isrunning is false), we detected a change from ON
				// to OFF exactly when old_sensorledsrunning is true;
				// and we can now Start() the multiblobproducer
				if old_sensorledsrunning && !isrunning {
					if c.CONFIG.MultiBlobLED.Enabled {
						ledproducers[MULTI_BLOB_UID].Start()
					}
					if c.CONFIG.CylonLED.Enabled {
						ledproducers[CYLON_LED_UID].Start()
					}
				} else if !old_sensorledsrunning && isrunning {
					// or the other way around: Stopping the multiblobproducer
					if c.CONFIG.MultiBlobLED.Enabled {
						ledproducers[MULTI_BLOB_UID].Stop()
					}
					if c.CONFIG.CylonLED.Enabled {
						ledproducers[CYLON_LED_UID].Stop()
					}
				}
				old_sensorledsrunning = isrunning
			}

			allLedRanges[s.GetUID()] = s.GetLeds()
			sumLeds := p.CombineLeds(allLedRanges)
			if !reflect.DeepEqual(sumLeds, oldSumLeds) {
				w <- sumLeds
			}
			oldSumLeds = sumLeds
		case <-ticker.C:
			// We do this purely because there occasionally are
			// artifacts on the led line from - maybe/somehow -
			// electrical distortions or crosstalk so we make sure to
			// regularily force an update of the Led stripe
			w <- p.CombineLeds(allLedRanges)
		case <-sig:
			log.Println("Ending combineAndupdateDisplay go-routine")
			ticker.Stop()
			return
		}
	}
}

func fireController(sensor chan (hw.Trigger), sig chan bool) {
	var firstSameTrigger hw.Trigger = hw.Trigger{}
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
					if newStamp.Sub(oldStamp) < (triggerDelay + 1*time.Second) {
						if ledproducers[HOLD_LED_UID].GetIsRunning() {
							ledproducers[HOLD_LED_UID].Stop()
						} else {
							ledproducers[HOLD_LED_UID].Start()
						}
					}
					firstSameTrigger = trigger
				}
			} else {
				firstSameTrigger = hw.Trigger{}
				if producer, ok := ledproducers[trigger.ID]; ok {
					producer.Start()
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
