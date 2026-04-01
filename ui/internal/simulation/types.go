package simulation

// ServiceType describes an available algorithm/service discovered on disk.
type ServiceType struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SchemaRaw string `json:"schemaRaw"`
}

// DeployedService represents a user-selected service assigned to a UAV,
// including its resolved runtime configuration.
type DeployedService struct {
	InstanceId   string                 `json:"instanceId"`
	ServiceId    string                 `json:"serviceId"`
	ServiceTitle string                 `json:"serviceTitle"`
	Config       map[string]interface{} `json:"config"`
}

// UAV groups a UAV identifier with its set of deployed services.
type UAV struct {
	ID       string            `json:"id"`
	Services []DeployedService `json:"services"`
}
