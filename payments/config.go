package payments

import (
	"fmt"
	"os"
)

// Config holds credentials and endpoints for the Asaas API.
type Config struct {
	APIURL   string
	APIToken string
}

// LoadConfigFromEnv builds a Config using environment variables.
func LoadConfigFromEnv() (Config, error) {
	apiURL := os.Getenv("ASAAS_API_URL")
	token := os.Getenv("ASAAS_API_TOKEN")
	if apiURL == "" {
		return Config{}, fmt.Errorf("ASAAS_API_URL n\u00e3o est\u00e1 definida")
	}
	if token == "" {
		return Config{}, fmt.Errorf("ASAAS_API_TOKEN n\u00e3o est\u00e1 definido")
	}
	return Config{APIURL: apiURL, APIToken: token}, nil
}
