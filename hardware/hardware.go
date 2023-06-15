package hardware

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
	"periph.io/x/conn/v3/driver/driverreg"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

var (
	pin17, pin22, pin23, pin24                         rpio.Pin
	PeriphPin17, PeriphPin22, PeriphPin23, PeriphPin24 gpio.PinIO
	spiPort                                            spi.PortCloser
	spiConn                                            spi.Conn
	spiMutex                                           sync.Mutex
)

func InitGPIO() {
	if c.CONFIG.RealHW {
		log.Println("Initialise GPI and Spi...")
		if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
			log.Println("using periph.io...")
			host.Init()
			PeriphPin17 = gpioreg.ByName("GPIO17")
			PeriphPin17.Out(gpio.Low)
			PeriphPin22 = gpioreg.ByName("GPIO22")
			PeriphPin22.Out(gpio.Low)
			PeriphPin23 = gpioreg.ByName("GPIO23")
			PeriphPin23.Out(gpio.High)
			PeriphPin24 = gpioreg.ByName("GPIO24")
			PeriphPin24.Out(gpio.High)
			if _, err := driverreg.Init(); err != nil {
				panic(err)
			}
			// Use spireg SPI port registry to find the first available SPI bus.
			port, err := spireg.Open("")
			if err != nil {
				panic(err)
			}
			spiPort = port
			var freq physic.Frequency
			freq.Set(strconv.Itoa(c.CONFIG.Hardware.SPIFrequency))
			// Convert the spi.Port into a spi.Conn so it can be used for communication.
			conn, err := spiPort.Connect(freq, spi.Mode3, 8)
			if err != nil {
				panic(err)
			}
			spiConn = conn
		} else {
			if err := rpio.Open(); err != nil {
				panic(err)
			}
			if err := rpio.SpiBegin(rpio.Spi0); err != nil {
				panic(err)
			}

			rpio.SpiSpeed(c.CONFIG.Hardware.SPIFrequency)
			pin17 = rpio.Pin(17)
			pin17.Output()
			pin17.Low()

			pin22 = rpio.Pin(22)
			pin22.Output()
			pin22.Low()

			pin23 = rpio.Pin(23)
			pin23.Output()
			pin23.High()

			pin24 = rpio.Pin(24)
			pin24.Output()
			pin24.High()
		}
	} else {
		log.Println("No GPI init done as we are not running on real hardware...")
	}
}

func CloseGPIO() {
	if c.CONFIG.RealHW {
		if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
			spiPort.Close()
		} else {
			rpio.SpiEnd(rpio.Spi0)
			if err := rpio.Close(); err != nil {
				panic(err)
			}
		}
	}
}

func SPIExchange(write []byte) []byte {
	if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
		read := make([]byte, len(write))
		time.Sleep(c.CONFIG.Hardware.SPIDelay)
		if err := spiConn.Tx(write, read); err != nil {
			panic(err)
		}
		// time.Sleep(c.CONFIG.Hardware.Display.SPIDelay)
		return read
	} else {
		time.Sleep(c.CONFIG.Hardware.SPIDelay)
		rpio.SpiExchange(write)
		// time.Sleep(c.CONFIG.Hardware.Display.SPIDelay)
		return write
	}
}

func selectLed(index int) {
	if index == 0 {
		if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
			PeriphPin17.Out(gpio.Low)
			PeriphPin22.Out(gpio.High)
			PeriphPin23.Out(gpio.High)
			PeriphPin24.Out(gpio.High)
		} else {
			pin17.Low()
			pin22.High()
			pin23.High()
			pin24.High()
		}
	} else if index == 1 {
		if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
			PeriphPin17.Out(gpio.High)
			PeriphPin22.Out(gpio.Low)
			PeriphPin23.Out(gpio.High)
			PeriphPin24.Out(gpio.High)
		} else {
			pin17.High()
			pin22.Low()
			pin23.High()
			pin24.High()
		}
	} else {
		panic("No LED")
	}
}

func selectAdc(index int) {
	if index == 0 {
		if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
			PeriphPin17.Out(gpio.Low)
			PeriphPin22.Out(gpio.Low)
			PeriphPin23.Out(gpio.Low)
			PeriphPin24.Out(gpio.High)
		} else {
			pin17.Low()
			pin22.Low()
			pin23.Low()
			pin24.High()
		}
	} else if index == 1 {
		if c.CONFIG.Hardware.GPIOLibrary == "periph.io" {
			PeriphPin17.Out(gpio.Low)
			PeriphPin22.Out(gpio.Low)
			PeriphPin23.Out(gpio.High)
			PeriphPin24.Out(gpio.Low)
		} else {
			pin17.Low()
			pin22.Low()
			pin23.High()
			pin24.Low()
		}
	} else {
		panic("No ADC")
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
