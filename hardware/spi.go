package hardware

// SPI is an interface for SPI communication.
type SPI interface {
	Exchange(write []byte) []byte
}
