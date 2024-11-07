# Fiz Reader Test Client

This Arduino sketch provides a test client for simulating a Fiz Reader device. It connects to WiFi, establishes an MQTT connection with the FizHub, and simulates NFC tag readings.

## Requirements

### Hardware
- Arduino Uno
- ESP8266 WiFi Module
- LED (optional, for status indication)

### Libraries
- ESP8266WiFi
- PubSubClient
- ArduinoJson

Install these libraries through the Arduino IDE Library Manager:
1. Tools -> Manage Libraries
2. Search for each library name
3. Click Install

## Configuration

Edit the following constants in the sketch:

```cpp
// WiFi credentials
const char* ssid = "YourWiFiSSID";
const char* password = "YourWiFiPassword";

// MQTT Broker settings
const char* mqtt_server = "fiznode.local";
const int mqtt_port = 1883;

// Device info
const char* device_id = "FIZR001";  // Change this for each reader
```

## MQTT Topics

The test client uses the following MQTT topics:

1. `fiz/register` - Device registration
   ```json
   {
     "device_id": "FIZR001",
     "type": "reader",
     "firmware": "1.0.0",
     "ip": "192.168.1.100"
   }
   ```

2. `fiz/status` - Device status updates
   ```json
   {
     "device_id": "FIZR001",
     "status": "online",
     "uptime": 123,
     "rssi": -70
   }
   ```

3. `fiz/uid` - NFC tag readings
   ```json
   {
     "device_id": "FIZR001",
     "uid": "ec586341127a6414",
     "timestamp": 1234567890
   }
   ```

## Testing

1. Upload the sketch to your Arduino
2. Open Serial Monitor (Tools -> Serial Monitor)
3. Set baud rate to 115200
4. Watch the connection and registration process
5. Every 10 seconds, the device will:
   - Send a status update
   - Simulate an NFC tag reading

## Debugging

Monitor the serial output for:
- WiFi connection status
- MQTT connection attempts
- Device registration
- Message publishing
- Incoming messages

Example serial output:
```
Connecting to WiFi...
WiFi connected
IP address: 192.168.1.100
Attempting MQTT connection...connected
Device registered
Message arrived [fiz/status] {"device_id":"FIZR001","status":"online"}
NFC tap simulated
```

## LED Status Indicators

The built-in LED indicates:
- Fast blink: Connecting to WiFi
- Slow blink: Connected, normal operation
- Solid: NFC tag being read
- Off: Error state

## Customization

To modify the test behavior:
1. Adjust the simulation interval in `loop()`
2. Change the test UID in `simulateNFCTap()`
3. Add additional sensors or indicators
4. Modify the status payload structure

## Troubleshooting

1. If WiFi won't connect:
   - Verify SSID and password
   - Check WiFi signal strength
   - Ensure ESP8266 is properly powered

2. If MQTT won't connect:
   - Verify FizHub is running
   - Check MQTT broker address
   - Ensure port 1883 is accessible

3. If messages aren't received:
   - Check topic subscriptions
   - Verify JSON payload format
   - Monitor MQTT broker logs
