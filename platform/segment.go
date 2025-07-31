package platform

import (
	"fmt"
	"log"
	"sort"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

// displayManager manages the LED segments for a display.
type displayManager struct {
	segments map[string][]*segment
}

// segment represents a single LED segment.
type segment struct {
	firstLed     int
	lastLed      int
	visible      bool
	reverse      bool
	spiMultiplex string
	leds         []p.Led
}

func parseDisplaySegments(displayConfig c.DisplayConfig) map[string][]*segment {
	segments := make(map[string][]*segment)

	for name, segarray := range displayConfig.LedSegments {
		for _, seg := range segarray {
			segments[name] = append(segments[name], newSegment(seg.FirstLed, seg.LastLed, seg.SpiMultiplex, seg.Reverse, true, displayConfig.LedsTotal))
		}
	}

	// This part handles the "invisible" segments to fill gaps,
	// ensuring all LEDs are accounted for in each named group.
	for name, segarray := range segments {
		all := make([]bool, displayConfig.LedsTotal)

		for _, seg := range segarray {
			for i := seg.firstLed; i <= seg.lastLed; i++ {
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
				segments[name] = append(segments[name], newSegment(start, index-1, "__", false, false, displayConfig.LedsTotal))
				start = -1
			}
		}
		if start != -1 {
			segments[name] = append(segments[name], newSegment(start, len(all)-1, "__", false, false, displayConfig.LedsTotal))
		}

		sort.Slice(segments[name], func(i, j int) bool { return segments[name][i].firstLed < segments[name][j].firstLed })
	}
	return segments
}

// newSegment creates a new segment instance.
func newSegment(firstled, lastled int, spimultiplex string, reverse bool, visible bool, ledsTotal int) *segment {
	if firstled > lastled {
		log.Printf("First led index %d is bigger than last led index %d - reversing", firstled, lastled)
		tmp := firstled
		firstled = lastled
		lastled = tmp
	}
	if !visible {
		spimultiplex = "__" // Use a dummy multiplexer for invisible segments
	}
	inst := segment{
		firstLed:     clamp(firstled, ledsTotal),
		lastLed:      clamp(lastled, ledsTotal),
		visible:      visible,
		reverse:      reverse,
		spiMultiplex: spimultiplex,
	}
	return &inst
}

// setLeds sets the LEDs for the segment, applying reversal if configured.
func (s *segment) setLeds(sumleds []p.Led) {
	if s.visible {
		s.leds = sumleds[s.firstLed : s.lastLed+1]
		if s.reverse {
			for i, j := 0, len(s.leds)-1; i < j; i, j = i+1, j-1 {
				s.leds[i], s.leds[j] = s.leds[j], s.leds[i]
			}
		}
	}
}

// getLeds returns the LEDs for the segment if visible, otherwise nil.
func (s *segment) getLeds() []p.Led {
	if s.visible {
		return s.leds
	}
	return nil
}

// clamp ensures the LED index is within bounds.
func clamp(led int, ledsTotal int) int {
	if led < 0 {
		log.Printf("led index %d is smaller than 0 - using 0", led)
		return 0
	} else if led >= 0 && led <= (ledsTotal-1) {
		return led
	} else {
		log.Printf("led index %d is smaller than max index %d - using max", led, ledsTotal-1)
		return ledsTotal - 1
	}
}
