package power

import (
	"context"
	"sync"
	"time"
)

// State represents the power state
type State int

const (
	StateActive State = iota
	StateIdle
	StateDeepSleep
)

// Config holds power management configuration
type Config struct {
	IdleTimeout    time.Duration
	DeepSleepDelay time.Duration
}

// Manager handles power state management
type Manager struct {
	mutex         sync.RWMutex
	config        Config
	currentState  State
	lastActivity  time.Time
	stateHandler  func(State)
}

// NewManager creates a new power manager instance
func NewManager(config Config) *Manager {
	return &Manager{
		config:       config,
		currentState: StateActive,
		lastActivity: time.Now(),
	}
}

// Start initializes the power manager
func (m *Manager) Start(ctx context.Context) error {
	// Start power state monitoring
	go m.monitorPowerState(ctx)
	return nil
}

// Stop shuts down the power manager
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.currentState = StateDeepSleep
	if m.stateHandler != nil {
		m.stateHandler(m.currentState)
	}
	return nil
}

// RecordActivity records user activity to prevent sleep
func (m *Manager) RecordActivity() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.lastActivity = time.Now()
	if m.currentState != StateActive {
		m.currentState = StateActive
		if m.stateHandler != nil {
			m.stateHandler(m.currentState)
		}
	}
}

// SetOnStateChange sets the callback for power state changes
func (m *Manager) SetOnStateChange(handler func(State)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.stateHandler = handler
}

// GetState returns the current power state
func (m *Manager) GetState() State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.currentState
}

// GetLastActivity returns the timestamp of the last recorded activity
func (m *Manager) GetLastActivity() time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.lastActivity
}

// monitorPowerState monitors activity and updates power state
func (m *Manager) monitorPowerState(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updatePowerState()
		}
	}
}

// updatePowerState updates the power state based on activity
func (m *Manager) updatePowerState() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	inactiveTime := time.Since(m.lastActivity)

	switch {
	case inactiveTime >= m.config.DeepSleepDelay:
		if m.currentState != StateDeepSleep {
			m.currentState = StateDeepSleep
			if m.stateHandler != nil {
				m.stateHandler(m.currentState)
			}
		}
	case inactiveTime >= m.config.IdleTimeout:
		if m.currentState != StateIdle {
			m.currentState = StateIdle
			if m.stateHandler != nil {
				m.stateHandler(m.currentState)
			}
		}
	}
}
