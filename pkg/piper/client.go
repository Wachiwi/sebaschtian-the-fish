package piper

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type PiperClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewPiperClient(baseURL string) *PiperClient {
	return &PiperClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

type SynthesizeRequest struct {
	Text string `json:"text"`
}

func (c *PiperClient) Synthesize(text string) ([]byte, error) {
	requestBody, err := json.Marshal(SynthesizeRequest{Text: text})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
