package hardware

import (
	"log"
	"math"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

var SEGMENTS []*Segment

type Segment struct {
	firstled     int
	lastled      int
	visible      bool
	spimultiplex int
	leds         []p.Led
}

func NewSegment(firstled, lastled, spimultiplex int, visible bool) *Segment {
	inst := Segment{
		firstled:     firstled,
		lastled:      lastled,
		visible:      visible,
		spimultiplex: spimultiplex,
	}
	return &inst
}

func (s *Segment) getSegmentLeds() []p.Led {
	if s.visible {
		return s.leds
	} else {
		return nil
	}
}

func (s *Segment) setSegmentLeds(sumleds []p.Led) {
	if s.visible {
		s.leds = sumleds[s.firstled : s.lastled+1]
	}
}

func InitDisplay() {
	SEGMENTS = make([]*Segment, 0, len(c.CONFIG.Hardware.Display.Segments))
	for _, seg := range c.CONFIG.Hardware.Display.Segments {
		SEGMENTS = append(SEGMENTS, NewSegment(seg.FirstLed, seg.LastLed, seg.SpiMultiplex, seg.Visible))
	}
}

func DisplayDriver(display chan ([]p.Led), sig chan bool) {
	for {
		select {
		case <-sig:
			log.Println("Ending DisplayDriver go-routine")
			return
		case sumLeds := <-display:
			for _, seg := range SEGMENTS {
				seg.setSegmentLeds(sumLeds)
			}

			if !c.CONFIG.RealHW && !c.CONFIG.HideTUI {
				simulateLedDisplay()
			} else if c.CONFIG.RealHW {
				// spiMutex.Lock()
				for _, seg := range SEGMENTS {
					if seg.visible {
						setLedSegment(seg.spimultiplex, seg.getSegmentLeds())
					}
				}
				// spiMutex.Unlock()
			}
		}
	}
}

func setLedSegment(multiplex int, values []p.Led) {
	display := make([]byte, 3*len(values))
	for idx, led := range values {
		display[3*idx] = byte(math.Min(led.Red*c.CONFIG.Hardware.Display.ColorCorrection[0], 255))
		display[(3*idx)+1] = byte(math.Min(led.Green*c.CONFIG.Hardware.Display.ColorCorrection[1], 255))
		display[(3*idx)+2] = byte(math.Min(led.Blue*c.CONFIG.Hardware.Display.ColorCorrection[2], 255))
	}
	SPIExchangeMultiplex(multiplex, display)
}
