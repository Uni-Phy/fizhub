package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fizhub/internal/audio"
	"github.com/fizhub/internal/led"
	"github.com/fizhub/internal/network"
	"github.com/fizhub/internal/nfc"
	"github.com/fizhub/internal/power"
	"github.com/fizhub/internal/state"
	"github.com/gorilla/mux"
)

type Config struct {
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
	Cursive struct {
		URL     string        `json:"url"`
		Timeout time.Duration `json:"timeout"`
	} `json:"cursive"`
	NFC struct {
		PowerTimeout time.Duration `json:"power_timeout"`
	} `json:"nfc"`
	Power struct {
		IdleTimeout    time.Duration `json:"idle_timeout"`
		DeepSleepDelay time.Duration `json:"deep_sleep_delay"`
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
}

func loadConfig() (Config, error) {
	var config Config
	file, err := os.Open("configs/config.json")
	if err != nil {
		// Use default configuration if file doesn't exist
		return getDefaultConfig(), nil
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func getDefaultConfig() Config {
	config := Config{}
	config.Server.Port = "8080"
	config.Cursive.URL = "http://cursive-server/api/validate_uids"
	config.Cursive.Timeout = 30 * time.Second
	config.NFC.PowerTimeout = 30 * time.Second
	config.Power.IdleTimeout = 5 * time.Minute
	config.Power.DeepSleepDelay = 10 * time.Minute
	config.Audio = audio.DefaultConfig()
	return config
}

func NewApplication(config Config) *Application {
	app := &Application{
		config: config,
		router: mux.NewRouter(),
	}

	// Initialize components
	app.nfcReader = nfc.NewReader(nfc.Config{
		PowerTimeout: config.NFC.PowerTimeout,
	})

	app.ledCtrl = led.NewController()

	app.powerMgr = power.NewManager(power.Config{
		IdleTimeout:    config.Power.IdleTimeout,
		DeepSleepDelay: config.Power.DeepSleepDelay,
	})

	app.stateMgr = state.NewManager()

	app.recorder = audio.NewRecorder(config.Audio)

	app.client = network.NewClient(network.ClientConfig{
		BaseURL: config.Cursive.URL,
		Timeout: config.Cursive.Timeout,
	})

	return app
}

func (app *Application) Start(ctx context.Context) error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	if err := app.initializeComponents(ctx); err != nil {
		return err
	}

	// Set up HTTP server
	app.setupRoutes()
	app.server = &http.Server{
		Addr:    ":" + app.config.Server.Port,
		Handler: app.router,
	}

	// Start HTTP server
	go func() {
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	}

	return app.Shutdown()
}

func (app *Application) initializeComponents(ctx context.Context) error {
	// Start LED controller
	if err := app.ledCtrl.Start(ctx); err != nil {
		return err
	}

	// Start NFC reader
	if err := app.nfcReader.Start(ctx); err != nil {
		return err
	}

	// Start power manager
	if err := app.powerMgr.Start(ctx); err != nil {
		return err
	}

	// Start state manager
	if err := app.stateMgr.Start(ctx); err != nil {
		return err
	}

	// Start audio recorder
	if err := app.recorder.Start(ctx); err != nil {
		return err
	}

	// Set up component interactions
	app.setupComponentInteractions()

	return nil
}

func (app *Application) setupComponentInteractions() {
	// Handle NFC tap events
	app.nfcReader.SetTapHandler(func(uid string) error {
		app.powerMgr.RecordActivity()
		return app.stateMgr.HandleEvent(state.EventNFCTap, uid)
	})

	// Handle state changes
	app.stateMgr.Subscribe(state.PhaseValidating, func(phase state.Phase) {
		app.ledCtrl.SetState(led.StateWaiting)
		uids := app.stateMgr.GetCollectedUIDs()
		go app.validateUIDs(uids)
	})

	app.stateMgr.Subscribe(state.PhaseRecordingMessage, func(phase state.Phase) {
		app.ledCtrl.SetState(led.StateSuccess)
		app.recorder.StartRecording()
	})

	// Handle power state changes
	app.powerMgr.SetOnStateChange(func(powerState power.State) {
		switch powerState {
		case power.StateDeepSleep:
			app.ledCtrl.SetState(led.StateOff)
		case power.StateActive:
			app.ledCtrl.SetState(led.StateIdle)
		}
	})

	// Handle recording state changes
	app.recorder.SetOnStateChange(func(recState audio.State) {
		if recState == audio.StateFinished {
			app.stateMgr.HandleEvent(state.EventRecordingComplete, nil)
		}
	})
}

func (app *Application) setupRoutes() {
	app.router.HandleFunc("/api/receive_uid", app.handleReceiveUID).Methods("POST")
	app.router.HandleFunc("/api/status", app.handleStatus).Methods("GET")
}

func (app *Application) handleReceiveUID(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID string `json:"uid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := app.stateMgr.HandleEvent(state.EventNFCTap, payload.UID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *Application) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Phase       state.Phase `json:"phase"`
		PowerState  power.State `json:"power_state"`
		UIDs        []string    `json:"uids"`
		BondID      string      `json:"bond_id,omitempty"`
		LastActivity time.Time  `json:"last_activity"`
	}{
		Phase:       app.stateMgr.GetPhase(),
		PowerState:  app.powerMgr.GetState(),
		UIDs:        app.stateMgr.GetCollectedUIDs(),
		BondID:      app.stateMgr.GetBondID(),
		LastActivity: app.powerMgr.GetLastActivity(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (app *Application) validateUIDs(uids []string) {
	ctx := context.Background()
	resp, err := app.client.ValidateUIDs(ctx, uids)
	if err != nil {
		app.stateMgr.HandleEvent(state.EventError, err)
		return
	}

	if resp.Valid {
		app.stateMgr.HandleEvent(state.EventUIDValidated, resp.Accounts)
	} else {
		app.stateMgr.HandleEvent(state.EventError, errors.New(resp.Reason))
	}
}

func (app *Application) Shutdown() error {
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop components
	app.nfcReader.Stop()
	app.ledCtrl.Stop()
	app.powerMgr.Stop()
	app.recorder.StopRecording()

	return nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	app := NewApplication(config)
	ctx := context.Background()

	if err := app.Start(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
