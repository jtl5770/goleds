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
// (default: config.yml) and to be able to react to signals to
// reload the config file or to exit. The config file is read by the
// config package (package config) and the config is stored in a
// global variable CONFIG. The config file is read on startup and
// whenever a SIGHUP signal is received.
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
	d "lautenbacher.net/goleds/driver"
	hw "lautenbacher.net/goleds/hardware"
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
	ledproducers map[string]p.LedProducer
	stopsignal   chan bool
	shutdownWg   sync.WaitGroup
}

// NewApp creates a new App instance
func NewApp() *App {
	return &App{
		ledproducers: make(map[string]p.LedProducer),
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
	exPath := filepath.Dir(ex)
	cfile := flag.String("config", exPath+"/"+c.CONFILE, "Config file to use")
	realp := flag.Bool("real", false, "Set to true if program runs on real hardware")
	sensp := flag.Bool("show-sensors", false, "Set to true if program should only display"+"sensor values (will be random values if -real is not given)")
	flag.Parse()

	c.ReadConfig(*cfile, *realp, *sensp)

	app := NewApp()
	ossignal := make(chan os.Signal, 1)
	app.initialise(ossignal)

	signal.Notify(ossignal, os.Interrupt, syscall.SIGHUP)

	for {
		select {
		case sig := <-ossignal:
			if sig == os.Interrupt {
				log.Println("Exiting...")
				app.shutdown()
				os.Exit(0)
			} else if sig == syscall.SIGHUP {
				log.Println("Resetting...")
				app.shutdown()
				c.ReadConfig(*cfile, *realp, *sensp)
				app.initialise(ossignal)
			}
		}
	}
}

func (a *App) initialise(ossignal chan os.Signal) {
	log.Println("Initializing...")
	a.stopsignal = make(chan bool)
	hw.InitHardware()
	d.InitSensors()
	d.InitDisplay()

	if !c.CONFIG.RealHW || c.CONFIG.SensorShow {
		// we need to pass the os signal channel here to be able to exit the TUI
		d.InitSimulationTUI(ossignal)
	}

	a.ledproducers = make(map[string]p.LedProducer)

	ledReader := u.NewAtomicEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	ledsTotal := c.CONFIG.Hardware.Display.LedsTotal

	// This is the main producer: reacting to a sensor trigger to light the stripes
	if c.CONFIG.SensorLED.Enabled {
		cfg := c.CONFIG.SensorLED
		for uid, sen := range d.Sensors {
			a.ledproducers[uid] = p.NewSensorLedProducer(uid, sen.LedIndex, ledReader, ledsTotal, cfg.HoldTime, cfg.RunUpDelay, cfg.RunDownDelay, cfg.LedRGB)
		}
	}

	if c.CONFIG.HoldLED.Enabled {
		cfg := c.CONFIG.HoldLED
		prodhold := p.NewHoldProducer(HOLD_LED_UID, ledReader, ledsTotal, cfg.HoldTime, cfg.LedRGB)
		a.ledproducers[HOLD_LED_UID] = prodhold
	}

	var prodnight *p.NightlightProducer = nil
	if c.CONFIG.NightLED.Enabled {
		cfg := c.CONFIG.NightLED
		// The Nightlight producer creates a permanent glow during night time
		prodnight = p.NewNightlightProducer(NIGHT_LED_UID, ledReader, ledsTotal,
			cfg.Latitude, cfg.Longitude, cfg.LedRGB)
		a.ledproducers[NIGHT_LED_UID] = prodnight
		prodnight.Start()
	}

	if c.CONFIG.MultiBlobLED.Enabled {
		cfg := c.CONFIG.MultiBlobLED
		blobCfg := make(map[string]p.BlobConfig)
		for k, v := range cfg.BlobCfg {
			blobCfg[k] = p.BlobConfig{
				DeltaX: v.DeltaX,
				X:      v.X,
				Width:  v.Width,
				LedRGB: v.LedRGB,
			}
		}
		// multiblobproducer gets the - maybe nil - prodnight instance to control it
		multiblob := p.NewMultiBlobProducer(MULTI_BLOB_UID, ledReader, prodnight,
			ledsTotal, cfg.Duration, cfg.Delay, blobCfg)
		a.ledproducers[MULTI_BLOB_UID] = multiblob
	}

	if c.CONFIG.CylonLED.Enabled {
		cfg := c.CONFIG.CylonLED
		cylon := p.NewCylonProducer(CYLON_LED_UID, ledReader, ledsTotal,
			cfg.Duration, cfg.Delay, cfg.Step, cfg.Width, cfg.LedRGB)
		a.ledproducers[CYLON_LED_UID] = cylon
	}

	// *FUTURE* init more types of ledproducers if needed/wanted

	a.shutdownWg.Add(4)
	go a.combineAndUpdateDisplay(ledReader, ledWriter)
	go a.fireController()
	go d.DisplayDriver(ledWriter, a.stopsignal, &a.shutdownWg)
	go d.SensorDriver(a.stopsignal, &a.shutdownWg)
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
	hw.CloseGPIO()
}

func (a *App) combineAndUpdateDisplay(r *u.AtomicEvent[p.LedProducer], w chan []p.Led) {
	defer a.shutdownWg.Done()
	var oldSumLeds []p.Led
	allLedRanges := make(map[string][]p.Led)
	ticker := time.NewTicker(c.CONFIG.Hardware.Display.ForceUpdateDelay)
	defer ticker.Stop()
	old_sensorledsrunning := false
	for {
		select {
		case <-r.Channel():
			s := r.Value()
			if c.CONFIG.MultiBlobLED.Enabled || c.CONFIG.CylonLED.Enabled {
				isrunning := false
				for uid := range d.Sensors {
					isrunning = (isrunning || a.ledproducers[uid].GetIsRunning())
				}
				if c.CONFIG.HoldLED.Enabled {
					isrunning = (isrunning || a.ledproducers[HOLD_LED_UID].GetIsRunning())
				}
				// Now we know if any of the sensor driven producers
				// is still running (aka: has any LED on) if NOT (aka:
				// isrunning is false), we detected a change from ON
				// to OFF exactly when old_sensorledsrunning is true;
				// and we can now Start() the multiblobproducer or the
				// cylonproducer.
				if old_sensorledsrunning && !isrunning {
					if c.CONFIG.MultiBlobLED.Enabled {
						a.ledproducers[MULTI_BLOB_UID].Start()
					}
					if c.CONFIG.CylonLED.Enabled {
						a.ledproducers[CYLON_LED_UID].Start()
					}
				} else if !old_sensorledsrunning && isrunning {
					// or the other way around: Stopping the multiblobproducer
					if c.CONFIG.MultiBlobLED.Enabled {
						a.ledproducers[MULTI_BLOB_UID].Stop()
					}
					if c.CONFIG.CylonLED.Enabled {
						a.ledproducers[CYLON_LED_UID].Stop()
					}
				}
				old_sensorledsrunning = isrunning
			}

			allLedRanges[s.GetUID()] = s.GetLeds()
			sumLeds := p.CombineLeds(allLedRanges, c.CONFIG.Hardware.Display.LedsTotal)
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
			case w <- p.CombineLeds(allLedRanges, c.CONFIG.Hardware.Display.LedsTotal):
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

func (a *App) fireController() {
	defer a.shutdownWg.Done()
	var firstSameTrigger *d.Trigger = d.NewTrigger("", 0, time.Now())
	triggerDelay := c.CONFIG.HoldLED.TriggerDelay

	for {
		select {
		case trigger := <-d.SensorReader:
			oldStamp := firstSameTrigger.Timestamp
			newStamp := trigger.Timestamp

			if c.CONFIG.HoldLED.Enabled && (trigger.Value >= c.CONFIG.HoldLED.TriggerValue) {
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
				firstSameTrigger = d.NewTrigger("", 0, time.Now())
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
