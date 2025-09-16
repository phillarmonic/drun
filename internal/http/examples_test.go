package http

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
)

// Example_basicUsage demonstrates basic HTTP client usage
func Example_basicUsage() {
	client := NewClient()

	// Simple GET request
	resp, err := client.GET("https://api.github.com/users/octocat").Send()
	if err != nil {
		log.Fatal(err)
	}

	// Parse JSON response
	var user struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if err := resp.JSON(&user); err != nil {
		log.Fatal(err)
	}

	log.Printf("User: %s (%s)", user.Name, user.Login)
}

// Example_postWithJSON demonstrates POST request with JSON body
func Example_postWithJSON() {
	client := NewClient().BaseURL("https://api.example.com")

	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	resp, err := client.POST("/users").
		JSON(data).
		Send()

	if err != nil {
		log.Fatal(err)
	}

	if resp.IsSuccess() {
		log.Println("User created successfully")
	}
}

// Example_authentication demonstrates various authentication methods
func Example_authentication() {
	// Basic Authentication
	client1 := NewClient().Auth(Basic("username", "password"))
	_, _ = client1.GET("https://api.example.com/protected").Send()

	// Bearer Token
	client2 := NewClient().Auth(Bearer("your-token-here"))
	_, _ = client2.GET("https://api.example.com/protected").Send()

	// API Key in header
	client3 := NewClient().Auth(APIKey("X-API-Key", "your-api-key"))
	_, _ = client3.GET("https://api.example.com/protected").Send()

	// API Key in query parameter
	client4 := NewClient().Auth(APIKeyQuery("api_key", "your-api-key"))
	_, _ = client4.GET("https://api.example.com/protected").Send()

	// OAuth2
	client5 := NewClient().Auth(OAuth2("access-token"))
	_, _ = client5.GET("https://api.example.com/protected").Send()

	// Custom authentication
	client6 := NewClient().Auth(Custom(func(c *Client) *Client {
		return c.Header("Authorization", "Custom token")
	}))
	_, _ = client6.GET("https://api.example.com/protected").Send()
}

// Example_retryAndBackoff demonstrates retry configuration
func Example_retryAndBackoff() {
	client := NewClient().Retry(&RetryConfig{
		MaxAttempts: 5,
		Backoff:     NewExponentialBackoff(time.Second),
		RetryIf: func(resp *http.Response, err error) bool {
			// Retry on network errors or 5xx status codes
			return err != nil || (resp != nil && resp.StatusCode >= 500)
		},
	})

	resp, err := client.GET("https://unreliable-api.example.com").Send()
	if err != nil {
		log.Printf("Request failed after %d retries", resp.RetryCount())
	}
}

// Example_middleware demonstrates middleware usage
func Example_middleware() {
	logger := log.New(os.Stdout, "HTTP: ", log.LstdFlags)

	client := NewClient().
		Use(LoggingMiddleware(logger)).
		Use(UserAgentMiddleware("MyApp/1.0")).
		Use(CompressionMiddleware()).
		Intercept(LoggingInterceptor(logger)).
		Intercept(ClientErrorInterceptor())

	_, _ = client.GET("https://api.example.com/data").Send()
}

// Example_caching demonstrates response caching
func Example_caching() {
	cache := NewMemoryCache()
	client := NewClient().Cache(cache)

	// First request - hits the server
	resp1, _ := client.GET("https://api.example.com/data").
		Cache("api-data", 5*time.Minute).
		Send()

	// Second request - served from cache
	resp2, _ := client.GET("https://api.example.com/data").
		Cache("api-data", 5*time.Minute).
		Send()

	log.Printf("First request cached: %v", resp1.IsCached())
	log.Printf("Second request cached: %v", resp2.IsCached())
}

// Example_contextAndTimeout demonstrates context usage and timeouts
func Example_contextAndTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := NewClient()

	resp, err := client.GET("https://slow-api.example.com").
		Context(ctx).
		Timeout(5 * time.Second).
		Send()

	if err != nil {
		log.Printf("Request failed: %v", err)
	} else {
		log.Printf("Request completed in %v", resp.Duration())
	}
}

// Example_formData demonstrates form data submission
func Example_formData() {
	client := NewClient()

	formData := map[string]string{
		"username": "john",
		"password": "secret",
		"remember": "true",
	}

	resp, err := client.POST("https://example.com/login").
		Form(formData).
		Send()

	if err != nil {
		log.Fatal(err)
	}

	if resp.IsSuccess() {
		log.Println("Login successful")
	}
}

// Example_fileUpload demonstrates file upload (multipart form)
func Example_fileUpload() {
	client := NewClient()

	// For file uploads, you would typically use multipart/form-data
	// This is a simplified example using text content
	resp, err := client.POST("https://api.example.com/upload").
		Header("Content-Type", "multipart/form-data").
		Text("file content here").
		Send()

	if err != nil {
		log.Fatal(err)
	}

	if resp.IsSuccess() {
		log.Println("File uploaded successfully")
	}
}

// Example_rateLimiting demonstrates rate limiting
func Example_rateLimiting() {
	rateLimiter := NewRateLimiter(10, time.Minute) // 10 requests per minute

	client := NewClient().Use(rateLimiter.Middleware())

	for i := 0; i < 15; i++ {
		_, err := client.GET("https://api.example.com/data").Send()
		if err != nil {
			log.Printf("Request %d failed: %v", i+1, err)
			break
		}
		log.Printf("Request %d successful", i+1)
	}
}

// Example_metrics demonstrates metrics collection
func Example_metrics() {
	metrics := NewMetricsCollector()
	client := NewClient().Intercept(metrics.Interceptor())

	// Make several requests
	for i := 0; i < 5; i++ {
		_, _ = client.GET("https://api.example.com/data").Send()
	}

	log.Printf("Total requests: %d", metrics.RequestCount)
	log.Printf("Average response time: %v", metrics.AverageResponseTime())
	log.Printf("Error rate: %.2f%%", metrics.ErrorRate())
}

// Example_conditionalRequests demonstrates conditional middleware
func Example_conditionalRequests() {
	client := NewClient().Use(
		ConditionalMiddleware(
			func(req *Request) bool {
				// Only add auth header for API requests
				return req.url != "" && req.url != "https://api.example.com"
			},
			func(req *Request) error {
				req.Header("Authorization", "Bearer token")
				return nil
			},
		),
	)

	// This request will have auth header
	_, _ = client.GET("https://api.example.com/data").Send()

	// This request won't have auth header
	_, _ = client.GET("https://public.example.com/data").Send()
}

// Example_chainedMiddleware demonstrates chaining multiple middleware
func Example_chainedMiddleware() {
	logger := log.New(os.Stdout, "HTTP: ", log.LstdFlags)

	middleware := ChainMiddleware(
		LoggingMiddleware(logger),
		UserAgentMiddleware("MyApp/1.0"),
		CompressionMiddleware(),
		TimeoutMiddleware(30*time.Second),
	)

	client := NewClient().Use(middleware)

	_, _ = client.GET("https://api.example.com/data").Send()
}

// Example_errorHandling demonstrates comprehensive error handling
func Example_errorHandling() {
	client := NewClient().
		Intercept(StatusCodeInterceptor(404, 403)). // Error on 404 and 403
		Intercept(JSONErrorInterceptor()).          // Parse JSON error responses
		Intercept(ServerErrorInterceptor())         // Error on 5xx responses

	resp, err := client.GET("https://api.example.com/nonexistent").Send()
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	if !resp.IsSuccess() {
		log.Printf("Request failed with status: %d", resp.StatusCode)
	}
}

// Example_customBackoff demonstrates custom backoff strategies
func Example_customBackoff() {
	// Custom backoff that increases delay based on attempt number
	customBackoff := NewCustomBackoff(func(attempt int) time.Duration {
		// Fibonacci-like backoff: 1s, 1s, 2s, 3s, 5s, 8s...
		if attempt == 0 {
			return time.Second
		}
		if attempt == 1 {
			return time.Second
		}

		prev, curr := time.Second, time.Second
		for i := 2; i <= attempt; i++ {
			prev, curr = curr, prev+curr
		}
		return curr
	})

	client := NewClient().Retry(&RetryConfig{
		MaxAttempts: 6,
		Backoff:     customBackoff,
		RetryIf:     DefaultRetryCondition,
	})

	_, _ = client.GET("https://unreliable-api.example.com").Send()
}

// Example_lruCache demonstrates LRU cache usage
func Example_lruCache() {
	// LRU cache with capacity of 100 items
	cache := NewLRUCache(100)
	client := NewClient().Cache(cache)

	// Make requests that will be cached
	for i := 0; i < 150; i++ {
		url := "https://api.example.com/data/" + string(rune(i))
		_, _ = client.GET(url).Cache(url, time.Hour).Send()
	}

	// First 50 requests should be evicted due to LRU policy
	log.Println("LRU cache automatically manages memory usage")
}

// Example_xmlHandling demonstrates XML request/response handling
func Example_xmlHandling() {
	type User struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	client := NewClient()

	user := User{Name: "John Doe", Email: "john@example.com"}

	resp, err := client.POST("https://api.example.com/users").
		XML(user).
		Send()

	if err != nil {
		log.Fatal(err)
	}

	var createdUser User
	if err := resp.XML(&createdUser); err != nil {
		log.Fatal(err)
	}

	log.Printf("Created user: %+v", createdUser)
}
