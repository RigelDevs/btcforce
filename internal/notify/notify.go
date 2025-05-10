// internal/notify/notify.go
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"btcforce/pkg/config"
)

type WhatsAppPayload struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

func SendWhatsApp(message string, cfg *config.Config) error {
	payload := WhatsAppPayload{
		Phone:   cfg.NotifyPhone,
		Message: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(cfg.NotifyURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✅ WhatsApp notification sent to %s\n", cfg.NotifyPhone)
		return nil
	}

	return fmt.Errorf("failed to send notification: HTTP %d", resp.StatusCode)
}
