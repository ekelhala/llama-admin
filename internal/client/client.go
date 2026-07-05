package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL      string
	SessionToken string
	HTTP         *http.Client
}

func New(serverURL, sessionToken string) *Client {
	return &Client{
		BaseURL:      strings.TrimRight(serverURL, "/"),
		SessionToken: sessionToken,
		HTTP:         &http.Client{},
	}
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	url := c.BaseURL + path
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.SessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.SessionToken)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.ErrorResponse.Message != "" {
			return nil, apiErr
		}
		return nil, fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) Get(path string) ([]byte, error) {
	return c.do("GET", path, nil)
}

func (c *Client) Post(path string, body any) ([]byte, error) {
	return c.do("POST", path, body)
}

func (c *Client) Delete(path string) ([]byte, error) {
	return c.do("DELETE", path, nil)
}

type APIError struct {
	ErrorResponse struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error (%s): %s", e.ErrorResponse.Type, e.ErrorResponse.Message)
}
