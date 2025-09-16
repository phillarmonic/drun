package http

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/model"
)

// TemplateHTTPClient provides HTTP functionality for templates
type TemplateHTTPClient struct {
	client    *Client
	endpoints map[string]model.HTTPEndpoint
	secrets   map[string]string
}

// NewTemplateHTTPClient creates a new template HTTP client
func NewTemplateHTTPClient(endpoints map[string]model.HTTPEndpoint, secrets map[string]string) *TemplateHTTPClient {
	return &TemplateHTTPClient{
		client:    NewClient().Cache(NewMemoryCache()),
		endpoints: endpoints,
		secrets:   secrets,
	}
}

// HTTPCall makes an HTTP call using a predefined endpoint
func (t *TemplateHTTPClient) HTTPCall(endpointName string, options ...map[string]interface{}) (string, error) {
	endpoint, exists := t.endpoints[endpointName]
	if !exists {
		return "", fmt.Errorf("HTTP endpoint '%s' not found", endpointName)
	}

	// Merge options
	opts := make(map[string]interface{})
	for _, opt := range options {
		for k, v := range opt {
			opts[k] = v
		}
	}

	// Build the request
	req, err := t.buildRequest(endpoint, opts)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	// Execute the request
	resp, err := req.Send()
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.String(), nil
}

// HTTPCallJSON makes an HTTP call and returns parsed JSON
func (t *TemplateHTTPClient) HTTPCallJSON(endpointName string, options ...map[string]interface{}) (map[string]interface{}, error) {
	responseStr, err := t.HTTPCall(endpointName, options...)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(responseStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}

// HTTPGet makes a GET request to a URL
func (t *TemplateHTTPClient) HTTPGet(url string, options ...map[string]interface{}) (string, error) {
	req := t.client.GET(url)

	// Apply options
	for _, opts := range options {
		if err := t.applyOptions(req, opts); err != nil {
			return "", err
		}
	}

	resp, err := req.Send()
	if err != nil {
		return "", fmt.Errorf("HTTP GET failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("HTTP GET failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.String(), nil
}

// HTTPPost makes a POST request to a URL
func (t *TemplateHTTPClient) HTTPPost(url string, body interface{}, options ...map[string]interface{}) (string, error) {
	req := t.client.POST(url)

	// Set body based on type
	switch v := body.(type) {
	case string:
		req.Text(v)
	case map[string]interface{}:
		req.JSON(v)
	default:
		req.JSON(v)
	}

	// Apply options
	for _, opts := range options {
		if err := t.applyOptions(req, opts); err != nil {
			return "", err
		}
	}

	resp, err := req.Send()
	if err != nil {
		return "", fmt.Errorf("HTTP POST failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("HTTP POST failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.String(), nil
}

// HTTPPut makes a PUT request to a URL
func (t *TemplateHTTPClient) HTTPPut(url string, body interface{}, options ...map[string]interface{}) (string, error) {
	req := t.client.PUT(url)

	// Set body based on type
	switch v := body.(type) {
	case string:
		req.Text(v)
	case map[string]interface{}:
		req.JSON(v)
	default:
		req.JSON(v)
	}

	// Apply options
	for _, opts := range options {
		if err := t.applyOptions(req, opts); err != nil {
			return "", err
		}
	}

	resp, err := req.Send()
	if err != nil {
		return "", fmt.Errorf("HTTP PUT failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("HTTP PUT failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.String(), nil
}

// HTTPDelete makes a DELETE request to a URL
func (t *TemplateHTTPClient) HTTPDelete(url string, options ...map[string]interface{}) (string, error) {
	req := t.client.DELETE(url)

	// Apply options
	for _, opts := range options {
		if err := t.applyOptions(req, opts); err != nil {
			return "", err
		}
	}

	resp, err := req.Send()
	if err != nil {
		return "", fmt.Errorf("HTTP DELETE failed: %w", err)
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("HTTP DELETE failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.String(), nil
}

// buildRequest builds a request from an endpoint configuration
func (t *TemplateHTTPClient) buildRequest(endpoint model.HTTPEndpoint, opts map[string]interface{}) (*Request, error) {
	// Determine method
	method := strings.ToUpper(endpoint.Method)
	if method == "" {
		method = "GET"
	}

	// Create request
	var req *Request
	switch method {
	case "GET":
		req = t.client.GET(endpoint.URL)
	case "POST":
		req = t.client.POST(endpoint.URL)
	case "PUT":
		req = t.client.PUT(endpoint.URL)
	case "PATCH":
		req = t.client.PATCH(endpoint.URL)
	case "DELETE":
		req = t.client.DELETE(endpoint.URL)
	case "HEAD":
		req = t.client.HEAD(endpoint.URL)
	case "OPTIONS":
		req = t.client.OPTIONS(endpoint.URL)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	// Apply endpoint configuration
	if err := t.applyEndpointConfig(req, endpoint); err != nil {
		return nil, err
	}

	// Apply runtime options
	if err := t.applyOptions(req, opts); err != nil {
		return nil, err
	}

	return req, nil
}

// applyEndpointConfig applies endpoint configuration to a request
func (t *TemplateHTTPClient) applyEndpointConfig(req *Request, endpoint model.HTTPEndpoint) error {
	// Set headers
	for k, v := range endpoint.Headers {
		req.Header(k, t.resolveSecretValue(v))
	}

	// Set timeout
	if endpoint.Timeout > 0 {
		req.Timeout(endpoint.Timeout)
	}

	// Set authentication
	if err := t.applyAuth(req, endpoint.Auth); err != nil {
		return err
	}

	// Set retry configuration
	if endpoint.Retry.MaxAttempts > 0 {
		retryConfig := &RetryConfig{
			MaxAttempts: endpoint.Retry.MaxAttempts,
			RetryIf:     DefaultRetryCondition,
		}

		// Set backoff strategy
		switch endpoint.Retry.Backoff {
		case "exponential":
			retryConfig.Backoff = &ExponentialBackoff{
				BaseDelay: endpoint.Retry.BaseDelay,
				MaxDelay:  endpoint.Retry.MaxDelay,
			}
		case "linear":
			retryConfig.Backoff = &LinearBackoff{
				BaseDelay: endpoint.Retry.BaseDelay,
				MaxDelay:  endpoint.Retry.MaxDelay,
			}
		case "fixed":
			retryConfig.Backoff = &FixedBackoff{
				Delay: endpoint.Retry.BaseDelay,
			}
		default:
			retryConfig.Backoff = NewExponentialBackoff(endpoint.Retry.BaseDelay)
		}

		req.Retry(retryConfig)
	}

	// Set cache configuration
	if endpoint.Cache.TTL > 0 {
		cacheKey := endpoint.Cache.Key
		if cacheKey == "" {
			cacheKey = fmt.Sprintf("endpoint:%s", endpoint.URL)
		}
		req.Cache(cacheKey, endpoint.Cache.TTL)
	}

	return nil
}

// applyAuth applies authentication configuration
func (t *TemplateHTTPClient) applyAuth(req *Request, auth model.HTTPAuth) error {
	switch strings.ToLower(auth.Type) {
	case "basic":
		user := t.resolveSecretValue(auth.User)
		pass := t.resolveSecretValue(auth.Pass)
		req.Header("Authorization", "Basic "+encodeBasicAuth(user, pass))
	case "bearer":
		token := t.resolveSecretValue(auth.Token)
		req.Header("Authorization", "Bearer "+token)
	case "api-key":
		value := t.resolveSecretValue(auth.Token)
		if auth.Header != "" {
			req.Header(auth.Header, value)
		} else if auth.Query != "" {
			req.Query(auth.Query, value)
		} else {
			return fmt.Errorf("api-key auth requires either header or query parameter name")
		}
	case "oauth2":
		token := t.resolveSecretValue(auth.Token)
		req.Header("Authorization", "Bearer "+token)
	case "":
		// No authentication
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}

	return nil
}

// applyOptions applies runtime options to a request
func (t *TemplateHTTPClient) applyOptions(req *Request, opts map[string]interface{}) error {
	for key, value := range opts {
		switch strings.ToLower(key) {
		case "headers":
			if headers, ok := value.(map[string]interface{}); ok {
				for k, v := range headers {
					if str, ok := v.(string); ok {
						req.Header(k, str)
					}
				}
			}
		case "query", "params":
			if params, ok := value.(map[string]interface{}); ok {
				for k, v := range params {
					if str, ok := v.(string); ok {
						req.Query(k, str)
					}
				}
			}
		case "timeout":
			if duration, ok := value.(time.Duration); ok {
				req.Timeout(duration)
			} else if str, ok := value.(string); ok {
				if d, err := time.ParseDuration(str); err == nil {
					req.Timeout(d)
				}
			}
		case "body":
			switch v := value.(type) {
			case string:
				req.Text(v)
			case map[string]interface{}:
				req.JSON(v)
			default:
				req.JSON(v)
			}
		}
	}

	return nil
}

// resolveSecretValue resolves secret references in values
func (t *TemplateHTTPClient) resolveSecretValue(value string) string {
	// Check if it's a secret reference: {{ secret "name" }}
	if strings.HasPrefix(value, "{{ secret ") && strings.HasSuffix(value, " }}") {
		// Extract secret name (simplified parsing)
		secretName := strings.TrimSpace(value[10 : len(value)-3])
		secretName = strings.Trim(secretName, `"'`)

		if secretValue, exists := t.secrets[secretName]; exists {
			return secretValue
		}
	}

	return value
}

// encodeBasicAuth encodes username and password for basic auth
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return encodeBase64([]byte(auth))
}

// encodeBase64 encodes bytes to base64 string
func encodeBase64(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	result := make([]byte, ((len(data)+2)/3)*4)

	for i, j := 0, 0; i < len(data); i, j = i+3, j+4 {
		b := uint32(data[i]) << 16
		if i+1 < len(data) {
			b |= uint32(data[i+1]) << 8
		}
		if i+2 < len(data) {
			b |= uint32(data[i+2])
		}

		result[j] = base64Table[(b>>18)&63]
		result[j+1] = base64Table[(b>>12)&63]
		if i+1 < len(data) {
			result[j+2] = base64Table[(b>>6)&63]
		} else {
			result[j+2] = '='
		}
		if i+2 < len(data) {
			result[j+3] = base64Table[b&63]
		} else {
			result[j+3] = '='
		}
	}

	return string(result)
}

// GetTemplateFunctions returns HTTP template functions
func (t *TemplateHTTPClient) GetTemplateFunctions() map[string]interface{} {
	return map[string]interface{}{
		"httpCall":     t.HTTPCall,
		"httpCallJSON": t.HTTPCallJSON,
		"httpGet":      t.HTTPGet,
		"httpPost":     t.HTTPPost,
		"httpPut":      t.HTTPPut,
		"httpDelete":   t.HTTPDelete,
	}
}
