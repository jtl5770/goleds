package platform

import (
	"fmt"
	"log"
	"sort"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

// DisplayManager manages the LED segments for a display.
type DisplayManager struct {
	Segments map[string][]*Segment
}

// Segment represents a single LED segment.
type Segment struct {
	FirstLed     int
	LastLed      int
	Visible      bool
	Reverse      bool
	SpiMultiplex string
	Leds         []p.Led
}

// NewDisplayManager initializes and returns a new DisplayManager.
func NewDisplayManager(displayConfig c.DisplayConfig) *DisplayManager {
	dm := &DisplayManager{
		Segments: make(map[string][]*Segment),
	}

	for name, segarray := range displayConfig.LedSegments {
		for _, seg := range segarray {
			dm.Segments[name] = append(dm.Segments[name], NewSegment(seg.FirstLed, seg.LastLed, seg.SpiMultiplex, seg.Reverse, true, displayConfig.LedsTotal))
		}
	}

	// This part handles the "invisible" segments to fill gaps,
	// ensuring all LEDs are accounted for in each named group.
	for name, segarray := range dm.Segments {
		all := make([]bool, displayConfig.LedsTotal)

		for _, seg := range segarray {
			for i := seg.FirstLed; i <= seg.LastLed; i++ {
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
				dm.Segments[name] = append(dm.Segments[name], NewSegment(start, index-1, "__", false, false, displayConfig.LedsTotal))
				start = -1
			}
		}
		if start != -1 {
			dm.Segments[name] = append(dm.Segments[name], NewSegment(start, len(all)-1, "__", false, false, displayConfig.LedsTotal))
		}

		sort.Slice(dm.Segments[name], func(i, j int) bool { return dm.Segments[name][i].FirstLed < dm.Segments[name][j].FirstLed })
	}
	return dm
}

// NewSegment creates a new Segment instance.
func NewSegment(firstled, lastled int, spimultiplex string, reverse bool, visible bool, ledsTotal int) *Segment {
	if firstled > lastled {
		log.Printf("First led index %d is bigger than last led index %d - reversing", firstled, lastled)
		tmp := firstled
		firstled = lastled
		lastled = tmp
	}
	if !visible {
		spimultiplex = "__" // Use a dummy multiplexer for invisible segments
	}
	inst := Segment{
		FirstLed:     clamp(firstled, ledsTotal),
		LastLed:      clamp(lastled, ledsTotal),
		Visible:      visible,
		Reverse:      reverse,
		SpiMultiplex: spimultiplex,
	}
	return &inst
}

// SetLeds sets the LEDs for the segment, applying reversal if configured.
func (s *Segment) SetLeds(sumleds []p.Led) {
	if s.Visible {
		s.Leds = sumleds[s.FirstLed : s.LastLed+1]
		if s.Reverse {
			for i, j := 0, len(s.Leds)-1; i < j; i, j = i+1, j-1 {
				s.Leds[i], s.Leds[j] = s.Leds[j], s.Leds[i]
			}
		}
	}
}

// GetLeds returns the LEDs for the segment if visible, otherwise nil.
func (s *Segment) GetLeds() []p.Led {
	if s.Visible {
		return s.Leds
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
