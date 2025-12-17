package balena

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type SupervisorClient struct {
	Address string
	APIKey  string
	Client  *http.Client
}

type DeviceState struct {
	Status           string  `json:"status"`
	UpdatePending    bool    `json:"update_pending"`
	DownloadProgress float64 `json:"download_progress"`
	OSVersion        string  `json:"os_version"`
	MacAddress       string  `json:"mac_address"`
}

func NewSupervisorClient() (*SupervisorClient, error) {
	addr := os.Getenv("BALENA_SUPERVISOR_ADDRESS")
	key := os.Getenv("BALENA_SUPERVISOR_API_KEY")

	if addr == "" || key == "" {
		return nil, fmt.Errorf("BALENA_SUPERVISOR_ADDRESS and BALENA_SUPERVISOR_API_KEY must be set")
	}

	return &SupervisorClient{
		Address: addr,
		APIKey:  key,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (c *SupervisorClient) GetState() (*DeviceState, error) {
	url := fmt.Sprintf("%s/v2/state/status?apikey=%s", c.Address, c.APIKey)
	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get supervisor state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supervisor returned status %d", resp.StatusCode)
	}

	var state DeviceState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode supervisor response: %w", err)
	}

	return &state, nil
}
