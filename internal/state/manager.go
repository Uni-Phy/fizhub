package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Phase represents different system phases
type Phase int

const (
	PhaseInitial Phase = iota
	PhaseCollectingUIDs
	PhaseValidating
	PhaseRecordingMessage
	PhaseComplete
)

// Event represents different system events
type Event int

const (
	EventNFCTap Event = iota
	EventUIDValidated
	EventRecordingStarted
	EventRecordingComplete
	EventError
)

// Manager handles system state and phase transitions
type Manager struct {
	mutex           sync.RWMutex
	currentPhase    Phase
	collectedUIDs   []string
	validAccounts   []string
	bondID          string
	lastEvent       time.Time
	subscribers     map[Phase][]func(Phase)
	errorHandlers   []func(error)
}

// NewManager creates a new state manager instance
func NewManager() *Manager {
	return &Manager{
		currentPhase:  PhaseInitial,
		collectedUIDs: make([]string, 0),
		subscribers:   make(map[Phase][]func(Phase)),
		errorHandlers: make([]func(error), 0),
	}
}

// Start initializes the state manager
func (m *Manager) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.currentPhase = PhaseCollectingUIDs
	m.notifySubscribers()
	return nil
}

// HandleEvent processes system events and updates state accordingly
func (m *Manager) HandleEvent(event Event, data interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.lastEvent = time.Now()

	switch event {
	case EventNFCTap:
		return m.handleNFCTap(data.(string))
	case EventUIDValidated:
		return m.handleUIDValidated(data.([]string))
	case EventRecordingStarted:
		return m.handleRecordingStarted()
	case EventRecordingComplete:
		return m.handleRecordingComplete()
	case EventError:
		return m.handleError(data.(error))
	default:
		return errors.New("unknown event")
	}
}

// handleNFCTap processes an NFC tap event
func (m *Manager) handleNFCTap(uid string) error {
	if m.currentPhase != PhaseCollectingUIDs {
		return errors.New("not collecting UIDs")
	}

	// Check for duplicate UID
	for _, existingUID := range m.collectedUIDs {
		if existingUID == uid {
			return errors.New("duplicate UID")
		}
	}

	// Store the raw UID
	m.collectedUIDs = append(m.collectedUIDs, uid)

	// If we have three UIDs, transition to validation phase
	if len(m.collectedUIDs) == 3 {
		m.currentPhase = PhaseValidating
		m.notifySubscribers()
	}

	return nil
}

// handleUIDValidated processes successful UID validation
func (m *Manager) handleUIDValidated(accounts []string) error {
	if m.currentPhase != PhaseValidating {
		return errors.New("not in validation phase")
	}

	m.validAccounts = accounts
	m.bondID = generateBondID(m.collectedUIDs)
	m.currentPhase = PhaseRecordingMessage
	m.notifySubscribers()

	return nil
}

// handleRecordingStarted processes the start of message recording
func (m *Manager) handleRecordingStarted() error {
	if m.currentPhase != PhaseRecordingMessage {
		return errors.New("not in recording phase")
	}

	return nil
}

// handleRecordingComplete processes completion of message recording
func (m *Manager) handleRecordingComplete() error {
	if m.currentPhase != PhaseRecordingMessage {
		return errors.New("not in recording phase")
	}

	m.currentPhase = PhaseComplete
	m.notifySubscribers()

	return nil
}

// handleError processes system errors
func (m *Manager) handleError(err error) error {
	for _, handler := range m.errorHandlers {
		handler(err)
	}
	return nil
}

// Subscribe registers a callback for phase changes
func (m *Manager) Subscribe(phase Phase, callback func(Phase)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.subscribers[phase] == nil {
		m.subscribers[phase] = make([]func(Phase), 0)
	}
	m.subscribers[phase] = append(m.subscribers[phase], callback)
}

// SubscribeError registers an error handler
func (m *Manager) SubscribeError(handler func(error)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.errorHandlers = append(m.errorHandlers, handler)
}

// notifySubscribers notifies all subscribers of the current phase
func (m *Manager) notifySubscribers() {
	if callbacks, ok := m.subscribers[m.currentPhase]; ok {
		for _, callback := range callbacks {
			callback(m.currentPhase)
		}
	}
}

// GetPhase returns the current system phase
func (m *Manager) GetPhase() Phase {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.currentPhase
}

// GetCollectedUIDs returns the currently collected UIDs
func (m *Manager) GetCollectedUIDs() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return append([]string{}, m.collectedUIDs...)
}

// GetFormattedUIDs returns the UIDs in the Cursive URL format
func (m *Manager) GetFormattedUIDs() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	formatted := make([]string, len(m.collectedUIDs))
	for i, uid := range m.collectedUIDs {
		formatted[i] = fmt.Sprintf("https://nfc.cursive.team/tap?uid=%s", uid)
	}
	return formatted
}

// GetBondID returns the current bond ID
func (m *Manager) GetBondID() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.bondID
}

// Reset resets the state manager to initial conditions
func (m *Manager) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.currentPhase = PhaseCollectingUIDs
	m.collectedUIDs = make([]string, 0)
	m.validAccounts = nil
	m.bondID = ""
	m.notifySubscribers()
}

// generateBondID creates a unique bond ID from UIDs
func generateBondID(uids []string) string {
	// Create a unique identifier by combining UIDs and timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339)
	combined := fmt.Sprintf("%s-%s", timestamp, uids)
	
	// Generate SHA-256 hash
	hash := sha256.New()
	hash.Write([]byte(combined))
	
	// Return first 16 characters of the hex-encoded hash
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
