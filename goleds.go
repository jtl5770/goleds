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
	"hash/fnv"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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
	CLOCK_UID      = "__clock_producer"
	AUDIO_LED_UID  = "__audio_producer"
	MULTI_BLOB_UID = "__multiblob_producer"
	CYLON_LED_UID  = "__cylon_producer"
)

// App holds the global state of the application
type App struct {
	ledproducers       map[string]p.LedProducer
	sensorProd         []p.LedProducer
	stopsignal         chan bool
	shutdownWg         sync.WaitGroup
	ossignal           chan os.Signal
	platform           pl.Platform
	prodMutex          sync.RWMutex
	sensorProdWg       sync.WaitGroup
	afterProdIsRunning bool
	afterProd          []p.LedProducer
	afterProdWg        sync.WaitGroup
	permProdIsRunning  bool
	permProd           []p.LedProducer
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
	a.permProdIsRunning = false
	a.permProd = make([]p.LedProducer, 0)
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

	ledReader := u.NewAtomicMapEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)
	ledsTotal := a.platform.GetLedsTotal()

	// These producers runs all the time and will be started right away here
	if conf.NightLED.Enabled {
		cfg := conf.NightLED
		prodnight := p.NewNightlightProducer(NIGHT_LED_UID, ledReader,
			ledsTotal, cfg.Latitude, cfg.Longitude, cfg.LedRGB)
		a.ledproducers[NIGHT_LED_UID] = prodnight
		a.permProd = append(a.permProd, prodnight)
		prodnight.Start()
	}

	if conf.ClockLED.Enabled {
		cfg := conf.ClockLED
		prodclock := p.NewClockProducer(CLOCK_UID, ledReader, ledsTotal, cfg)
		a.ledproducers[CLOCK_UID] = prodclock
		a.permProd = append(a.permProd, prodclock)
		prodclock.Start()
	}

	if conf.AudioLED.Enabled {
		cfg := conf.AudioLED
		prodaudio := p.NewAudioLEDProducer(AUDIO_LED_UID, ledReader, ledsTotal, cfg)
		a.ledproducers[AUDIO_LED_UID] = prodaudio
		a.permProd = append(a.permProd, prodaudio)
		prodaudio.Start()
	}

	// These producers will be started and stopped on demand depending
	// on the running state of the SensorLedProducers.
	if conf.MultiBlobLED.Enabled {
		cfg := conf.MultiBlobLED
		prodmulti := p.NewMultiBlobProducer(MULTI_BLOB_UID, ledReader,
			ledsTotal, cfg.Duration, cfg.Delay, cfg.BlobCfg, &a.afterProdWg)
		a.ledproducers[MULTI_BLOB_UID] = prodmulti
		a.afterProd = append(a.afterProd, prodmulti)
	}

	if conf.CylonLED.Enabled {
		cfg := conf.CylonLED
		prodcylon := p.NewCylonProducer(CYLON_LED_UID, ledReader, ledsTotal,
			cfg.Duration, cfg.Delay, cfg.Step, cfg.Width, cfg.LedRGB, &a.afterProdWg)
		a.ledproducers[CYLON_LED_UID] = prodcylon
		a.afterProd = append(a.afterProd, prodcylon)
	}

	// This producer reacts on sensor triggers to light the strips.
	if conf.SensorLED.Enabled {
		cfg := conf.SensorLED
		a.sensorProd = make([]p.LedProducer, 0, len(a.platform.GetSensorLedIndices()))
		for uid, ledIndex := range a.platform.GetSensorLedIndices() {
			producer := p.NewSensorLedProducer(uid, ledIndex, ledReader,
				ledsTotal, cfg, &a.sensorProdWg)
			a.ledproducers[uid] = producer
			a.sensorProd = append(a.sensorProd, producer)
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
func (a *App) combineAndUpdateDisplay(ledreader *u.AtomicMapEvent[p.LedProducer], ledwriter chan []p.Led) {
	defer a.shutdownWg.Done()

	var oldLedsHash uint64
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
			pmap := ledreader.Value()
			for key, prod := range pmap {
				allLedRanges[key] = prod.GetLeds()
			}
			sumLeds := p.CombineLeds(allLedRanges, a.platform.GetLedsTotal())
			newLedshash := hashLeds(sumLeds)
			if newLedshash != oldLedsHash {
				select {
				case ledwriter <- sumLeds:
				case <-a.stopsignal:
					log.Println("Ending combineAndupdateDisplay go-routine")
					return
				}
			}
			oldLedsHash = newLedshash
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
				a.prodMutex.Lock()
				if !producer.GetIsRunning() {
					for _, prod := range a.afterProd {
						if prod.GetIsRunning() {
							log.Printf("===> Stopping after Producer %s", prod.GetUID())
							prod.Stop() // This will signal the go-routine to stop
						}
					}
					for _, prod := range a.permProd {
						if prod.GetIsRunning() {
							log.Printf("===> Stopping perm Producer %s", prod.GetUID())
							prod.Stop() // This will signal the go-routine to stop
						}
					}
					log.Printf("   ===> Starting SensorLedProducer %s", trigger.ID)
					producer.Start()
					if !a.afterProdIsRunning {
						log.Printf("      ---> Starting afterprodRunner go-routine")
						// we need to give the stop chan as a
						// parameter to make sure the go routines use
						// this and not a newly created inside the
						// still existing App struct after reset...
						go a.afterProdRunner(a.stopsignal)
						a.afterProdIsRunning = true
					}
				}
				producer.SendTrigger(trigger)
				a.prodMutex.Unlock()
			} else {
				log.Printf("Unknown UID %s", trigger.ID)
			}
		case <-a.stopsignal:
			log.Println("Ending fireController go-routine")
			return
		}
	}
}

func (a *App) afterProdRunner(stop chan bool) {
	log.Println("         --> In afterProdRunner go-routine... Blocking on WaitGroup sensorProdWg")
	a.sensorProdWg.Wait()
	log.Println("         <-- sensorProdWg unblocked - ending afterProdRunner go-routine")
	select {
	case <-stop:
		log.Println("*** afterProdRunner go-routine stopped by signal")
		return
	default:
	}

	a.prodMutex.Lock()
	for _, prod := range a.afterProd {
		log.Printf("===> Starting afterProd %s", prod.GetUID())
		prod.Start()
	}
	go a.permProdRunner(stop)
	a.afterProdIsRunning = false
	a.prodMutex.Unlock()
}

func (a *App) permProdRunner(stop chan bool) {
	log.Println("            --> In permProdRunner go-routine... Blocking on WaitGroup afterProdWg")
	a.afterProdWg.Wait()
	log.Println("            <-- afterProdWg unblocked - ending permProdRunner go-routine")
	select {
	case <-stop:
		log.Println("*** permProdRunner go-routine stopped by signal")
		return
	default:
	}

	a.prodMutex.Lock()
	s_running := false
	for _, prod := range a.sensorProd {
		s_running = s_running || prod.GetIsRunning()
	}
	if !s_running {
		for _, prod := range a.permProd {
			log.Printf("===> Starting permProd %s", prod.GetUID())
			prod.Start()
		}
	} else {
		log.Println("===> Not starting permProd, because SensorLedProducers are running")
	}

	a.permProdIsRunning = false
	a.prodMutex.Unlock()
}

// hashLeds computes a hash for the given LED state array.
// This is used to detect changes in the LED state and avoid unnecessary updates.
func hashLeds(leds []p.Led) uint64 {
	h := fnv.New64a() // FNV-1a is a fast, non-cryptographic hash function.
	for _, led := range leds {
		h.Write([]byte{byte(led.Red), byte(led.Green), byte(led.Blue)})
	}
	return h.Sum64()
}
