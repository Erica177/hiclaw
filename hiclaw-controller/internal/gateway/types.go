package gateway

// Config holds connection parameters for an AI gateway.
type Config struct {
	ConsoleURL    string // gateway console API, e.g. http://127.0.0.1:8001
	AdminUser     string // console login username
	AdminPassword string // console login password
}

// ConsumerRequest describes a gateway consumer to create.
type ConsumerRequest struct {
	Name          string // consumer name, e.g. "worker-alice"
	CredentialKey string // API key for key-auth (self-hosted Higress)
	ConsumerID    string // platform-specific consumer ID (cloud APIG, optional)
}

// ConsumerResult holds the result of an EnsureConsumer call.
type ConsumerResult struct {
	Status     string // "created" or "exists"
	APIKey     string // the active API key
	ConsumerID string // platform-specific consumer ID (cloud only)
}

// AIRoute represents an AI route in the gateway.
type AIRoute struct {
	Name             string   `json:"name"`
	AllowedConsumers []string `json:"allowedConsumers,omitempty"`
}

// MCPServerAuth describes MCP server authorization state.
type MCPServerAuth struct {
	Name             string   `json:"name"`
	AllowedConsumers []string `json:"allowedConsumers,omitempty"`
}

// PortExposeRequest describes a port to expose through the gateway.
type PortExposeRequest struct {
	WorkerName  string // worker identifier
	ServiceHost string // DNS hostname of the service
	Port        int    // port number to expose
	Domain      string // domain name to bind
}
