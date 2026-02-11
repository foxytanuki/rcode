// Package main implements the rcode client CLI application.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/foxytanuki/rcode/internal/config"
	"github.com/foxytanuki/rcode/internal/logger"
	"github.com/foxytanuki/rcode/internal/version"
	"github.com/foxytanuki/rcode/pkg/api"
)

// Client represents the rcode CLI client
type Client struct {
	config     *config.ClientConfig
	log        *logger.Logger
	httpClient *http.Client
}

// NewClient creates a new client instance
func NewClient(cfg *config.ClientConfig, log *logger.Logger) *Client {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: cfg.Network.Timeout * 2, // Double the timeout for the full request
	}

	return &Client{
		config:     cfg,
		log:        log,
		httpClient: httpClient,
	}
}

// ensurePort appends the default port if the host doesn't include one.
func ensurePort(host string) string {
	if !strings.Contains(host, ":") {
		return host + ":3339"
	}
	return host
}

// withFallback tries fn against the primary host, then the fallback host.
func (c *Client) withFallback(fn func(host string) error) error {
	err := fn(c.config.Hosts.Server.Primary)
	if err == nil {
		return nil
	}
	c.log.Warn("Primary host failed", "host", c.config.Hosts.Server.Primary, "error", err)

	if c.config.Hosts.Server.Fallback != "" {
		if err2 := fn(c.config.Hosts.Server.Fallback); err2 == nil {
			return nil
		}
		c.log.Warn("Fallback host failed", "host", c.config.Hosts.Server.Fallback, "error", err)
	}
	return fmt.Errorf("failed to connect to any configured host: %w", err)
}

// OpenEditor opens a file/directory in an editor on the host machine
func (c *Client) OpenEditor(path, editor string, sshInfo *SSHInfo) error {
	// Use default editor if not specified
	if editor == "" {
		editor = c.config.DefaultEditor
	}

	// Create the request
	req := api.OpenRequest{
		Path:   path,
		Editor: editor,
		User:   sshInfo.User,
		Host:   sshInfo.Host,
	}
	req.SetTimestamp()

	return c.withFallback(func(host string) error {
		return c.sendRequest(host, req)
	})
}

// sendRequest sends the open editor request to a specific host
func (c *Client) sendRequest(host string, req api.OpenRequest) error {
	host = ensurePort(host)
	url := fmt.Sprintf("http://%s/open-editor", host)

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Network.Timeout)
	defer cancel()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", fmt.Sprintf("rcode/%s", version.Version))

	// Perform retries if configured
	var lastErr error
	attempts := c.config.Network.RetryAttempts
	if attempts <= 0 {
		attempts = 1
	}

	for i := 0; i < attempts; i++ {
		if i > 0 {
			c.log.Debug("Retrying request",
				"attempt", i+1,
				"max_attempts", attempts,
			)
			time.Sleep(c.config.Network.RetryDelay)
		}

		// Send request
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		// Process response - close body when done
		func() {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					c.log.Warn("Failed to close response body", "error", err)
				}
			}()

			// Check status code
			if resp.StatusCode == http.StatusOK {
				// Parse successful response
				var openResp api.OpenResponse
				if err := json.NewDecoder(resp.Body).Decode(&openResp); err != nil {
					lastErr = fmt.Errorf("failed to decode response: %w", err)
					return
				}

				c.log.Info("Editor opened successfully",
					"editor", openResp.Editor,
					"command", openResp.Command,
				)

				lastErr = nil
				return
			}

			// Parse error response
			var errResp api.ErrorResponse
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				lastErr = fmt.Errorf("server returned status %d", resp.StatusCode)
			} else {
				lastErr = fmt.Errorf("server error: %s", errResp.Error())
			}
		}()

		// If successful, return immediately
		if lastErr == nil {
			return nil
		}
	}

	return lastErr
}

// ListEditors lists available editors from the server
func (c *Client) ListEditors() error {
	var editors *api.EditorsResponse

	err := c.withFallback(func(host string) error {
		var fetchErr error
		editors, fetchErr = c.fetchEditors(host)
		return fetchErr
	})
	if err != nil {
		return err
	}

	// Display editors
	fmt.Println("Available Editors:")
	fmt.Println("==================")
	for _, editor := range editors.Editors {
		status := ""
		if editor.Default {
			status = " (default)"
		}
		if !editor.Available {
			status += " [unavailable]"
		}
		fmt.Printf("  %s%s\n", editor.Name, status)
		fmt.Printf("    Command: %s\n", editor.Command)
	}

	return nil
}

// fetchEditors fetches the list of editors from a specific host
func (c *Client) fetchEditors(host string) (*api.EditorsResponse, error) {
	host = ensurePort(host)
	url := fmt.Sprintf("http://%s/editors", host)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Network.Timeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("rcode/%s", version.Version))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Warn("Failed to close response body", "error", err)
		}
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Parse response
	var editorsResp api.EditorsResponse
	if err := json.NewDecoder(resp.Body).Decode(&editorsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &editorsResp, nil
}

// GetManualCommand generates a manual command that can be run on the host.
// It first tries to fetch the command template from the server.
// If the server is unreachable, it falls back to configured fallback editors.
func (c *Client) GetManualCommand(path, editor string, sshInfo *SSHInfo) string {
	// Use default editor if not specified
	if editor == "" {
		editor = c.config.DefaultEditor
	}

	// Try to fetch editor command from server
	editorCmd := c.fetchEditorCommand(editor)

	if editorCmd == "" {
		// Fall back to configured fallback editors
		if c.config.FallbackEditors != nil {
			editorCmd = c.config.FallbackEditors[editor]
		}
	}

	if editorCmd == "" {
		return ""
	}

	// Replace placeholders
	cmd := strings.ReplaceAll(editorCmd, "{user}", sshInfo.User)
	cmd = strings.ReplaceAll(cmd, "{host}", sshInfo.Host)
	cmd = strings.ReplaceAll(cmd, "{path}", path)

	return cmd
}

// fetchEditorCommand fetches the command template for a specific editor from the server
func (c *Client) fetchEditorCommand(editorName string) string {
	var editors *api.EditorsResponse

	err := c.withFallback(func(host string) error {
		var fetchErr error
		editors, fetchErr = c.fetchEditors(host)
		return fetchErr
	})
	if err != nil {
		c.log.Debug("Failed to fetch editors from server", "error", err)
		return ""
	}

	// Find the editor command
	for _, editor := range editors.Editors {
		if editor.Name == editorName {
			return editor.Command
		}
	}

	return ""
}

// CheckHealth checks the health of the server
func (c *Client) CheckHealth() error {
	// Try primary host
	healthy, err := c.checkHostHealth(c.config.Hosts.Server.Primary)
	if err == nil && healthy {
		fmt.Printf("Primary host (%s) is healthy\n", c.config.Hosts.Server.Primary)
		return nil
	}

	if err != nil {
		fmt.Printf("Primary host (%s) check failed: %v\n", c.config.Hosts.Server.Primary, err)
	}

	// Try fallback host if configured
	if c.config.Hosts.Server.Fallback != "" {
		healthy, err = c.checkHostHealth(c.config.Hosts.Server.Fallback)
		if err == nil && healthy {
			fmt.Printf("Fallback host (%s) is healthy\n", c.config.Hosts.Server.Fallback)
			return nil
		}

		if err != nil {
			fmt.Printf("Fallback host (%s) check failed: %v\n", c.config.Hosts.Server.Fallback, err)
		}
	}

	return fmt.Errorf("no healthy hosts found")
}

// checkHostHealth checks the health of a specific host
func (c *Client) checkHostHealth(host string) (bool, error) {
	host = ensurePort(host)
	url := fmt.Sprintf("http://%s/health", host)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Network.Timeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("rcode/%s", version.Version))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Warn("Failed to close response body", "error", err)
		}
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Parse response
	var healthResp api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return healthResp.IsHealthy(), nil
}
