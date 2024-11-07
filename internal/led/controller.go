package led

import (
	"context"
	"sync"
	"time"
)

// State represents different LED states
type State int

const (
	StateOff State = iota
	StateIdle
	StateWaiting
	StateSuccess
	StateError
)

// Color represents RGB values for LED control
type Color struct {
	R, G, B uint8
}

var (
	ColorOff     = Color{0, 0, 0}
	ColorIdle    = Color{0, 0, 255}  // Blue
	ColorSuccess = Color{0, 255, 0}  // Green
	ColorError   = Color{255, 0, 0}  // Red
)

// Controller manages LED ring behavior
type Controller struct {
	mutex       sync.RWMutex
	state       State
	brightness  uint8
	isAnimating bool
	stopChan    chan struct{}
}

// NewController creates a new LED controller instance
func NewController() *Controller {
	return &Controller{
		brightness: 255,
		stopChan:   make(chan struct{}),
	}
}

// Start initializes the LED controller
func (c *Controller) Start(ctx context.Context) error {
	// Initialize GPIO for RPi
	// TODO: Implement actual GPIO initialization
	
	go c.animationLoop(ctx)
	return nil
}

// Stop stops all LED animations and turns off LEDs
func (c *Controller) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isAnimating {
		close(c.stopChan)
		c.isAnimating = false
	}

	return c.setColor(ColorOff)
}

// SetState changes the LED state and triggers appropriate animation
func (c *Controller) SetState(state State) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.state = state

	switch state {
	case StateOff:
		return c.setColor(ColorOff)
	case StateIdle:
		return c.setColor(ColorIdle)
	case StateSuccess:
		return c.setColor(ColorSuccess)
	case StateError:
		return c.setColor(ColorError)
	case StateWaiting:
		return c.startSpinAnimation()
	default:
		return nil
	}
}

// setColor sets a solid color on the LED ring
func (c *Controller) setColor(color Color) error {
	// TODO: Implement actual GPIO control for RPi
	// This would involve setting PWM values for RGB channels
	return nil
}

// startSpinAnimation starts the spinning animation for waiting state
func (c *Controller) startSpinAnimation() error {
	if c.isAnimating {
		close(c.stopChan)
	}

	c.stopChan = make(chan struct{})
	c.isAnimating = true

	return nil
}

// animationLoop handles continuous LED animations
func (c *Controller) animationLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var position int
	numLEDs := 12 // Number of LEDs in the ring

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.mutex.RLock()
			if c.state == StateWaiting {
				// Rotate the position for spinning animation
				position = (position + 1) % numLEDs
				c.updateSpinAnimation(position, numLEDs)
			}
			c.mutex.RUnlock()
		}
	}
}

// updateSpinAnimation updates the LED ring for the spinning animation
func (c *Controller) updateSpinAnimation(position, numLEDs int) {
	// TODO: Implement actual LED ring animation
	// This would involve:
	// 1. Calculating brightness for each LED based on position
	// 2. Setting PWM values for each LED
	// 3. Creating a smooth spinning effect
}

// SetBrightness sets the overall brightness of the LED ring
func (c *Controller) SetBrightness(brightness uint8) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.brightness = brightness
	return nil
}

// GetState returns the current LED state
func (c *Controller) GetState() State {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.state
}
