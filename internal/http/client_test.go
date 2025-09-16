package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", client.timeout)
	}

	if client.retryConfig.MaxAttempts != 3 {
		t.Errorf("Expected default max attempts of 3, got %d", client.retryConfig.MaxAttempts)
	}
}

func TestClientBaseURL(t *testing.T) {
	client := NewClient().BaseURL("https://api.example.com")

	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}

	// Test trailing slash removal
	client = NewClient().BaseURL("https://api.example.com/")
	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}
}

func TestClientHeaders(t *testing.T) {
	client := NewClient().
		Header("Authorization", "Bearer token").
		Headers(map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		})

	if client.headers["Authorization"] != "Bearer token" {
		t.Errorf("Expected Authorization header 'Bearer token', got '%s'", client.headers["Authorization"])
	}

	if client.headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got '%s'", client.headers["Content-Type"])
	}
}

func TestClientQuery(t *testing.T) {
	client := NewClient().Query("api_key", "secret")

	if client.queryParams["api_key"] != "secret" {
		t.Errorf("Expected query param 'api_key' = 'secret', got '%s'", client.queryParams["api_key"])
	}
}

func TestGETRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

func TestPOSTRequestWithJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if body["name"] != "test" {
			t.Errorf("Expected name 'test', got '%s'", body["name"])
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.POST(server.URL).
		JSON(map[string]string{"name": "test"}).
		Send()

	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestRequestWithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("param1") != "value1" {
			t.Errorf("Expected param1 'value1', got '%s'", r.URL.Query().Get("param1"))
		}

		if r.URL.Query().Get("param2") != "value2" {
			t.Errorf("Expected param2 'value2', got '%s'", r.URL.Query().Get("param2"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.GET(server.URL).
		Query("param1", "value1").
		Queries(map[string]string{"param2": "value2"}).
		Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestRequestWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", r.Header.Get("X-Custom-Header"))
		}

		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Expected Authorization 'Bearer token', got '%s'", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.GET(server.URL).
		Header("X-Custom-Header", "custom-value").
		Headers(map[string]string{"Authorization": "Bearer token"}).
		Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestRequestWithForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type 'application/x-www-form-urlencoded', got '%s'", r.Header.Get("Content-Type"))
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.Form.Get("field1") != "value1" {
			t.Errorf("Expected field1 'value1', got '%s'", r.Form.Get("field1"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.POST(server.URL).
		Form(map[string]string{"field1": "value1"}).
		Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestRequestWithText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("Expected Content-Type 'text/plain', got '%s'", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		if string(body) != "Hello, World!" {
			t.Errorf("Expected body 'Hello, World!', got '%s'", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.POST(server.URL).
		Text("Hello, World!").
		Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.GET(server.URL).
		Timeout(100 * time.Millisecond).
		Send()

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestRequestContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := NewClient()
	_, err := client.GET(server.URL).
		Context(ctx).
		Send()

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

func TestRetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewClient().Retry(&RetryConfig{
		MaxAttempts: 3,
		Backoff:     NewFixedBackoff(10 * time.Millisecond),
		RetryIf:     DefaultRetryCondition,
	})

	resp, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.RetryCount() != 2 {
		t.Errorf("Expected 2 retries, got %d", resp.RetryCount())
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestResponseHelpers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if !resp.IsSuccess() {
		t.Error("Expected IsSuccess() to be true")
	}

	if resp.IsClientError() {
		t.Error("Expected IsClientError() to be false")
	}

	if resp.IsServerError() {
		t.Error("Expected IsServerError() to be false")
	}

	body := resp.String()
	if body != `{"message": "success"}` {
		t.Errorf("Expected body '{\"message\": \"success\"}', got '%s'", body)
	}

	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

func TestMiddleware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Middleware") != "applied" {
			t.Errorf("Expected X-Middleware header 'applied', got '%s'", r.Header.Get("X-Middleware"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	middleware := func(req *Request) error {
		req.Header("X-Middleware", "applied")
		return nil
	}

	client := NewClient().Use(middleware)
	_, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestInterceptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	interceptorCalled := false
	interceptor := func(resp *Response) error {
		interceptorCalled = true
		return nil
	}

	client := NewClient().Intercept(interceptor)
	_, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if !interceptorCalled {
		t.Error("Expected interceptor to be called")
	}
}

func TestCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()

	cache := NewMemoryCache()
	client := NewClient().Cache(cache)

	// First request - should hit the server
	resp1, err := client.GET(server.URL).Cache("test-key", time.Minute).Send()
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	if resp1.IsCached() {
		t.Error("Expected first response to not be cached")
	}

	// Second request - should hit the cache
	resp2, err := client.GET(server.URL).Cache("test-key", time.Minute).Send()
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}

	if !resp2.IsCached() {
		t.Error("Expected second response to be cached")
	}

	if resp1.String() != resp2.String() {
		t.Error("Expected cached response to match original")
	}
}

func TestAuthenticationBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Expected Basic auth header, got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient().Auth(Basic("user", "pass"))
	_, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestAuthenticationBearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer token123" {
			t.Errorf("Expected 'Bearer token123', got '%s'", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient().Auth(Bearer("token123"))
	_, err := client.GET(server.URL).Send()

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestAllHTTPMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					t.Errorf("Expected method %s, got %s", method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient()
			var resp *Response
			var err error

			switch method {
			case http.MethodGet:
				resp, err = client.GET(server.URL).Send()
			case http.MethodPost:
				resp, err = client.POST(server.URL).Send()
			case http.MethodPut:
				resp, err = client.PUT(server.URL).Send()
			case http.MethodPatch:
				resp, err = client.PATCH(server.URL).Send()
			case http.MethodDelete:
				resp, err = client.DELETE(server.URL).Send()
			case http.MethodHead:
				resp, err = client.HEAD(server.URL).Send()
			case http.MethodOptions:
				resp, err = client.OPTIONS(server.URL).Send()
			}

			if err != nil {
				t.Fatalf("%s request failed: %v", method, err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}
