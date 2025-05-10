// internal/bruteforce/apiclient.go
package bruteforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"btcforce/internal/wallet"
	"btcforce/pkg/config"
)

type APIClient struct {
	client     *http.Client
	url        string
	maxRetries int
}

type APIRequest struct {
	Address    string `json:"address"`
	WIF        string `json:"wif"`
	PrivateKey string `json:"private_key"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Balance string `json:"balance,omitempty"`
}

func NewAPIClient(cfg *config.Config) *APIClient {
	return &APIClient{
		client: &http.Client{
			Timeout: time.Duration(cfg.APITimeout) * time.Millisecond,
		},
		url:        cfg.APIURL,
		maxRetries: cfg.MaxRetries,
	}
}

func (c *APIClient) CheckAddress(wallet *wallet.WalletInfo) (bool, string) {
	request := APIRequest{
		Address:    wallet.Address,
		WIF:        wallet.WIF,
		PrivateKey: wallet.PrivateKey,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return false, ""
	}

	var lastErr error
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		resp, err := c.client.Post(c.url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = err
			backoff := time.Duration(300*attempt) * time.Millisecond
			time.Sleep(backoff)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var apiResp APIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err == nil {
				return apiResp.Success, apiResp.Balance
			}
		}

		backoff := time.Duration(300*attempt) * time.Millisecond
		time.Sleep(backoff)
	}

	if lastErr != nil {
		fmt.Printf("API check failed after %d attempts: %v\n", c.maxRetries, lastErr)
	}

	return false, ""
}
