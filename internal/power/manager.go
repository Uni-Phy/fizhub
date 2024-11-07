package power

import (
	"context"
	"sync"
	"time"
)

// State represents different power states
type State int

const (
	StateActive State = iota
	StateIdle
	StateDeepSleep
)

// Manager handles power management functionality
type Manager struct {
	mutex           sync.RWMutex
	state           State
	idleTimeout     time.Duration
	deepSleepTimer  *time.Timer
	lastActivity    time.Time
	onStateChange   func(State)
	wakeupChannels  []chan struct{}
}

// Config holds power management configuration
type Config struct {
	IdleTimeout     time.Duration
	DeepSleepDelay  time.Duration
}

// NewManager creates a new power manager instance
func NewManager(config Config) *Manager {
	return &Manager{
		state:          StateActive,
		idleTimeout:    config.IdleTimeout,
		lastActivity:   time.Now(),
		wakeupChannels: make([]chan struct{}, 0),
	}
}

// Start initializes the power manager
func (m *Manager) Start(ctx context.Context) error {
	go m.monitorPowerState(ctx)
	return nil
}

// Stop stops the power manager
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.deepSleepTimer != nil {
		m.deepSleepTimer.Stop()
	}

	return nil
}

// RecordActivity records system activity to prevent sleep
func (m *Manager) RecordActivity() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.lastActivity = time.Now()
	
	if m.state != StateActive {
		m.setState(StateActive)
	}

	// Reset deep sleep timer if it exists
	if m.deepSleepTimer != nil {
		m.deepSleepTimer.Reset(m.idleTimeout)
	}
}

// RegisterWakeupChannel registers a channel for deep sleep wake-up notifications
func (m *Manager) RegisterWakeupChannel() chan struct{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ch := make(chan struct{})
	m.wakeupChannels = append(m.wakeupChannels, ch)
	return ch
}

// SetOnStateChange sets the callback for power state changes
func (m *Manager) SetOnStateChange(callback func(State)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.onStateChange = callback
}

// monitorPowerState continuously monitors system activity
func (m *Manager) monitorPowerState(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkIdleState()
		}
	}
}

// checkIdleState checks if the system should transition to idle or deep sleep
func (m *Manager) checkIdleState() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	idleTime := time.Since(m.lastActivity)

	switch {
	case idleTime >= m.idleTimeout && m.state == StateActive:
		m.setState(StateIdle)
		// Start deep sleep timer
		m.deepSleepTimer = time.AfterFunc(m.idleTimeout, func() {
			m.enterDeepSleep()
		})
	}
}

// setState changes the power state and notifies listeners
func (m *Manager) setState(state State) {
	m.state = state
	if m.onStateChange != nil {
		m.onStateChange(state)
	}
}

// enterDeepSleep transitions the system to deep sleep mode
func (m *Manager) enterDeepSleep() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// TODO: Implement actual hardware deep sleep
	// This would involve:
	// 1. Saving necessary state
	// 2. Configuring wake-up sources
	// 3. Setting GPIO pins to appropriate states
	// 4. Triggering hardware deep sleep mode

	m.setState(StateDeepSleep)
}

// WakeUp wakes the system from deep sleep
func (m *Manager) WakeUp() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.state != StateDeepSleep {
		return nil
	}

	// TODO: Implement actual hardware wake-up
	// This would involve:
	// 1. Restoring system state
	// 2. Reinitializing peripherals
	// 3. Resuming normal operation

	// Notify all registered wake-up channels
	for _, ch := range m.wakeupChannels {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	m.setState(StateActive)
	m.lastActivity = time.Now()

	return nil
}

// GetState returns the current power state
func (m *Manager) GetState() State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state
}

// GetLastActivity returns the timestamp of the last recorded activity
func (m *Manager) GetLastActivity() time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.lastActivity
}
