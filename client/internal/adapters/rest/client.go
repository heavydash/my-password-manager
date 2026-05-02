package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL string
	Token   string
	client  *http.Client
}

type LoginResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
	UserID  string `json:"user_id"`
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *Client) SetToken(token string) {
	c.Token = token
}

// Login — реальный запрос на сервер
func (c *Client) Login(payload map[string]string) (*LoginResponse, error) {
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/login", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var loginResp LoginResponse

	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, err
	}

	return &loginResp, nil
}

func (c *Client) CreateSecret(title, secretType, data string) (string, error) {

	payload := map[string]string{
		"title": title,
		"type":  secretType,
		"data":  data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/secrets", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		fmt.Printf("DEBUG CLIENT: Sending token (length %d)\n", len(c.Token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if secretID, ok := result["secret_id"].(string); ok && secretID != "" {
		fmt.Printf("Secret created with ID: %s\n", secretID)
		return secretID, nil
	}

	return "", fmt.Errorf("no secret_id in server response")
}

func (c *Client) Register(email, password string) error {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/register", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	// Сервер может вернуть 201 Created или 200 OK
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Попробуем прочитать тело ошибки
		var errorResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			if msg, ok := errorResp["message"]; ok {
				return fmt.Errorf("registration failed: %s", msg)
			}
			if errorMsg, ok := errorResp["error"]; ok {
				return fmt.Errorf("registration failed: %s", errorMsg)
			}
		}
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetSecrets - получение всех секретов пользователя
func (c *Client) GetSecrets() ([]map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/secrets", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	// Декодируем один раз
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG: Server response: %+v\n", result)

	if secretsData, ok := result["secrets"]; ok {
		if secrets, ok := secretsData.([]interface{}); ok {
			var list []map[string]interface{}
			for _, s := range secrets {
				if m, ok := s.(map[string]interface{}); ok {
					list = append(list, m)
				}
			}
			return list, nil
		}
	}
	if data, ok := result["data"]; ok {
		if secrets, ok := data.([]interface{}); ok {
			var list []map[string]interface{}
			for _, s := range secrets {
				if m, ok := s.(map[string]interface{}); ok {
					list = append(list, m)
				}
			}
			return list, nil
		}
	}

	return []map[string]interface{}{}, nil
}

// GetSecret - получение одного секрета по ID
func (c *Client) GetSecret(id string) (map[string]interface{}, error) {

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/secrets/"+id, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG GETSECRET: Full response = %+v\n", result)

	return result, nil
}

func (c *Client) DeleteSecret(id string) error {

	if id == "" {
		return fmt.Errorf("secret id is required")
	}

	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/secrets/"+id, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
