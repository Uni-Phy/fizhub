package fizhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fizhub/internal/audio"
	"fizhub/internal/led"
	"fizhub/internal/network"
	"fizhub/internal/nfc"
	"fizhub/internal/power"
	"fizhub/internal/state"
	"github.com/gorilla/mux"
)

type Config struct {
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
	Cursive struct {
		URL     string   `json:"url"`
		Timeout Duration `json:"timeout"`
	} `json:"cursive"`
	MQTT struct {
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"mqtt"`
	NFC struct {
		PowerTimeout Duration `json:"power_timeout"`
	} `json:"nfc"`
	Power struct {
		IdleTimeout    Duration `json:"idle_timeout"`
		DeepSleepDelay Duration `json:"deep_sleep_delay"`
	} `json:"power"`
	Audio audio.Config `json:"audio"`
}

type Application struct {
	config     Config
	router     *mux.Router
	server     *http.Server
	nfcReader  *nfc.Reader
	ledCtrl    *led.Controller
	powerMgr   *power.Manager
	stateMgr   *state.Manager
	recorder   *audio.Recorder
	client     *network.Client
	mqttBroker *network.MQTTBroker
}

// Duration is a wrapper around time.Duration for JSON unmarshaling
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

func loadConfig() (Config, error) {
	log.Println("Loading configuration...")
	var config Config
	file, err := os.Open("configs/config.json")
	if err != nil {
		log.Println("Configuration file not found, using default configuration")
		return getDefaultConfig(), nil
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return Config{}, fmt.Errorf("failed to decode config: %w", err)
	}
	log.Println("Configuration loaded successfully")
	return config, nil
}

func getDefaultConfig() Config {
	config := Config{}
	config.Server.Port = "8080"
	config.Cursive.URL = "http://nfc.cursive.team"
	config.Cursive.Timeout = Duration{30 * time.Second}
	config.MQTT.Port = 1883
	config.MQTT.Username = "fizhub"
	config.MQTT.Password = "fizpassword"
	config.NFC.PowerTimeout = Duration{30 * time.Second}
	config.Power.IdleTimeout = Duration{5 * time.Minute}
	config.Power.DeepSleepDelay = Duration{10 * time.Minute}
	config.Audio = audio.DefaultConfig()
	return config
}

func NewApplication(config Config) *Application {
	log.Println("Initializing FizHub application...")
	app := &Application{
		config: config,
		router: mux.NewRouter(),
	}

	// Initialize components
	log.Println("Initializing NFC reader...")
	app.nfcReader = nfc.NewReader(nfc.Config{
		PowerTimeout: config.NFC.PowerTimeout.Duration,
	})

	log.Println("Initializing LED controller...")
	app.ledCtrl = led.NewController()

	log.Println("Initializing power manager...")
	app.powerMgr = power.NewManager(power.Config{
		IdleTimeout:    config.Power.IdleTimeout.Duration,
		DeepSleepDelay: config.Power.DeepSleepDelay.Duration,
	})

	log.Println("Initializing state manager...")
	app.stateMgr = state.NewManager()

	log.Println("Initializing audio recorder...")
	app.recorder = audio.NewRecorder(config.Audio)

	log.Println("Initializing network client...")
	app.client = network.NewClient(network.ClientConfig{
		BaseURL: config.Cursive.URL,
		Timeout: config.Cursive.Timeout.Duration,
	})

	log.Println("Initializing MQTT broker...")
	app.mqttBroker = network.NewMQTTBroker(network.MQTTConfig{
		Port:     config.MQTT.Port,
		Username: config.MQTT.Username,
		Password: config.MQTT.Password,
	})

	return app
}

func (app *Application) Start(ctx context.Context) error {
	log.Println("Starting FizHub components...")
	
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	if err := app.initializeComponents(ctx); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	// Set up HTTP server
	app.setupRoutes()
	app.server = &http.Server{
		Addr:    ":" + app.config.Server.Port,
		Handler: app.router,
	}

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s...", app.config.Server.Port)
	go func() {
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	log.Printf("HTTP server is running on port %s", app.config.Server.Port)

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down...")
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	}

	return app.Shutdown()
}

func (app *Application) initializeComponents(ctx context.Context) error {
	log.Println("Starting LED controller...")
	if err := app.ledCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LED controller: %w", err)
	}

	log.Println("Starting NFC reader...")
	if err := app.nfcReader.Start(ctx); err != nil {
		return fmt.Errorf("failed to start NFC reader: %w", err)
	}

	log.Println("Starting power manager...")
	if err := app.powerMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start power manager: %w", err)
	}

	log.Println("Starting state manager...")
	if err := app.stateMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start state manager: %w", err)
	}

	log.Println("Starting audio recorder...")
	if err := app.recorder.Start(ctx); err != nil {
		return fmt.Errorf("failed to start audio recorder: %w", err)
	}

	log.Println("Starting MQTT broker...")
	if err := app.mqttBroker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MQTT broker: %w", err)
	}

	log.Println("Setting up component interactions...")
	app.setupComponentInteractions()

	return nil
}

func (app *Application) setupComponentInteractions() {
	// Handle NFC tap events from local reader
	app.nfcReader.SetTapHandler(func(uid string) error {
		log.Printf("NFC tap detected: %s", uid)
		app.powerMgr.RecordActivity()
		return app.stateMgr.HandleEvent(state.EventNFCTap, uid)
	})

	// Handle NFC tap events from remote readers
	app.mqttBroker.SetUIDHandler(func(msg network.UIDMessage) {
		log.Printf("Received UID from device %s: %s", msg.DeviceID, msg.UID)
		app.stateMgr.HandleEvent(state.EventNFCTap, msg.UID)
	})

	// Handle state changes
	app.stateMgr.Subscribe(state.PhaseValidating, func(phase state.Phase) {
		log.Println("Validating UIDs...")
		app.ledCtrl.SetState(led.StateWaiting)
		uids := app.stateMgr.GetCollectedUIDs()
		go app.validateUIDs(uids)
	})

	app.stateMgr.Subscribe(state.PhaseRecordingMessage, func(phase state.Phase) {
		log.Println("Starting message recording...")
		app.ledCtrl.SetState(led.StateSuccess)
		app.recorder.StartRecording()
	})

	// Handle power state changes
	app.powerMgr.SetOnStateChange(func(powerState power.State) {
		log.Printf("Power state changed to: %v", powerState)
		switch powerState {
		case power.StateDeepSleep:
			app.ledCtrl.SetState(led.StateOff)
		case power.StateActive:
			app.ledCtrl.SetState(led.StateIdle)
		}
	})

	// Handle recording state changes
	app.recorder.SetOnStateChange(func(recState audio.State) {
		log.Printf("Recording state changed to: %v", recState)
		if recState == audio.StateFinished {
			app.stateMgr.HandleEvent(state.EventRecordingComplete, nil)
		}
	})
}

func (app *Application) setupRoutes() {
	log.Println("Setting up HTTP routes...")
	app.router.HandleFunc("/api/receive_uid", app.handleReceiveUID).Methods("POST")
	app.router.HandleFunc("/api/status", app.handleStatus).Methods("GET")
	app.router.HandleFunc("/api/devices", app.handleDevices).Methods("GET")
}

func (app *Application) handleReceiveUID(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID string `json:"uid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Invalid request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received UID: %s", payload.UID)
	if err := app.stateMgr.HandleEvent(state.EventNFCTap, payload.UID); err != nil {
		log.Printf("Error handling UID: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *Application) handleStatus(w http.ResponseWriter, r *http.Request) {
	log.Println("Status request received")
	status := struct {
		Phase        state.Phase  `json:"phase"`
		PowerState   power.State  `json:"power_state"`
		UIDs         []string     `json:"uids"`
		BondID       string       `json:"bond_id,omitempty"`
		LastActivity time.Time    `json:"last_activity"`
	}{
		Phase:        app.stateMgr.GetPhase(),
		PowerState:   app.powerMgr.GetState(),
		UIDs:         app.stateMgr.GetCollectedUIDs(),
		BondID:       app.stateMgr.GetBondID(),
		LastActivity: app.powerMgr.GetLastActivity(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding status response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (app *Application) handleDevices(w http.ResponseWriter, r *http.Request) {
	log.Println("Devices request received")
	devices := app.mqttBroker.GetDevices()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(devices); err != nil {
		log.Printf("Error encoding devices response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (app *Application) validateUIDs(uids []string) {
	log.Printf("Validating UIDs: %v", uids)
	ctx := context.Background()
	resp, err := app.client.ValidateUIDs(ctx, uids)
	if err != nil {
		log.Printf("UID validation error: %v", err)
		app.stateMgr.HandleEvent(state.EventError, err)
		return
	}

	if resp.Valid {
		log.Printf("UIDs validated successfully: %v", resp.Accounts)
		app.stateMgr.HandleEvent(state.EventUIDValidated, resp.Accounts)
	} else {
		log.Printf("UID validation failed: %s", resp.Reason)
		app.stateMgr.HandleEvent(state.EventError, errors.New(resp.Reason))
	}
}

func (app *Application) Shutdown() error {
	log.Println("Initiating graceful shutdown...")
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	log.Println("Shutting down HTTP server...")
	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop components
	log.Println("Stopping NFC reader...")
	app.nfcReader.Stop()
	
	log.Println("Stopping LED controller...")
	app.ledCtrl.Stop()
	
	log.Println("Stopping power manager...")
	app.powerMgr.Stop()
	
	log.Println("Stopping audio recorder...")
	app.recorder.StopRecording()

	log.Println("Stopping MQTT broker...")
	app.mqttBroker.Stop()

	log.Println("Shutdown complete")
	return nil
}

// Run starts the FizHub application
func Run() error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	app := NewApplication(config)
	ctx := context.Background()

	return app.Start(ctx)
}
