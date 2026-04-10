#include <ESP8266WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include <ESP8266HTTPClient.h>

WiFiClient espClient;
PubSubClient mqtt_client(espClient);

// WiFi settings
const char *ssid = "SIAPA";             // Replace with your WiFi name
const char *password = "absendulubang";   // Replace with your WiFi password

// Set your Static IP address
IPAddress local_IP(192, 168, 14, 11);
IPAddress gateway(192, 168, 14, 1);
IPAddress subnet(255, 255, 0, 0);
IPAddress primaryDNS(8, 8, 8, 8);

// MQTT Broker settings
const char *mqtt_broker = "10.20.30.26";  // EMQX broker endpoint
const char *mqtt_topic = "hmstt_channel/hmstt/#";     // MQTT topic
const char *mqtt_username = "admin";  // MQTT username for authentication
const char *mqtt_password = "sitolol";  // MQTT password for authentication
const int mqtt_port = 1883;  // MQTT port (TCP)
const char *client_id = "esp8266-client-electric1";

const char *http_base_url = "http://10.20.30.28:8080";
const char *http_bearer_token = "replace-with-v1-bearer-token";

const char *STATE_ON = "on";
const char *STATE_OFF = "off";

const char *MQTT_PREFIX_TOPIC = "hmstt_channel/hmstt";

// -- multiple switches (expandable) -------------------------------------------------
const int NUM_SWITCHES = 8;

const char *NAME_SWITCHES[NUM_SWITCHES] = {
    "switch/server_1",
    "switch/server_2",
    "switch/server_3",
    "switch/server_4",
    "switch/server_5",
    "switch/server_6",
    "switch/server_7",
    "switch/server_8",
};

bool STATE_SWITCHES[NUM_SWITCHES] = {true, true, true, true, true, true, true, true};

// Pins for each switch. Adjust these pins as needed for your hardware.
const int PIN_SWITCHES[NUM_SWITCHES] = { D8, D7, D6, D5, D4, D3, D2, D0 };
const int KEY_SWITCH = D1;

// Per-switch wiring: true = active-low (LOW means ON), false = active-high (HIGH means ON)
const bool ACTIVE_LOW[NUM_SWITCHES] = { true, true, true, true, true, true, true, true };

const unsigned long defaultRequestInterval = 300000UL;
unsigned long requestInterval = defaultRequestInterval;
unsigned long lastRequestTime = 0;
const unsigned long maxBackoffInterval = 15000UL;
const unsigned long backoffInterval = 5000UL;
bool lastStatusGetPersistentData = false;

const char* extractKeyName(const char* name) {
    const char* separator = strchr(name, '/');
    if (separator == nullptr || *(separator + 1) == '\0') {
        return name;
    }
    return separator + 1;
}

void syncPersistentData() {
    WiFiClient espClient2;
    HTTPClient httpClient;

    Serial.println("Getting persistent data...");
    String fullUrl = String(http_base_url) + "/v1/states/switch/batch";
    for (int i = 0; i < NUM_SWITCHES; ++i) {
        fullUrl += (i == 0 ? "?key=" : "&key=");
        fullUrl += extractKeyName(NAME_SWITCHES[i]);
    }

    httpClient.begin(espClient2, fullUrl);
    httpClient.addHeader("Authorization", String("Bearer ") + http_bearer_token);
    int httpResponseCode = httpClient.GET();
    if (httpResponseCode == HTTP_CODE_OK) {
        String payload = httpClient.getString();
        DynamicJsonDocument doc(4096);
        DeserializationError error = deserializeJson(doc, payload);
        if (error) {
            Serial.print("Failed to parse persistent data: ");
            Serial.println(error.c_str());
            lastStatusGetPersistentData = false;
            httpClient.end();
            return;
        }

        JsonArray data = doc["data"].as<JsonArray>();
        for (JsonObject item : data) {
            const char* key = item["key"] | "";
            const char* value = item["value"] | "";
            if (strlen(key) == 0 || strlen(value) == 0) {
                continue;
            }
            String fullName = String("switch/") + key;
            setStateData(fullName.c_str(), value);
        }

        lastStatusGetPersistentData = true;
        requestInterval = defaultRequestInterval;
    } else {
        Serial.print("Error on HTTP request: ");
        Serial.println(httpResponseCode);
        if (httpResponseCode < 0) {
            if (lastStatusGetPersistentData) {
                requestInterval = backoffInterval;
            } else {
                requestInterval = minUL(requestInterval + backoffInterval, maxBackoffInterval);
            }
        }
        lastStatusGetPersistentData = false;
    }
    httpClient.end();
}

unsigned long minUL(unsigned long a, unsigned long b) {
    return (a < b) ? a : b;
}

void setStateData(const char* name, const char* state) {
    char full_topic[128];

    // Normalize incoming state string: trim + toLower
    String sstate = String(state);
    sstate.trim();
    sstate.toLowerCase();

    // Try to match against every configured switch. Accept either the bare name
    // (e.g. "switch/server_1") or the full topic (e.g. "hmstt_channel/hmstt/switch/server_1").
    for (int i = 0; i < NUM_SWITCHES; ++i) {
        snprintf(full_topic, sizeof(full_topic), "%s/%s", MQTT_PREFIX_TOPIC, NAME_SWITCHES[i]);
        if (strcmp(name, full_topic) == 0 || strcmp(name, NAME_SWITCHES[i]) == 0) {
            bool on = (sstate == String(STATE_ON));
            STATE_SWITCHES[i] = on;

            // Write pin according to wiring (active-low vs active-high)
            if (ACTIVE_LOW[i]) {
                // LOW = ON, HIGH = OFF
                digitalWrite(PIN_SWITCHES[i], on ? LOW : HIGH);
            } else {
                // HIGH = ON, LOW = OFF
                digitalWrite(PIN_SWITCHES[i], on ? HIGH : LOW);
            }

            // Log what we set (print actual level written for clarity)
            const char* levelWritten = (ACTIVE_LOW[i] ? (on ? "LOW" : "HIGH") : (on ? "HIGH" : "LOW"));
            Serial.printf("Set %s (match %s) -> state='%s' on=%d -> wrote %s\n", NAME_SWITCHES[i], name, sstate.c_str(), on, levelWritten);
            return;
        }
    }
    // If we reach here, no matching switch was found
    Serial.print("No matching switch for name/topic: ");
    Serial.println(name);
}

void connectToWiFi() {
  WiFi.begin(ssid, password);
  WiFi.config(local_IP, gateway, subnet, primaryDNS);
  Serial.print("Connecting to WiFi");
  while (WiFi.status() != WL_CONNECTED) {
      delay(500);
      Serial.print(".");
  }
  Serial.println("");
  Serial.println("WiFi connected.");
  Serial.println("IP address: ");
  Serial.println(WiFi.localIP());
}

bool connectToMQTTBroker() {
    while (!mqtt_client.connected()) {
        Serial.printf("Connecting to MQTT Broker as %s.....\n", client_id);
        if (mqtt_client.connect(client_id, mqtt_username, mqtt_password)) {
            Serial.println("Connected to MQTT broker");
            mqtt_client.subscribe(mqtt_topic);
            return true;
        } else {
            Serial.print("Failed to connect to MQTT broker, rc=");
            Serial.print(mqtt_client.state());
            Serial.println(" try again in 5 seconds");
            delay(5000);
        }
    }

    return false;
}

void mqttCallback(char *topic, byte *payload, unsigned int length) {
    Serial.print("Message received on topic: ");
    Serial.println(topic);
    Serial.print("Message:");
    char message[length + 1];
    for (unsigned int i = 0; i < length; i++) {
        message[i] = (char) payload[i];
    }
    message[length] = '\0';
    Serial.println(message);

    setStateData(topic, message);

    Serial.println("-----------------------");
}

void setup() {
    Serial.begin(115200);
    pinMode(KEY_SWITCH, OUTPUT);

    connectToWiFi();
    mqtt_client.setServer(mqtt_broker, mqtt_port);
    mqtt_client.setCallback(mqttCallback);

    // initialize pins for all switches and set a safe default (OFF)
    for (int i = 0; i < NUM_SWITCHES; ++i) {
        pinMode(PIN_SWITCHES[i], OUTPUT);
        // Set safe default OFF depending on wiring
        if (ACTIVE_LOW[i]) {
            // active-low: HIGH means OFF
            digitalWrite(PIN_SWITCHES[i], HIGH);
        } else {
            // active-high: LOW means OFF
            digitalWrite(PIN_SWITCHES[i], LOW);
        }
    }

    syncPersistentData();
    connectToMQTTBroker();

    digitalWrite(KEY_SWITCH, HIGH);
}

void loop() {
    if (!mqtt_client.connected()) {
        if (connectToMQTTBroker()) {
            syncPersistentData();
        }
    }
    mqtt_client.loop();

    unsigned long currentTime = millis();
    if (currentTime - lastRequestTime >= requestInterval) {
        Serial.println("Periodic sync interval reached. Fetching persistent data...");
        syncPersistentData();
        lastRequestTime = currentTime;
    }
}
