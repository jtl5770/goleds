package driver

import (
	"fmt"
	"log"
	"sort"

	c "lautenbacher.net/goleds/config"
	hw "lautenbacher.net/goleds/hardware"
	p "lautenbacher.net/goleds/producer"
)

var SEGMENTS []*ledsegment

type ledsegment struct {
	firstled     int
	lastled      int
	visible      bool
	spimultiplex int
	leds         []p.Led
}

func clamp(led int) int {
	if led < 0 {
		log.Printf("led index %d is smaller than 0 - using 0", led)
		return 0
	} else if led >= 0 && led <= (c.CONFIG.Hardware.Display.LedsTotal-1) {
		return led
	} else {
		log.Printf("led index %d is smaller than max index %d - using max", led, c.CONFIG.Hardware.Display.LedsTotal-1)
		return c.CONFIG.Hardware.Display.LedsTotal - 1
	}
}

func NewLedSegment(firstled, lastled, spimultiplex int, visible bool) *ledsegment {
	if firstled > lastled {
		log.Printf("First led index %d is bigger than last led index %d - reversing", firstled, lastled)
		tmp := firstled
		firstled = lastled
		lastled = tmp
	}
	if !visible {
		spimultiplex = -1
	}
	inst := ledsegment{
		firstled:     clamp(firstled),
		lastled:      clamp(lastled),
		visible:      visible,
		spimultiplex: spimultiplex,
	}
	return &inst
}

func (s *ledsegment) getSegmentLeds() []p.Led {
	if s.visible {
		return s.leds
	} else {
		return nil
	}
}

func (s *ledsegment) setSegmentLeds(sumleds []p.Led) {
	if s.visible {
		s.leds = sumleds[s.firstled : s.lastled+1]
	}
}

func InitDisplay() {
	SEGMENTS = make([]*ledsegment, 0, len(c.CONFIG.Hardware.Display.LedSegments))
	for _, seg := range c.CONFIG.Hardware.Display.LedSegments {
		SEGMENTS = append(SEGMENTS, NewLedSegment(seg.FirstLed, seg.LastLed, seg.SpiMultiplex, true))
	}

	all := make([]bool, c.CONFIG.Hardware.Display.LedsTotal)
	for _, seg := range SEGMENTS {
		for i := seg.firstled; i <= seg.lastled; i++ {
			if all[i] {
				panic(fmt.Sprintf("Overlapping display segments at index %d", i))
			}
			all[i] = true
		}
	}

	start := -1
	for index, elem := range all {
		if start == -1 && !elem {
			start = index
		} else if start != -1 && elem {
			SEGMENTS = append(SEGMENTS, NewLedSegment(start, index-1, -1, false))
			start = -1
		}
	}
	if start != -1 {
		SEGMENTS = append(SEGMENTS, NewLedSegment(start, len(all)-1, -1, false))
	}

	sort.Slice(SEGMENTS, func(i, j int) bool { return SEGMENTS[i].firstled < SEGMENTS[j].firstled })
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

			if !c.CONFIG.RealHW {
				simulateLedDisplay()
			} else {
				for _, seg := range SEGMENTS {
					if seg.visible {
						hw.SetLedSegment(seg.spimultiplex, seg.getSegmentLeds())
					}
				}
			}
		}
	}
}
