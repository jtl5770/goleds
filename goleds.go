// A package to read infrared sensor data and drive LED stripes.  The
// hardware interaction is abstracted through a platform interface,
// allowing the application to run on real Raspberry Pi hardware or in
// a terminal-based simulation.
//
// The core logic is organized into several packages:
//
//   - platform: Defines the interface for hardware interaction and
//     provides implementations for Raspberry Pi (rpi) and a terminal
//     UI (tui).
//
//   - producer: Contains different animation producers that generate
//     LED patterns.
//
//   - config: Handles loading and parsing of the application
//     configuration from a YAML file.
//
// The application is configured via a file (default: config.yml) and
// supports dynamic reloading of the configuration on SIGHUP
// signals. It can be gracefully shut down with an Interrupt signal.
//
// The main functionality is to read sensor data from the chosen
// platform and drive the LED stripes using various
// producers. Multiple producers can be active simultaneously, and
// their outputs are combined to create complex lighting effects.
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
	NIGHT_LED_UID  = "__night_producer"
	MULTI_BLOB_UID = "__multiblob_producer"
	CYLON_LED_UID  = "__cylon_producer"
)

// App holds the global state of the application
type App struct {
	ledproducers       map[string]p.LedProducer
	sensorProducers    []p.LedProducer
	stopsignal         chan bool
	shutdownWg         sync.WaitGroup
	ossignal           chan os.Signal
	platform           pl.Platform
	afterprodWg        sync.WaitGroup
	afterpMutex        sync.RWMutex
	afterProdIsRunning bool
	afterProd          []p.LedProducer
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
			log.SetOutput(os.Stderr)
			log.Println("Exiting...")
			app.shutdown()
			os.Exit(0)
		case syscall.SIGHUP:
			log.SetOutput(os.Stderr)
			log.Println("Resetting...")
			app.shutdown()
			app.initialise(*cfile, *realp, *sensp)
		}
	}
}

func (a *App) initialise(cfile string, realp bool, sensp bool) {
	log.Println("Initializing...")

	a.afterProdIsRunning = false
	a.afterProd = make([]p.LedProducer, 0)
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

	// This producer runs all the time and will be started right away here
	if conf.NightLED.Enabled {
		cfg := conf.NightLED
		prodnight := p.NewNightlightProducer(NIGHT_LED_UID, ledReader,
			ledsTotal, cfg.Latitude, cfg.Longitude, cfg.LedRGB)
		a.ledproducers[NIGHT_LED_UID] = prodnight
		prodnight.Start()
	}

	// These producers will be started and stopped on demand depending
	// on the running state of the SensorLedProducers.
	if conf.MultiBlobLED.Enabled {
		cfg := conf.MultiBlobLED
		prodmulti := p.NewMultiBlobProducer(MULTI_BLOB_UID, ledReader,
			ledsTotal, cfg.Duration, cfg.Delay, cfg.BlobCfg)
		a.ledproducers[MULTI_BLOB_UID] = prodmulti
		a.afterProd = append(a.afterProd, prodmulti)
	}

	if conf.CylonLED.Enabled {
		cfg := conf.CylonLED
		prodcylon := p.NewCylonProducer(CYLON_LED_UID, ledReader, ledsTotal,
			cfg.Duration, cfg.Delay, cfg.Step, cfg.Width, cfg.LedRGB)
		a.ledproducers[CYLON_LED_UID] = prodcylon
		a.afterProd = append(a.afterProd, prodcylon)
	}

	// This producer reacts on sensor triggers to light the stripes.
	if conf.SensorLED.Enabled {
		cfg := conf.SensorLED
		a.sensorProducers = make([]p.LedProducer, 0, len(a.platform.GetSensorLedIndices()))
		for uid, ledIndex := range a.platform.GetSensorLedIndices() {
			producer := p.NewSensorLedProducer(uid, ledIndex, ledReader,
				ledsTotal, cfg, &a.afterprodWg)
			a.ledproducers[uid] = producer
			a.sensorProducers = append(a.sensorProducers, producer)
		}
	}

	// *FUTURE* init more types of ledproducers if needed/wanted

	a.shutdownWg.Add(4)

	go a.combineAndUpdateDisplay(ledReader, ledWriter)
	go a.fireController()
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

// This go-routine combines the LED values from all producers and writes them to the
// ledWriter channel.
// It also forces an update of the LED stripe at regular intervals to avoid artifacts.
func (a *App) combineAndUpdateDisplay(ledreader *u.AtomicEvent[p.LedProducer], ledwriter chan []p.Led) {
	defer a.shutdownWg.Done()

	var oldSumLeds []p.Led
	forceupdatedelay := a.platform.GetForceUpdateDelay()
	allLedRanges := make(map[string][]p.Led)
	var ticker *time.Ticker
	if forceupdatedelay > 0 {
		ticker = time.NewTicker(forceupdatedelay)
		defer ticker.Stop()
	}

	for {
		select {
		case <-ledreader.Channel():
			s := ledreader.Value()
			allLedRanges[s.GetUID()] = s.GetLeds()
			sumLeds := p.CombineLeds(allLedRanges, a.platform.GetLedsTotal())
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
			// artifacts on the led stripe from - maybe/somehow -
			// electrical distortions or cross talk so we make sure to
			// regularly force an update of the Led stripe
			select {
			case ledwriter <- p.CombineLeds(allLedRanges, a.platform.GetLedsTotal()):
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

// fireController listens for sensor events and starts/stops the
// corresponding SensorLedProducers. Subsequent triggers for a running
// SensorLedProducer are sent via SendTrigger().  It also manages the
// afterProd producers, which are started after all running
// SensorLedProducers have ended.  and which are stopped
//
// Before a new SensorLedProducer is started, all afterProd producers are stopped
func (a *App) fireController() {
	defer a.shutdownWg.Done()

	for {
		select {
		case trigger := <-a.platform.GetSensorEvents():
			if producer, ok := a.ledproducers[trigger.ID]; ok {
				a.afterpMutex.Lock()
				if !producer.GetIsRunning() {
					for _, prod := range a.afterProd {
						if prod.GetIsRunning() {
							log.Printf("===> Stopping afterprod %s", prod.GetUID())
							prod.Stop() // This will signal the go-routine to stop
						}
					}
					log.Printf("   ===> Starting SensorLedProducer %s", trigger.ID)
					a.afterprodWg.Add(1)
					producer.Start()
					if !a.afterProdIsRunning {
						log.Printf("      ---> Starting afterprodRunner go-routine")
						go a.afterProdRunner()
					}
					a.afterProdIsRunning = true
				}
				producer.SendTrigger(trigger)
				a.afterpMutex.Unlock()
			} else {
				log.Printf("Unknown UID %s", trigger.ID)
			}
		case <-a.stopsignal:
			log.Println("Ending fireController go-routine")
			return
		}
	}
}

func (a *App) afterProdRunner() {
	defer func() {
		a.afterProdIsRunning = false
		a.afterpMutex.Unlock()
	}()

	log.Println("         --> In afterProdRunner go-routine... Blocking on WaitGroup")
	a.afterprodWg.Wait()
	log.Println("         <-- WaitGroup unblocked - ending afterProdRunner go-routing")
	a.afterpMutex.Lock()
	for _, prod := range a.afterProd {
		log.Printf("===> Starting afterProd %s", prod.GetUID())
		prod.Start() // This will start the producer
	}
}
