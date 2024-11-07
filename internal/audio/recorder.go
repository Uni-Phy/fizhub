package audio

import "context"

// State represents the recorder state
type State int

const (
	StateIdle State = iota
	StateRecording
	StateFinished
)

// Config holds audio recorder configuration
type Config struct {
	Format struct {
		SampleRate int `json:"sample_rate"`
		Channels   int `json:"channels"`
		BitDepth   int `json:"bit_depth"`
	} `json:"format"`
	MaxDuration string `json:"max_duration"`
	DeviceID    string `json:"device_id"`
}

// Recorder handles audio recording functionality
type Recorder struct {
	config Config
	state  State
}

// DefaultConfig returns default audio configuration
func DefaultConfig() Config {
	var config Config
	config.Format.SampleRate = 44100
	config.Format.Channels = 1
	config.Format.BitDepth = 16
	config.MaxDuration = "3m"
	config.DeviceID = "default"
	return config
}

// NewRecorder creates a new audio recorder instance
func NewRecorder(config Config) *Recorder {
	return &Recorder{
		config: config,
		state:  StateIdle,
	}
}

// Start initializes the recorder
func (r *Recorder) Start(ctx context.Context) error {
	// TODO: Implement actual audio recording initialization
	return nil
}

// StartRecording begins recording audio
func (r *Recorder) StartRecording() error {
	// TODO: Implement actual recording
	r.state = StateRecording
	return nil
}

// StopRecording stops the current recording
func (r *Recorder) StopRecording() error {
	// TODO: Implement actual recording stop
	r.state = StateFinished
	return nil
}

// SetOnStateChange sets the callback for state changes
func (r *Recorder) SetOnStateChange(callback func(State)) {
	// TODO: Implement state change notifications
}
