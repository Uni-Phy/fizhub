package nfc

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Reader represents the NFC reader device
type Reader struct {
	isActive     bool
	mutex        sync.RWMutex
	onTapHandler func(string) error
	powerTimeout time.Duration
	lastRead     time.Time
}

// Config holds the configuration for the NFC reader
type Config struct {
	PowerTimeout time.Duration
}

// NewReader creates a new NFC reader instance
func NewReader(config Config) *Reader {
	return &Reader{
		powerTimeout: config.PowerTimeout,
		lastRead:     time.Now(),
	}
}

// Start initializes the NFC reader and begins listening for tags
func (r *Reader) Start(ctx context.Context) error {
	r.mutex.Lock()
	if r.isActive {
		r.mutex.Unlock()
		return errors.New("reader already active")
	}
	r.isActive = true
	r.mutex.Unlock()

	go r.readLoop(ctx)
	return nil
}

// Stop stops the NFC reader
func (r *Reader) Stop() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if !r.isActive {
		return errors.New("reader not active")
	}
	
	r.isActive = false
	return nil
}

// SetTapHandler sets the callback function for NFC tag detection
func (r *Reader) SetTapHandler(handler func(string) error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.onTapHandler = handler
}

// readLoop continuously polls for NFC tags
func (r *Reader) readLoop(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if uid, err := r.pollForTag(); err == nil && uid != "" {
				r.handleTag(uid)
			}
			
			r.checkPowerTimeout()
		}
	}
}

// pollForTag simulates polling for an NFC tag
// In a real implementation, this would interact with the hardware
func (r *Reader) pollForTag() (string, error) {
	// TODO: Implement actual NFC hardware interaction
	// This is a placeholder that should be replaced with real hardware code
	return "", nil
}

// handleTag processes a detected NFC tag
func (r *Reader) handleTag(uid string) {
	r.mutex.RLock()
	handler := r.onTapHandler
	r.mutex.RUnlock()

	if handler != nil {
		r.lastRead = time.Now()
		if err := handler(uid); err != nil {
			// TODO: Implement error handling strategy
			// Could include LED feedback, logging, retry logic, etc.
		}
	}
}

// checkPowerTimeout checks if the reader should enter power saving mode
func (r *Reader) checkPowerTimeout() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if time.Since(r.lastRead) > r.powerTimeout {
		// TODO: Implement power saving mode
		// This could include:
		// - Reducing polling frequency
		// - Entering deep sleep mode
		// - Turning off non-essential components
	}
}

// IsActive returns whether the reader is currently active
func (r *Reader) IsActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isActive
}
