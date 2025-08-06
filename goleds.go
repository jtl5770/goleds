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
	"fmt"
	"hash/fnv"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	c "lautenbacher.net/goleds/config"
	l "lautenbacher.net/goleds/logging"
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
	ledproducers map[string]p.LedProducer
	sensorProd   []p.LedProducer
	afterProd    []p.LedProducer
	permProd     []p.LedProducer
	stopsignal   chan struct{}
	shutdownWg   sync.WaitGroup
	ossignal     chan os.Signal
	platform     pl.Platform
	sensorProdWg sync.WaitGroup
	afterProdWg  sync.WaitGroup
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

	l.InitialSetup()

	app := NewApp(ossignal)
	app.initialise(*cfile, *realp, *sensp)

	signal.Notify(ossignal, os.Interrupt, syscall.SIGHUP)

	for sig := range ossignal {
		switch sig {
		case os.Interrupt:
			l.BufferOutput() // Start capturing all shutdown logs
			slog.Info("Exiting...")
			app.shutdown()
			l.Close() // Final flush to console/file
			os.Exit(0)
		case syscall.SIGHUP:
			l.BufferOutput()
			slog.Info("Resetting...")
			app.shutdown()
			app.initialise(*cfile, *realp, *sensp)
		}
	}
}

func (a *App) initialise(cfile string, realp bool, sensp bool) {
	slog.Info("Initializing...")

	a.afterProd = make([]p.LedProducer, 0)
	a.permProd = make([]p.LedProducer, 0)
	a.stopsignal = make(chan struct{})
	a.ledproducers = make(map[string]p.LedProducer)

	conf, err := c.ReadConfig(cfile, realp, sensp)
	if err != nil {
		slog.Error("Failed to read config", "error", err)
		os.Exit(1)
	}

	// Configure logging with values from the config file.
	var logConf c.SingleLoggingConfig
	bufferLogs := false
	if conf.RealHW {
		logConf = conf.Logging.HW
	} else {
		logConf = conf.Logging.TUI
		bufferLogs = true
	}

	logToFile := logConf.File != ""

	if err := l.Configure(bufferLogs, logConf.Level, logConf.Format, logToFile, logConf.File); err != nil {
		slog.Error("Failed to configure logging with config values", "error", err)
		// We don't exit here, as logging might still be partially functional.
	}

	// Handle the special "-sensor-show development mode"
	if !conf.RealHW && conf.SensorShow {
		slog.Info("Starting in Sensor Viewer development mode...")
		viewer := pl.NewSensorViewer(conf.Hardware.Sensors, a.ossignal, true)
		go viewer.Start()
		// In this mode, we don't need any platforms or producers, so we exit early.
		return
	}

	// Standard platform setup
	if conf.RealHW {
		rpiPlatform := pl.NewRaspberryPiPlatform(conf)
		if conf.SensorShow {
			viewer := pl.NewSensorViewer(conf.Hardware.Sensors, a.ossignal, false)
			rpiPlatform.SetSensorViewer(viewer)
		}
		a.platform = rpiPlatform
	} else {
		a.platform = pl.NewTUIPlatform(conf, a.ossignal)
	}

	ledReader := u.NewAtomicMapEvent[p.LedProducer]()
	ledWriter := make(chan []p.Led, 1)

	if err := a.platform.Start(ledWriter); err != nil {
		slog.Error("Failed to start platform", "error", err)
		os.Exit(1)
	}

	// Block until the platform signals it's ready. This is crucial for the TUI
	// to prevent race conditions with libraries that interact with the terminal,
	// like portaudio.
	<-a.platform.Ready()
	slog.Info("Platform is ready, starting producers...")

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

	a.shutdownWg.Add(2)

	go a.combineAndUpdateDisplay(ledReader, ledWriter)
	go a.stateManager()
}

func (a *App) shutdown() {
	slog.Info("Shutting down...")
	for _, prod := range a.ledproducers {
		slog.Info("Exiting producer", "uid", prod.GetUID())
		prod.Exit()
	}

	slog.Info("Stopping running go-routines...")
	close(a.stopsignal)
	a.shutdownWg.Wait()
	slog.Info("Main go-routines from goleds.go successfully terminated")

	if a.platform != nil {
		slog.Info("Stopping main platform...", "class", fmt.Sprintf("%T", a.platform))
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
	combinedLeds := make([]p.Led, a.platform.GetLedsTotal())
	var ticker *time.Ticker
	if forceupdatedelay > 0 {
		ticker = time.NewTicker(forceupdatedelay)
		defer ticker.Stop()
	}

	for {
		select {
		case <-ledreader.Channel():
			pmap := ledreader.ConsumeValues()
			for key, prod := range pmap {
				allLedRanges[key] = prod.GetLeds()
			}
			p.CombineLeds(allLedRanges, combinedLeds)
			newLedshash := hashLeds(combinedLeds)
			if newLedshash != oldLedsHash {
				ledsCopy := make([]p.Led, len(combinedLeds))
				copy(ledsCopy, combinedLeds)
				select {
				case ledwriter <- ledsCopy:
				case <-a.stopsignal:
					slog.Info("Ending combineAndupdateDisplay go-routine")
					return
				}
			}
			oldLedsHash = newLedshash
		case <-ticker.C:
			// We do this purely because there occasionally are
			// artifacts on the led stripe from - maybe/somehow -
			// electrical distortions or cross talk so we make sure to
			// regularly force an update of the Led stripe
			ledsCopy := make([]p.Led, len(combinedLeds))
			copy(ledsCopy, combinedLeds)
			select {
			case ledwriter <- ledsCopy:
			case <-a.stopsignal:
				slog.Info("Ending combineAndupdateDisplay go-routine")
				return
			}
		case <-a.stopsignal:
			slog.Info("Ending combineAndupdateDisplay go-routine")
			return
		}
	}
}

// This go routine distributes sensor events and handles the
// transition of the states the LED strip can be in
func (a *App) stateManager() {
	defer a.shutdownWg.Done()

	type State int

	const (
		stateIdle State = iota
		stateSensor
		stateAfterProd
	)

	// This channel signals that all sensor producers are done. It carries a
	// generation number to prevent race conditions where a "done" signal from a
	// previous generation is received after a new sensor event has already
	// started a new generation of producers.
	sensorProdDoneChan := make(chan uint64)
	afterProdDoneChan := make(chan struct{})

	var sensorRun uint64 // The generation counter for sensor producer runs.

	// We are in idle State when starting
	currentState := stateIdle

	// sensorWaiter waits for all sensor producers to finish and then sends the
	// given run number on the done channel, which it captures from the parent scope.
	sensorWaiter := func(run uint64) {
		a.sensorProdWg.Wait()
		slog.Info("   All SensorLedProducer(s) finished, signalling event", "run", run)
		sensorProdDoneChan <- run
	}

	for {
		select {
		case event := <-a.platform.GetSensorEvents():
			// This is the new, simplified, and race-free logic for handling sensor events.
			// It delegates the responsibility of starting the producer to the producer itself.
			producer, ok := a.ledproducers[event.ID]
			if !ok {
				slog.Warn("Received sensor event for unknown producer", "uid", event.ID)
				continue
			}

			switch currentState {
			case stateIdle:
				slog.Info("Sensor event received in idle state", "uid", event.ID)
				// Stop permanent producers.
				for _, prod := range a.permProd {
					slog.Info("<=== Stopping perm Producer", "uid", prod.GetUID())
					prod.TryStop() // It's okay if it's already stopped.
				}

				currentState = stateSensor
				slog.Info("   ===> Starting/Triggering SensorLedProducer", "uid", event.ID)
				producer.SendTrigger(event)

				// Start a waiter that will signal when this generation is complete.
				sensorRun++ // Start a new generation.
				go sensorWaiter(sensorRun)

			case stateSensor:
				slog.Info("        Additional sensor event received in sensor state", "uid", event.ID)
				// A new event arrived while in the sensor state, extending the active phase.
				// We must handle a race condition where the waiter for the *previous*
				// generation of producers might signal completion just as we process this
				// new event, which would cause a premature and incorrect state transition.
				slog.Info("   ===> Starting/Triggering SensorLedProducer", "uid", event.ID)
				producer.SendTrigger(event)

				// The existing waiter goroutine will send a
				// completion signal that will carry the old,
				// now-invalid generation number. We need a new waiter
				// with the new generation number, which is also
				// guaranteed to wait for the just started producer.
				sensorRun++
				go sensorWaiter(sensorRun)

			case stateAfterProd:
				slog.Info("      Sensor event received in afterProd state", "uid", event.ID)
				// Stop any running after-producers.
				for _, prod := range a.afterProd {
					slog.Info("      <=== Stopping afterProd Producer", "uid", prod.GetUID())
					prod.TryStop() // It's okay if it's already stopped.
				}

				currentState = stateSensor
				slog.Info("   ===> Starting/Triggering SensorLedProducer", "uid", event.ID)
				producer.SendTrigger(event)

				// Start a waiter for the new generation.
				sensorRun++
				go sensorWaiter(sensorRun)
			}
		case recvdRun := <-sensorProdDoneChan:
			// Only process the "done" signal if it matches the current generation.
			if recvdRun != sensorRun {
				slog.Info("      Received stale [SensorLedProducer(s) finished] event, ignoring", "received_run", recvdRun, "current_run", sensorRun)
				continue
			}

			slog.Info("      Received valid [SensorLedProducer(s) finished] event, switching to afterProd state", "run", recvdRun)
			currentState = stateAfterProd
			for _, prod := range a.afterProd {
				slog.Info("      ===> Starting afterProd Producer", "uid", prod.GetUID())
				prod.Start()
			}
			go func() {
				a.afterProdWg.Wait()
				slog.Info("      All AfterProdProducer(s) finished, signalling event")
				afterProdDoneChan <- struct{}{}
			}()

		case <-afterProdDoneChan:
			if currentState == stateAfterProd {
				slog.Info("      Received [AfterProdProducer(s) finished] event, switching to idle state")
				currentState = stateIdle
				for _, prod := range a.permProd {
					slog.Info("===> Starting permProd Producer", "uid", prod.GetUID())
					prod.Start()
				}
			} else {
				slog.Info("      Received [AfterProdProducer(s) finished] event, but not in afterProd state, ignoring")
			}

		case <-a.stopsignal:
			slog.Info("Ending stateManager go-routine")
			return
		}
	}
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
