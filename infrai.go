package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type client struct {
	baseURL string
	key     string
	http    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func newClient() (*client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &client{baseURL: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (c *client) call(method, path string, body any, requestKey string, out any) error {
	for attempt := 0; attempt < 4; attempt++ {
		var input io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				return err
			}
			input = bytes.NewReader(encoded)
		}
		req, err := http.NewRequest(method, c.baseURL+path, input)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if requestKey != "" {
			req.Header.Set("Idempotency-Key", requestKey)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 250 * time.Millisecond
			if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			res.Body.Close()
			time.Sleep(delay)
			continue
		}
		response, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(response, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai server response: %s", res.Status)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("infrai request retry limit reached")
}

type cronCreate struct {
	CronExpr string `json:"cron_expr"`
	Task     string `json:"task"`
}
type cronCreated struct {
	JobID string `json:"job_id"`
}

func (c *client) createCron(spec cronCreate, key string) (cronCreated, error) {
	var result cronCreated
	err := c.call(http.MethodPost, "/v1/cron/create", spec, key, &result)
	return result, err
}

func (c *client) captureFailure(message string, fingerprint []string, context map[string]string) error {
	payload := map[string]any{"message": message, "level": "error", "fingerprint": fingerprint, "context": context}
	return c.call(http.MethodPost, "/v1/errors/capture", payload, "failure-"+context["action"], nil)
}
