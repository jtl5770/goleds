package driver

import (
	"fmt"
	"log"
	"sort"

	c "lautenbacher.net/goleds/config"
	hw "lautenbacher.net/goleds/hardware"
	p "lautenbacher.net/goleds/producer"
)

var SEGMENTS map[string][]*ledsegment

type ledsegment struct {
	firstled     int
	lastled      int
	visible      bool
	reverse      bool
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

func NewLedSegment(firstled, lastled, spimultiplex int, reverse bool, visible bool) *ledsegment {
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
		reverse:      reverse,
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
		if s.reverse {
			for i, j := 0, len(s.leds)-1; i < j; i, j = i+1, j-1 {
				s.leds[i], s.leds[j] = s.leds[j], s.leds[i]
			}
		}
	}
}

func InitDisplay() {
	SEGMENTS = make(map[string][]*ledsegment)
	for name, segarray := range c.CONFIG.Hardware.Display.LedSegments {
		for _, seg := range segarray {
			SEGMENTS[name] = append(SEGMENTS[name], NewLedSegment(seg.FirstLed, seg.LastLed, seg.SpiMultiplex, seg.Reverse, true))
		}
	}

	for name, segarray := range SEGMENTS {
		all := make([]bool, c.CONFIG.Hardware.Display.LedsTotal)

		for _, seg := range segarray {
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
				SEGMENTS[name] = append(SEGMENTS[name], NewLedSegment(start, index-1, -1, false, false))
				start = -1
			}
		}
		if start != -1 {
			SEGMENTS[name] = append(SEGMENTS[name], NewLedSegment(start, len(all)-1, -1, false, false))
		}

		sort.Slice(SEGMENTS[name], func(i, j int) bool { return SEGMENTS[name][i].firstled < SEGMENTS[name][j].firstled })
	}
}

func DisplayDriver(display chan ([]p.Led), sig chan bool) {
	for {
		select {
		case <-sig:
			log.Println("Ending DisplayDriver go-routine")
			return
		case sumLeds := <-display:
			for _, segarray := range SEGMENTS {
				for _, seg := range segarray {
					seg.setSegmentLeds(sumLeds)
				}
			}
			if !c.CONFIG.RealHW {
				simulateLedDisplay()
			} else {
				for _, segarray := range SEGMENTS {
					for _, seg := range segarray {
						if seg.visible {
							hw.SetLedSegment(seg.spimultiplex, seg.getSegmentLeds())
						}
					}
				}
			}
		}
	}
}
