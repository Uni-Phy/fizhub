package led

import "context"

// State represents the LED state
type State int

const (
	StateOff State = iota
	StateIdle
	StateWaiting
	StateSuccess
	StateError
)

// Controller manages LED ring visual feedback
type Controller struct {
	currentState State
}

// NewController creates a new LED controller instance
func NewController() *Controller {
	return &Controller{
		currentState: StateOff,
	}
}

// Start initializes the LED controller
func (c *Controller) Start(ctx context.Context) error {
	// TODO: Implement actual LED hardware initialization
	return nil
}

// Stop shuts down the LED controller
func (c *Controller) Stop() error {
	// TODO: Implement actual LED hardware shutdown
	c.currentState = StateOff
	return nil
}

// SetState updates the LED state and visual feedback
func (c *Controller) SetState(state State) error {
	c.currentState = state
	// TODO: Implement actual LED state changes
	return nil
}

// GetState returns the current LED state
func (c *Controller) GetState() State {
	return c.currentState
}
