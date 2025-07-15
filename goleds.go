// A package to read infrared sensor data via MCP3008 and drive
// WS-2801 LED stripes.  This is configured according to a very
// special hardware layout of two MCPs handling 2 sensors each and two
// segments of LEDs. Multiplexing 2 MCPs for just 4 sensors is
// normally not needed, but the hardware was built to originally
// having to drive around 14 sensors spaced very closely together
// alongside the two LED stripes. This idea has later been abandoned
// because of heavy cross-talk of the sensors. Now there is only a
// sensor at both sides of each stripe (4 in total). The LED stripe
// layout is due to the special situation in my hallway with a door
// separating the two stripes.
//
// The devices (stripes, MCPs) are talked to via SPI. The multiplexing
// is done via logical gates driven by GPIOs.  All hardware related
// things are defined in the hardware/ directory (package hardware)
// but the layout (number of stripe segments, MCPs, sensors) can be
// changed dynamically via the config file.
//
// The software is designed to be configured via an config file
// (default: config.yml) and to be able to react to signals to reload
// the config file or to exit. The config file is read by the config
// package (config.ReadConfig()). The config file is read on startup
// and whenever a SIGHUP signal is received.
//
// The main functionality is to read the sensors and to drive the LED
// stripes accordingly. The sensor data is read by the hardware package
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
	"sync"
	"syscall"
	"time"

	c "lautenbacher.net/goleds/config"
	pl "lautenbacher.net/goleds/platform"
	p "lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/rpi"
	"lautenbacher.net/goleds/tui"
	u "lautenbacher.net/goleds/util"
)

// UIDs for the different types of producers
const (
	HOLD_LED_UID   = "__hold_producer"
	NIGHT_LED_UID  = "__night_producer"
	MULTI_BLOB_UID = "__multiblob_producer"
	CYLON_LED_UID  = "__cylon_producer"
)

// App holds the global state of the application
type App struct {
	ledproducers    map[string]p.LedProducer
	sensorProducers []p.LedProducer
	stopsignal      chan bool
	shutdownWg      sync.WaitGroup
	ossignal        chan os.Signal
	platform        pl.Platform
}

// NewApp creates a new App instance
func NewApp(ossignal chan os.Signal) *App {
	return &App{
		ossignal: ossignal,
	}
}

// main driver loop to setup hardware, go routines etc.,
// The main loop is designed to be able to reload the config file
// dynamically and to react to signals to either exit or reload the
// config file.

func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	ossignal := make(chan os.Signal, 1)
	exPath := filepath.Dir(ex)
	cfile := flag.String("config", exPath+"/"+c.CONFILE, "Config file to use")
	realp := flag.Bool("real", false, "Set to true if program runs on real hardware")
	sensp := flag.Bool("show-sensors", false, "Set to true if program should only display"+"sensor values (will be random values if -real is not given)")
	flag.Parse()

	app := NewApp(ossignal)
	app.initialise(*cfile, *realp, *sensp)

	signal.Notify(ossignal, os.Interrupt, syscall.SIGHUP)

	for {
		select {
		case sig := <-ossignal:
			switch sig {
			case os.Interrupt:
				log.Println("Exiting...")
				app.shutdown()
				os.Exit(0)
			case syscall.SIGHUP:
				log.Println("Resetting...")
				app.shutdown()
				app.initialise(*cfile, *realp, *sensp)
			}
		}
	}
}

func (a *App) initialise(cfile string, realp bool, sensp bool) {
	log.Println("Initializing...")

	a.stopsignal = make(chan bool)
	a.ledproducers = make(map[string]p.LedProducer)

	conf := c.ReadConfig(cfile, realp, sensp)

	if conf.RealHW {
		a.platform = rpi.NewPlatform(conf)
	} else {
		a.platform = tui.NewPlatform(a.ossignal, conf)
	}

	if err := a.platform.Start(); err != nil {
		log.Fatalf("Failed to start platform: %v", err)
	}

	ledReader := u.NewAtomicEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	ledsTotal := a.platform.GetLedsTotal()

	sensorledp := conf.SensorLED.Enabled
	multiblobledp := conf.MultiBlobLED.Enabled
	cylonledp := conf.CylonLED.Enabled
	holdledp := conf.HoldLED.Enabled
	nightledp := conf.NightLED.Enabled

	// This is the main producer: reacting to a sensor trigger to light the stripes
	if sensorledp {
		cfg := conf.SensorLED
		a.sensorProducers = make([]p.LedProducer, 0, len(a.platform.GetSensorLedIndices()))
		for uid, ledIndex := range a.platform.GetSensorLedIndices() {
			producer := p.NewSensorLedProducer(uid, ledIndex, ledReader,
				ledsTotal, cfg.HoldTime, cfg.RunUpDelay, cfg.RunDownDelay, cfg.LedRGB)
			a.ledproducers[uid] = producer
			a.sensorProducers = append(a.sensorProducers, producer)
		}
	}

	if holdledp {
		cfg := conf.HoldLED
		prodhold := p.NewHoldProducer(HOLD_LED_UID, ledReader,
			ledsTotal, cfg.HoldTime, cfg.LedRGB)
		a.ledproducers[HOLD_LED_UID] = prodhold
	}

	var prodnight *p.NightlightProducer = nil
	if nightledp {
		cfg := conf.NightLED
		// The Nightlight producer creates a permanent glow during night time
		prodnight = p.NewNightlightProducer(NIGHT_LED_UID, ledReader,
			ledsTotal, cfg.Latitude, cfg.Longitude, cfg.LedRGB)
		a.ledproducers[NIGHT_LED_UID] = prodnight
		prodnight.Start()
	}

	if multiblobledp {
		cfg := conf.MultiBlobLED
		// multiblobproducer gets the - maybe nil - prodnight instance to control it
		multiblob := p.NewMultiBlobProducer(MULTI_BLOB_UID, ledReader, prodnight,
			ledsTotal, cfg.Duration, cfg.Delay, cfg.BlobCfg)
		a.ledproducers[MULTI_BLOB_UID] = multiblob
	}

	if cylonledp {
		cfg := conf.CylonLED
		cylon := p.NewCylonProducer(CYLON_LED_UID, ledReader, ledsTotal,
			cfg.Duration, cfg.Delay, cfg.Step, cfg.Width, cfg.LedRGB)
		a.ledproducers[CYLON_LED_UID] = cylon
	}

	// *FUTURE* init more types of ledproducers if needed/wanted

	a.shutdownWg.Add(4)

	go a.combineAndUpdateDisplay(a.sensorProducers, holdledp, multiblobledp, cylonledp,
		ledReader, ledWriter, ledsTotal, a.platform.GetForceUpdateDelay())
	go a.fireController(holdledp, conf.HoldLED.TriggerDelay, conf.HoldLED.TriggerValue)
	go a.platform.DisplayDriver(ledWriter, a.stopsignal, &a.shutdownWg)
	go a.platform.SensorDriver(a.stopsignal, &a.shutdownWg)
}

func (a *App) shutdown() {
	log.Println("Shutting down...")
	for _, prod := range a.ledproducers {
		log.Println("Exiting producer: ", prod.GetUID())
		prod.Exit()
	}

	log.Println("Stopping running go-routines... ")
	close(a.stopsignal)

	a.shutdownWg.Wait()
	a.platform.Stop()
}

func (a *App) combineAndUpdateDisplay(
	sensorProducers []p.LedProducer, holdledp bool, multiblobledp bool, cylonledp bool,
	r *u.AtomicEvent[p.LedProducer], w chan []p.Led, ledsTotal int, forceupdatedelay time.Duration,
) {
	defer a.shutdownWg.Done()
	var oldSumLeds []p.Led
	allLedRanges := make(map[string][]p.Led)
	ticker := time.NewTicker(forceupdatedelay)
	defer ticker.Stop()
	old_sensorledsrunning := false
	for {
		select {
		case <-r.Channel():
			s := r.Value()
			if multiblobledp || cylonledp {
				isrunning := false
				for _, producer := range sensorProducers {
					isrunning = (isrunning || producer.GetIsRunning())
				}
				if holdledp {
					isrunning = (isrunning || a.ledproducers[HOLD_LED_UID].GetIsRunning())
				}
				// Now we know if any of the sensor driven producers
				// is still running (aka: has any LED on) if NOT (aka:
				// isrunning is false), we detected a change from ON
				// to OFF exactly when old_sensorledsrunning is true;
				// and we can now Start() the multiblobproducer or the
				// cylonproducer.
				if old_sensorledsrunning && !isrunning {
					if multiblobledp {
						a.ledproducers[MULTI_BLOB_UID].Start()
					}
					if cylonledp {
						a.ledproducers[CYLON_LED_UID].Start()
					}
				} else if !old_sensorledsrunning && isrunning {
					// or the other way around: Stopping the multiblobproducer
					if multiblobledp {
						a.ledproducers[MULTI_BLOB_UID].Stop()
					}
					if cylonledp {
						a.ledproducers[CYLON_LED_UID].Stop()
					}
				}
				old_sensorledsrunning = isrunning
			}

			allLedRanges[s.GetUID()] = s.GetLeds()
			sumLeds := p.CombineLeds(allLedRanges, ledsTotal)
			if !reflect.DeepEqual(sumLeds, oldSumLeds) {
				select {
				case w <- sumLeds:
				case <-a.stopsignal:
					log.Println("Ending combineAndupdateDisplay go-routine")
					return
				}
			}
			oldSumLeds = sumLeds
		case <-ticker.C:
			// We do this purely because there occasionally are
			// artifacts on the led line from - maybe/somehow -
			// electrical distortions or cross talk so we make sure to
			// regularly force an update of the Led stripe
			select {
			case w <- p.CombineLeds(allLedRanges, ledsTotal):
			case <-a.stopsignal:
				log.Println("Ending combineAndupdateDisplay go-routine")
				return
			}
		case <-a.stopsignal:
			log.Println("Ending combineAndupdateDisplay go-routine")
			return
		}
	}
}

func (a *App) fireController(holdledp bool, triggerDelay time.Duration, triggerValue int) {
	defer a.shutdownWg.Done()
	var firstSameTrigger *pl.Trigger = pl.NewTrigger("", 0, time.Now())
	for {
		select {
		case trigger := <-a.platform.GetSensorEvents():
			oldStamp := firstSameTrigger.Timestamp
			newStamp := trigger.Timestamp

			if holdledp && (trigger.Value >= triggerValue) {
				if trigger.ID != firstSameTrigger.ID {
					firstSameTrigger = trigger
				} else if newStamp.Sub(oldStamp) > triggerDelay {
					if newStamp.Sub(oldStamp) < (triggerDelay + 1*time.Second) {
						if a.ledproducers[HOLD_LED_UID].GetIsRunning() {
							a.ledproducers[HOLD_LED_UID].Stop()
						} else {
							a.ledproducers[HOLD_LED_UID].Start()
						}
					}
					firstSameTrigger = trigger
				}
			} else {
				firstSameTrigger = pl.NewTrigger("", 0, time.Now())
				if producer, ok := a.ledproducers[trigger.ID]; ok {
					producer.Start()
				} else {
					log.Printf("Unknown UID %s", trigger.ID)
				}
			}
		case <-a.stopsignal:
			log.Println("Ending fireController go-routine")
			return
		}
	}
}
