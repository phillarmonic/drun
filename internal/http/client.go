package http

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client represents a semantic HTTP client with powerful features
type Client struct {
	httpClient   *http.Client
	baseURL      string
	headers      map[string]string
	queryParams  map[string]string
	timeout      time.Duration
	retryConfig  *RetryConfig
	middleware   []Middleware
	interceptors []Interceptor
	cache        Cache
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts int
	Backoff     BackoffStrategy
	RetryIf     func(*http.Response, error) bool
}

// BackoffStrategy defines how to calculate retry delays
type BackoffStrategy interface {
	NextDelay(attempt int) time.Duration
}

// Middleware processes requests before they are sent
type Middleware func(*Request) error

// Interceptor processes responses after they are received
type Interceptor func(*Response) error

// Cache interface for HTTP response caching
type Cache interface {
	Get(key string) (*Response, bool)
	Set(key string, response *Response, ttl time.Duration)
	Delete(key string)
}

// Request represents an HTTP request with fluent API
type Request struct {
	client      *Client
	method      string
	url         string
	headers     map[string]string
	queryParams map[string]string
	body        io.Reader
	bodyData    interface{}
	contentType string
	timeout     time.Duration
	retries     *RetryConfig
	cacheKey    string
	cacheTTL    time.Duration
	ctx         context.Context
}

// Response represents an HTTP response with helper methods
type Response struct {
	*http.Response
	body       []byte
	cached     bool
	retryCount int
	duration   time.Duration
}

// NewClient creates a new HTTP client with default configuration
func NewClient() *Client {
	return NewClientWithVersion("dev")
}

// NewClientWithVersion creates a new HTTP client with default configuration and version
func NewClientWithVersion(version string) *Client {
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["User-Agent"] = fmt.Sprintf("drun/%s", version)

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers:     headers,
		queryParams: make(map[string]string),
		timeout:     30 * time.Second,
		retryConfig: &RetryConfig{
			MaxAttempts: 3,
			Backoff:     &ExponentialBackoff{BaseDelay: time.Second},
			RetryIf:     DefaultRetryCondition,
		},
	}
}

// BaseURL sets the base URL for all requests
func (c *Client) BaseURL(url string) *Client {
	c.baseURL = strings.TrimSuffix(url, "/")
	return c
}

// Timeout sets the default timeout for requests
func (c *Client) Timeout(timeout time.Duration) *Client {
	c.timeout = timeout
	c.httpClient.Timeout = timeout
	return c
}

// Header sets a default header for all requests
func (c *Client) Header(key, value string) *Client {
	c.headers[key] = value
	return c
}

// Headers sets multiple default headers
func (c *Client) Headers(headers map[string]string) *Client {
	for k, v := range headers {
		c.headers[k] = v
	}
	return c
}

// Query sets a default query parameter for all requests
func (c *Client) Query(key, value string) *Client {
	c.queryParams[key] = value
	return c
}

// Auth sets authentication headers
func (c *Client) Auth(auth Auth) *Client {
	return auth.Apply(c)
}

// Retry configures retry behavior
func (c *Client) Retry(config *RetryConfig) *Client {
	c.retryConfig = config
	return c
}

// Use adds middleware to the client
func (c *Client) Use(middleware ...Middleware) *Client {
	c.middleware = append(c.middleware, middleware...)
	return c
}

// Intercept adds response interceptors
func (c *Client) Intercept(interceptors ...Interceptor) *Client {
	c.interceptors = append(c.interceptors, interceptors...)
	return c
}

// Cache sets the cache implementation
func (c *Client) Cache(cache Cache) *Client {
	c.cache = cache
	return c
}

// GET creates a GET request
func (c *Client) GET(url string) *Request {
	return c.newRequest(http.MethodGet, url)
}

// POST creates a POST request
func (c *Client) POST(url string) *Request {
	return c.newRequest(http.MethodPost, url)
}

// PUT creates a PUT request
func (c *Client) PUT(url string) *Request {
	return c.newRequest(http.MethodPut, url)
}

// PATCH creates a PATCH request
func (c *Client) PATCH(url string) *Request {
	return c.newRequest(http.MethodPatch, url)
}

// DELETE creates a DELETE request
func (c *Client) DELETE(url string) *Request {
	return c.newRequest(http.MethodDelete, url)
}

// HEAD creates a HEAD request
func (c *Client) HEAD(url string) *Request {
	return c.newRequest(http.MethodHead, url)
}

// OPTIONS creates an OPTIONS request
func (c *Client) OPTIONS(url string) *Request {
	return c.newRequest(http.MethodOptions, url)
}

// newRequest creates a new request with client defaults
func (c *Client) newRequest(method, url string) *Request {
	// Build full URL
	fullURL := url
	if c.baseURL != "" && !strings.HasPrefix(url, "http") {
		fullURL = c.baseURL + "/" + strings.TrimPrefix(url, "/")
	}

	req := &Request{
		client:      c,
		method:      method,
		url:         fullURL,
		headers:     make(map[string]string),
		queryParams: make(map[string]string),
		timeout:     c.timeout,
		retries:     c.retryConfig,
		ctx:         context.Background(),
	}

	// Copy client defaults
	for k, v := range c.headers {
		req.headers[k] = v
	}
	for k, v := range c.queryParams {
		req.queryParams[k] = v
	}

	return req
}

// Header sets a header for this request
func (r *Request) Header(key, value string) *Request {
	r.headers[key] = value
	return r
}

// Headers sets multiple headers for this request
func (r *Request) Headers(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers[k] = v
	}
	return r
}

// Query sets a query parameter for this request
func (r *Request) Query(key, value string) *Request {
	r.queryParams[key] = value
	return r
}

// Queries sets multiple query parameters
func (r *Request) Queries(params map[string]string) *Request {
	for k, v := range params {
		r.queryParams[k] = v
	}
	return r
}

// Body sets the request body from a reader
func (r *Request) Body(body io.Reader) *Request {
	r.body = body
	return r
}

// JSON sets the request body as JSON
func (r *Request) JSON(data interface{}) *Request {
	r.bodyData = data
	r.contentType = "application/json"
	return r
}

// XML sets the request body as XML
func (r *Request) XML(data interface{}) *Request {
	r.bodyData = data
	r.contentType = "application/xml"
	return r
}

// Form sets the request body as form data
func (r *Request) Form(data map[string]string) *Request {
	values := url.Values{}
	for k, v := range data {
		values.Set(k, v)
	}
	r.body = strings.NewReader(values.Encode())
	r.contentType = "application/x-www-form-urlencoded"
	return r
}

// Text sets the request body as plain text
func (r *Request) Text(text string) *Request {
	r.body = strings.NewReader(text)
	r.contentType = "text/plain"
	return r
}

// Timeout sets the timeout for this request
func (r *Request) Timeout(timeout time.Duration) *Request {
	r.timeout = timeout
	return r
}

// Context sets the context for this request
func (r *Request) Context(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Retry configures retry behavior for this request
func (r *Request) Retry(config *RetryConfig) *Request {
	r.retries = config
	return r
}

// Cache configures caching for this request
func (r *Request) Cache(key string, ttl time.Duration) *Request {
	r.cacheKey = key
	r.cacheTTL = ttl
	return r
}

// Send executes the request and returns the response
func (r *Request) Send() (*Response, error) {
	// Check cache first
	if r.cacheKey != "" && r.client.cache != nil {
		if cached, ok := r.client.cache.Get(r.cacheKey); ok {
			cached.cached = true
			return cached, nil
		}
	}

	// Apply middleware first
	for _, middleware := range r.client.middleware {
		if err := middleware(r); err != nil {
			return nil, fmt.Errorf("middleware error: %w", err)
		}
	}

	// Prepare the request body
	if err := r.prepareBody(); err != nil {
		return nil, fmt.Errorf("failed to prepare request body: %w", err)
	}

	// Build the HTTP request
	httpReq, err := r.buildHTTPRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}

	// Execute with retries
	resp, err := r.executeWithRetries(httpReq)
	if err != nil {
		return nil, err
	}

	// Apply interceptors
	for _, interceptor := range r.client.interceptors {
		if err := interceptor(resp); err != nil {
			return nil, fmt.Errorf("interceptor error: %w", err)
		}
	}

	// Cache the response
	if r.cacheKey != "" && r.client.cache != nil && resp.StatusCode < 400 {
		r.client.cache.Set(r.cacheKey, resp, r.cacheTTL)
	}

	return resp, nil
}

// prepareBody prepares the request body based on bodyData and contentType
func (r *Request) prepareBody() error {
	if r.bodyData == nil {
		return nil
	}

	var data []byte
	var err error

	switch r.contentType {
	case "application/json":
		data, err = json.Marshal(r.bodyData)
	case "application/xml":
		data, err = xml.Marshal(r.bodyData)
	default:
		return fmt.Errorf("unsupported content type: %s", r.contentType)
	}

	if err != nil {
		return err
	}

	r.body = bytes.NewReader(data)
	return nil
}

// buildHTTPRequest builds the standard HTTP request
func (r *Request) buildHTTPRequest() (*http.Request, error) {
	// Build URL with query parameters
	u, err := url.Parse(r.url)
	if err != nil {
		return nil, err
	}

	if len(r.queryParams) > 0 {
		q := u.Query()
		for k, v := range r.queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Create the request
	req, err := http.NewRequestWithContext(r.ctx, r.method, u.String(), r.body)
	if err != nil {
		return nil, err
	}

	// Set headers
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// Set content type if specified
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}

	return req, nil
}

// executeWithRetries executes the request with retry logic
func (r *Request) executeWithRetries(req *http.Request) (*Response, error) {
	var lastErr error
	var resp *Response

	maxAttempts := 1
	if r.retries != nil {
		maxAttempts = r.retries.MaxAttempts
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		start := time.Now()

		// Create a new context with timeout for this attempt
		ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
		reqWithTimeout := req.WithContext(ctx)

		httpResp, err := r.client.httpClient.Do(reqWithTimeout)
		cancel()

		duration := time.Since(start)

		if err != nil {
			lastErr = err
			if attempt < maxAttempts-1 && r.shouldRetry(nil, err) {
				time.Sleep(r.retries.Backoff.NextDelay(attempt))
				continue
			}
			return nil, err
		}

		// Read response body
		body, err := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close() // Ignore close error as we've already read the body
		if err != nil {
			lastErr = err
			if attempt < maxAttempts-1 && r.shouldRetry(httpResp, err) {
				time.Sleep(r.retries.Backoff.NextDelay(attempt))
				continue
			}
			return nil, err
		}

		resp = &Response{
			Response:   httpResp,
			body:       body,
			retryCount: attempt,
			duration:   duration,
		}

		// Check if we should retry based on response
		if attempt < maxAttempts-1 && r.shouldRetry(httpResp, nil) {
			time.Sleep(r.retries.Backoff.NextDelay(attempt))
			continue
		}

		return resp, nil
	}

	return resp, lastErr
}

// shouldRetry determines if a request should be retried
func (r *Request) shouldRetry(resp *http.Response, err error) bool {
	if r.retries == nil || r.retries.RetryIf == nil {
		return false
	}
	return r.retries.RetryIf(resp, err)
}

// Body returns the response body as bytes
func (r *Response) Body() []byte {
	return r.body
}

// String returns the response body as string
func (r *Response) String() string {
	return string(r.body)
}

// JSON unmarshals the response body as JSON
func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.body, v)
}

// XML unmarshals the response body as XML
func (r *Response) XML(v interface{}) error {
	return xml.Unmarshal(r.body, v)
}

// IsSuccess returns true if the response status code indicates success (2xx)
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsClientError returns true if the response status code indicates client error (4xx)
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response status code indicates server error (5xx)
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

// IsCached returns true if the response was served from cache
func (r *Response) IsCached() bool {
	return r.cached
}

// RetryCount returns the number of retries that were performed
func (r *Response) RetryCount() int {
	return r.retryCount
}

// Duration returns the total duration of the request (including retries)
func (r *Response) Duration() time.Duration {
	return r.duration
}

// DefaultRetryCondition is the default retry condition
func DefaultRetryCondition(resp *http.Response, err error) bool {
	if err != nil {
		return true // Retry on network errors
	}
	if resp == nil {
		return false
	}
	// Retry on server errors and rate limiting
	return resp.StatusCode >= 500 || resp.StatusCode == 429
}
