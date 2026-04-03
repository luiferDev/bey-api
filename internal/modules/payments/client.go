package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"bey/internal/config"
)

type WompiClient struct {
	httpClient *http.Client
	baseURL    string
	publicKey  string
	privateKey string
}

func NewWompiClient(cfg *config.WompiConfig) *WompiClient {
	baseURL := cfg.GetBaseURL()
	return &WompiClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		publicKey:  cfg.PublicKey,
		privateKey: cfg.PrivateKey,
	}
}

func (c *WompiClient) CreateTransaction(amount int64, currency, token, reference, redirectURL string) (*WompiTransactionResponse, error) {
	body := map[string]interface{}{
		"amount_in_cents": amount,
		"currency":        currency,
		"token":           token,
		"reference":       reference,
	}

	if redirectURL != "" {
		body["redirect_url"] = redirectURL
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", c.baseURL+"/transactions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var wompiErr WompiError
		if err := json.NewDecoder(resp.Body).Decode(&wompiErr); err == nil {
			return nil, fmt.Errorf("wompi error: %s - %s", wompiErr.Error, wompiErr.Message)
		}
		return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
	}

	var result WompiTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *WompiClient) GetTransaction(transactionID string) (*WompiTransactionResponse, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", c.baseURL+"/transactions/"+transactionID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var wompiErr WompiError
		if err := json.NewDecoder(resp.Body).Decode(&wompiErr); err == nil {
			return nil, fmt.Errorf("wompi error: %s - %s", wompiErr.Error, wompiErr.Message)
		}
		return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
	}

	var result WompiTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *WompiClient) VoidTransaction(transactionID string) (*WompiTransactionResponse, error) {
	body := map[string]interface{}{
		"status": "VOIDED",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "PATCH", c.baseURL+"/transactions/"+transactionID, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var wompiErr WompiError
		if err := json.NewDecoder(resp.Body).Decode(&wompiErr); err == nil {
			return nil, fmt.Errorf("wompi error: %s - %s", wompiErr.Error, wompiErr.Message)
		}
		return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
	}

	var result WompiTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *WompiClient) CreatePaymentLink(amount int64, currency, description, reference, redirectURL string, singleUse bool, expiresAt *time.Time) (*WompiPaymentLinkResponse, error) {
	body := map[string]interface{}{
		"amount_in_cents": amount,
		"currency":        currency,
		"title":           description,
		"description":     description,
		"reference":       reference,
		"single_use":      singleUse,
	}

	if redirectURL != "" {
		body["redirect_url"] = redirectURL
	}

	if expiresAt != nil {
		body["expires_at"] = expiresAt.Format(time.RFC3339)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", c.baseURL+"/payment_links", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Wompi create payment link error: %s", string(respBody))
		var wompiErr WompiError
		if err := json.Unmarshal(respBody, &wompiErr); err == nil {
			return nil, fmt.Errorf("wompi error: %s - %s", wompiErr.Error, wompiErr.Message)
		}
		return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
	}

	var result WompiPaymentLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *WompiClient) GetPaymentLink(linkID string) (*WompiPaymentLinkResponse, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", c.baseURL+"/payment_links/"+linkID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var wompiErr WompiError
		if err := json.NewDecoder(resp.Body).Decode(&wompiErr); err == nil {
			return nil, fmt.Errorf("wompi error: %s - %s", wompiErr.Error, wompiErr.Message)
		}
		return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
	}

	var result WompiPaymentLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *WompiClient) UpdatePaymentLink(linkID string, status string) (*WompiPaymentLinkResponse, error) {
	body := map[string]interface{}{
		"status": status,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "PATCH", c.baseURL+"/payment_links/"+linkID, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.privateKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var wompiErr WompiError
		if err := json.NewDecoder(resp.Body).Decode(&wompiErr); err == nil {
			return nil, fmt.Errorf("wompi error: %s - %s", wompiErr.Error, wompiErr.Message)
		}
		return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
	}

	var result WompiPaymentLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
