package domain

// AppConfig defines the connection parameters for the Local Broker and the Global Net Simulator
type AppConfig struct {
	BrokerIP   string `json:"broker_ip"`
	BrokerPort int    `json:"broker_port"`

	NetSimIP   string `json:"simulator_ip"`
	NetSimPort int    `json:"simulator_port"`

	// Topics to subscribe from Local Broker
	SubTelemetryTopic string `json:"sub_telemetry_topic"`
	SubMessagesTopic  string `json:"sub_messages_topic"`

	// Topics to publish to Local Broker (coming from NetSim)
	PubExternalTelemetryTopic string `json:"pub_ext_telemetry_topic"`
	PubExternalMessagesTopic  string `json:"pub_ext_messages_topic"`
}

// NetSimMessage envelope for sending/receiving data to the global network simulator
type NetSimMessage struct {
	Type    string                 `json:"type"` // "telemetry" or "message"
	Source  string                 `json:"source"`
	Payload map[string]interface{} `json:"payload"`
}
