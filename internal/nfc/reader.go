package nfc

import (
	"context"
	"time"
)

// Config holds NFC reader configuration
type Config struct {
	PowerTimeout time.Duration
}

// Reader manages NFC tag reading functionality
type Reader struct {
	config     Config
	tapHandler func(string) error
}

// NewReader creates a new NFC reader instance
func NewReader(config Config) *Reader {
	return &Reader{
		config: config,
	}
}

// Start initializes the NFC reader
func (r *Reader) Start(ctx context.Context) error {
	// TODO: Implement actual NFC hardware initialization
	return nil
}

// Stop shuts down the NFC reader
func (r *Reader) Stop() error {
	// TODO: Implement actual NFC hardware shutdown
	return nil
}

// SetTapHandler sets the callback for NFC tag taps
func (r *Reader) SetTapHandler(handler func(string) error) {
	r.tapHandler = handler
}

// handleTagDetected is called when an NFC tag is detected
func (r *Reader) handleTagDetected(uid string) {
	if r.tapHandler != nil {
		r.tapHandler(uid)
	}
}
