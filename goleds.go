// A package to read infrared sensor data and drive LED stripes.
// The hardware interaction is abstracted through a platform interface,
// allowing the application to run on real Raspberry Pi hardware or in a
// terminal-based simulation.
//
// The core logic is organized into several packages:
//   - platform: Defines the interface for hardware interaction and provides
//     implementations for Raspberry Pi (rpi) and a terminal UI (tui).
//   - producer: Contains different animation producers that generate LED patterns.
//   - config: Handles loading and parsing of the application configuration from a YAML file.
//
// The application is configured via a file (default: config.yml) and supports
// dynamic reloading of the configuration on SIGHUP signals. It can be gracefully
// shut down with an Interrupt signal.
//
// The main functionality is to read sensor data from the chosen platform and
// drive the LED stripes using various producers. Multiple producers can be active
// simultaneously, and their outputs are combined to create complex lighting effects.
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
	sensp := flag.Bool("show-sensors", false, "Set to true if program should only display sensor values.\n"+
		"* will be using live data from the sensor hardware if -real is given - useful for calibrating the sensors' trigger values\n"+
		"* will be using random values if -real is not given - useful only for development of the viewer component itself")
	flag.Parse()

	app := NewApp(ossignal)
	app.initialise(*cfile, *realp, *sensp)

	signal.Notify(ossignal, os.Interrupt, syscall.SIGHUP)

	for sig := range ossignal {
		switch sig {
		case os.Interrupt:
			// Restore logger to stderr
			log.SetOutput(os.Stderr)
			log.Println("Exiting...")
			app.shutdown()
			os.Exit(0)
		case syscall.SIGHUP:
			// Restore logger to stderr
			log.SetOutput(os.Stderr)
			log.Println("Resetting...")
			app.shutdown()
			app.initialise(*cfile, *realp, *sensp)
		}
	}
}

func (a *App) initialise(cfile string, realp bool, sensp bool) {
	log.Println("Initializing...")

	a.stopsignal = make(chan bool)
	a.ledproducers = make(map[string]p.LedProducer)

	conf := c.ReadConfig(cfile, realp, sensp)

	// Handle the special "-sensor-show development mode"
	if !conf.RealHW && conf.SensorShow {
		log.Println("Starting in Sensor Viewer development mode...")
		viewer := pl.NewSensorViewer(conf.Hardware.Sensors.SensorCfg, a.ossignal, true)
		a.shutdownWg.Add(2) // For viewer + generator
		go viewer.Start(a.stopsignal, &a.shutdownWg)
		go viewer.RunSensorDataGenForDev(conf.Hardware.Sensors.LoopDelay, a.stopsignal, &a.shutdownWg)
		// In this mode, we don't need any platforms or producers, so we exit early.
		return
	}

	// Standard platform setup
	if conf.RealHW {
		rpiPlatform := pl.NewRaspberryPiPlatform(conf)
		if conf.SensorShow {
			viewer := pl.NewSensorViewer(conf.Hardware.Sensors.SensorCfg, a.ossignal, false)
			rpiPlatform.SetSensorViewer(viewer)
			a.shutdownWg.Add(1)
			go viewer.Start(a.stopsignal, &a.shutdownWg)
		}
		a.platform = rpiPlatform
	} else {
		a.platform = pl.NewTUIPlatform(conf, a.ossignal, a.stopsignal)
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
		prodnight.Start(u.NewTrigger(NIGHT_LED_UID, 0, time.Now()))
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
	if len(a.ledproducers) > 0 {
		for _, prod := range a.ledproducers {
			log.Println("Exiting producer: ", prod.GetUID())
			prod.Exit()
		}
	}

	log.Println("Stopping running go-routines... ")
	close(a.stopsignal)

	a.shutdownWg.Wait()
	if a.platform != nil {
		a.platform.Stop()
	}
}

func (a *App) combineAndUpdateDisplay(
	sensorProducers []p.LedProducer, holdledp bool, multiblobledp bool, cylonledp bool,
	ledreader *u.AtomicEvent[p.LedProducer], ledwriter chan []p.Led, ledsTotal int, forceupdatedelay time.Duration,
) {
	defer a.shutdownWg.Done()
	var oldSumLeds []p.Led
	allLedRanges := make(map[string][]p.Led)
	var ticker *time.Ticker
	if forceupdatedelay > 0 {
		ticker = time.NewTicker(forceupdatedelay)
		defer ticker.Stop()
	}
	old_sensorledsrunning := false
	for {
		select {
		case <-ledreader.Channel():
			s := ledreader.Value()
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
						a.ledproducers[MULTI_BLOB_UID].Start(u.NewTrigger(MULTI_BLOB_UID, 0, time.Now()))
					}
					if cylonledp {
						a.ledproducers[CYLON_LED_UID].Start(u.NewTrigger(CYLON_LED_UID, 0, time.Now()))
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
				case ledwriter <- sumLeds:
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
			case ledwriter <- p.CombineLeds(allLedRanges, ledsTotal):
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
	var firstSameTrigger *u.Trigger = u.NewTrigger("", 0, time.Now())
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
							a.ledproducers[HOLD_LED_UID].Start(u.NewTrigger(HOLD_LED_UID, 0, time.Now()))
						}
					}
					firstSameTrigger = trigger
				}
			} else {
				firstSameTrigger = u.NewTrigger(trigger.ID, 0, time.Now())
				if producer, ok := a.ledproducers[trigger.ID]; ok {
					producer.Start(trigger)
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
