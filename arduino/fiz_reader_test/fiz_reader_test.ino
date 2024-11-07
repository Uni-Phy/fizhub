#include <ESP8266WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>

// WiFi credentials
const char* ssid = "YourWiFiSSID";
const char* password = "YourWiFiPassword";

// MQTT Broker settings
const char* mqtt_server = "fiznode.local";  // FizHub hostname
const int mqtt_port = 1883;
const char* mqtt_client_id = "fiz_reader_test";

// MQTT topics
const char* topic_register = "fiz/register";
const char* topic_status = "fiz/status";
const char* topic_uid = "fiz/uid";

// Device info
const char* device_id = "FIZR001";  // Unique device ID
const char* device_type = "reader";
const char* firmware_version = "1.0.0";

// Global objects
WiFiClient espClient;
PubSubClient client(espClient);
unsigned long lastMsg = 0;
char msg[100];

void setup_wifi() {
  delay(10);
  Serial.println();
  Serial.print("Connecting to ");
  Serial.println(ssid);

  WiFi.begin(ssid, password);

  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }

  Serial.println("");
  Serial.println("WiFi connected");
  Serial.println("IP address: ");
  Serial.println(WiFi.localIP());
}

void callback(char* topic, byte* payload, unsigned int length) {
  Serial.print("Message arrived [");
  Serial.print(topic);
  Serial.print("] ");
  
  char message[length + 1];
  for (int i = 0; i < length; i++) {
    message[i] = (char)payload[i];
    Serial.print((char)payload[i]);
  }
  message[length] = '\0';
  Serial.println();

  // Handle different topics
  if (strcmp(topic, topic_status) == 0) {
    // Handle status requests
    sendDeviceStatus();
  }
}

void reconnect() {
  while (!client.connected()) {
    Serial.print("Attempting MQTT connection...");
    if (client.connect(mqtt_client_id)) {
      Serial.println("connected");
      
      // Subscribe to topics
      client.subscribe(topic_status);
      
      // Register device
      registerDevice();
    } else {
      Serial.print("failed, rc=");
      Serial.print(client.state());
      Serial.println(" retrying in 5 seconds");
      delay(5000);
    }
  }
}

void registerDevice() {
  StaticJsonDocument<200> doc;
  doc["device_id"] = device_id;
  doc["type"] = device_type;
  doc["firmware"] = firmware_version;
  doc["ip"] = WiFi.localIP().toString();

  char buffer[200];
  serializeJson(doc, buffer);
  client.publish(topic_register, buffer);
  Serial.println("Device registered");
}

void sendDeviceStatus() {
  StaticJsonDocument<200> doc;
  doc["device_id"] = device_id;
  doc["status"] = "online";
  doc["uptime"] = millis() / 1000;
  doc["rssi"] = WiFi.RSSI();

  char buffer[200];
  serializeJson(doc, buffer);
  client.publish(topic_status, buffer);
}

void simulateNFCTap() {
  // Simulate reading an NFC tag
  const char* test_uid = "ec586341127a6414";
  
  StaticJsonDocument<200> doc;
  doc["device_id"] = device_id;
  doc["uid"] = test_uid;
  doc["timestamp"] = millis();

  char buffer[200];
  serializeJson(doc, buffer);
  client.publish(topic_uid, buffer);
  Serial.println("NFC tap simulated");
}

void setup() {
  Serial.begin(115200);
  
  // Setup WiFi
  setup_wifi();
  
  // Setup MQTT
  client.setServer(mqtt_server, mqtt_port);
  client.setCallback(callback);
}

void loop() {
  if (!client.connected()) {
    reconnect();
  }
  client.loop();

  unsigned long now = millis();
  if (now - lastMsg > 10000) {  // Every 10 seconds
    lastMsg = now;
    
    // Send status update
    sendDeviceStatus();
    
    // Simulate NFC tap every 10 seconds
    simulateNFCTap();
  }
}
