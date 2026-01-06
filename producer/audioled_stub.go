// +build !cgo

package producer

import (
	"log/slog"
	"sync"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

// AudioLEDProducer is a stub implementation for environments where CGO is disabled.
type AudioLEDProducer struct {
	*AbstractProducer
}

// NewAudioLEDProducer returns a stub producer that logs a warning.
func NewAudioLEDProducer(uid string, ledsChanged *u.AtomicMapEvent[LedProducer], ledsTotal int, cfg c.AudioLEDConfig) *AudioLEDProducer {
	slog.Warn("AudioLEDProducer: Audio support is disabled in this build (requires CGO).")
	p := &AudioLEDProducer{}
	p.AbstractProducer = NewAbstractProducer(uid, ledsChanged, p.runner, ledsTotal)
	return p
}

func (p *AudioLEDProducer) Exit() {
	p.AbstractProducer.Exit()
}

func (p *AudioLEDProducer) runner() {
	slog.Warn("AudioLEDProducer: runner started but no audio support available.")
	<-p.stopchan
}
