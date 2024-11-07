package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ReaderDevice represents a connected Fiz Reader
type ReaderDevice struct {
	DeviceID  string    `json:"device_id"`
	Type      string    `json:"type"`
	Firmware  string    `json:"firmware"`
	IP        string    `json:"ip"`
	LastSeen  time.Time `json:"last_seen"`
	Status    string    `json:"status"`
	RSSI      int       `json:"rssi"`
}

// UIDMessage represents an NFC tag read from a reader
type UIDMessage struct {
	DeviceID  string `json:"device_id"`
	UID       string `json:"uid"`
	Timestamp int64  `json:"timestamp"`
}

// MQTTBroker handles MQTT communication with Fiz Readers
type MQTTBroker struct {
	client     mqtt.Client
	devices    map[string]*ReaderDevice
	devicesMux sync.RWMutex
	uidHandler func(UIDMessage)
}

// MQTTConfig holds MQTT broker configuration
type MQTTConfig struct {
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewMQTTBroker creates a new MQTT broker instance
func NewMQTTBroker(config MQTTConfig) *MQTTBroker {
	broker := &MQTTBroker{
		devices: make(map[string]*ReaderDevice),
	}

	// Configure MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://localhost:%d", config.Port))
	opts.SetClientID("fizhub")
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetDefaultPublishHandler(broker.messageHandler)
	opts.SetOnConnectHandler(broker.connectHandler)
	opts.SetConnectionLostHandler(broker.connectionLostHandler)

	broker.client = mqtt.NewClient(opts)
	return broker
}

// Start initializes the MQTT broker
func (b *MQTTBroker) Start(ctx context.Context) error {
	log.Println("Starting MQTT broker...")
	
	if token := b.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	// Subscribe to topics
	topics := map[string]byte{
		"fiz/register": 1,
		"fiz/status":   1,
		"fiz/uid":      1,
	}

	for topic, qos := range topics {
		if token := b.client.Subscribe(topic, qos, nil); token.Wait() && token.Error() != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", topic, token.Error())
		}
		log.Printf("Subscribed to topic: %s", topic)
	}

	go b.monitorDevices(ctx)
	return nil
}

// Stop shuts down the MQTT broker
func (b *MQTTBroker) Stop() error {
	log.Println("Stopping MQTT broker...")
	b.client.Disconnect(250)
	return nil
}

// SetUIDHandler sets the callback for handling UID messages
func (b *MQTTBroker) SetUIDHandler(handler func(UIDMessage)) {
	b.uidHandler = handler
}

// messageHandler processes incoming MQTT messages
func (b *MQTTBroker) messageHandler(_ mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message on topic: %s", msg.Topic())

	switch msg.Topic() {
	case "fiz/register":
		var device ReaderDevice
		if err := json.Unmarshal(msg.Payload(), &device); err != nil {
			log.Printf("Error unmarshaling device registration: %v", err)
			return
		}
		b.registerDevice(&device)

	case "fiz/status":
		var status struct {
			DeviceID string `json:"device_id"`
			Status   string `json:"status"`
			RSSI     int    `json:"rssi"`
		}
		if err := json.Unmarshal(msg.Payload(), &status); err != nil {
			log.Printf("Error unmarshaling status update: %v", err)
			return
		}
		b.updateDeviceStatus(status.DeviceID, status.Status, status.RSSI)

	case "fiz/uid":
		var uidMsg UIDMessage
		if err := json.Unmarshal(msg.Payload(), &uidMsg); err != nil {
			log.Printf("Error unmarshaling UID message: %v", err)
			return
		}
		if b.uidHandler != nil {
			b.uidHandler(uidMsg)
		}
	}
}

// connectHandler is called when MQTT client connects
func (b *MQTTBroker) connectHandler(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

// connectionLostHandler is called when MQTT client loses connection
func (b *MQTTBroker) connectionLostHandler(client mqtt.Client, err error) {
	log.Printf("Connection lost to MQTT broker: %v", err)
}

// registerDevice registers a new reader device
func (b *MQTTBroker) registerDevice(device *ReaderDevice) {
	b.devicesMux.Lock()
	defer b.devicesMux.Unlock()

	device.LastSeen = time.Now()
	device.Status = "online"
	b.devices[device.DeviceID] = device
	log.Printf("Registered device: %s (%s)", device.DeviceID, device.IP)
}

// updateDeviceStatus updates a device's status
func (b *MQTTBroker) updateDeviceStatus(deviceID, status string, rssi int) {
	b.devicesMux.Lock()
	defer b.devicesMux.Unlock()

	if device, ok := b.devices[deviceID]; ok {
		device.Status = status
		device.RSSI = rssi
		device.LastSeen = time.Now()
	}
}

// GetDevices returns a list of all registered devices
func (b *MQTTBroker) GetDevices() []*ReaderDevice {
	b.devicesMux.RLock()
	defer b.devicesMux.RUnlock()

	devices := make([]*ReaderDevice, 0, len(b.devices))
	for _, device := range b.devices {
		devices = append(devices, device)
	}
	return devices
}

// monitorDevices checks for inactive devices
func (b *MQTTBroker) monitorDevices(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.checkInactiveDevices()
		}
	}
}

// checkInactiveDevices marks devices as offline if they haven't sent updates
func (b *MQTTBroker) checkInactiveDevices() {
	b.devicesMux.Lock()
	defer b.devicesMux.Unlock()

	now := time.Now()
	for _, device := range b.devices {
		if device.Status == "online" && now.Sub(device.LastSeen) > 60*time.Second {
			device.Status = "offline"
			log.Printf("Device %s marked as offline", device.DeviceID)
		}
	}
}
