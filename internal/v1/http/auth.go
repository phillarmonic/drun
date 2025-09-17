package http

import (
	"encoding/base64"
	"fmt"
)

// Auth represents different authentication methods
type Auth interface {
	Apply(client *Client) *Client
}

// BasicAuth implements HTTP Basic Authentication
type BasicAuth struct {
	Username string
	Password string
}

// Apply applies basic authentication to the client
func (a *BasicAuth) Apply(client *Client) *Client {
	credentials := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
	return client.Header("Authorization", "Basic "+credentials)
}

// BearerAuth implements Bearer token authentication
type BearerAuth struct {
	Token string
}

// Apply applies bearer authentication to the client
func (a *BearerAuth) Apply(client *Client) *Client {
	return client.Header("Authorization", "Bearer "+a.Token)
}

// APIKeyAuth implements API key authentication
type APIKeyAuth struct {
	Key      string
	Value    string
	InHeader bool // true for header, false for query parameter
}

// Apply applies API key authentication to the client
func (a *APIKeyAuth) Apply(client *Client) *Client {
	if a.InHeader {
		return client.Header(a.Key, a.Value)
	}
	return client.Query(a.Key, a.Value)
}

// CustomAuth allows custom authentication logic
type CustomAuth struct {
	ApplyFunc func(*Client) *Client
}

// Apply applies custom authentication to the client
func (a *CustomAuth) Apply(client *Client) *Client {
	return a.ApplyFunc(client)
}

// OAuth2Auth implements OAuth2 authentication
type OAuth2Auth struct {
	AccessToken string
	TokenType   string // defaults to "Bearer"
}

// Apply applies OAuth2 authentication to the client
func (a *OAuth2Auth) Apply(client *Client) *Client {
	tokenType := a.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return client.Header("Authorization", fmt.Sprintf("%s %s", tokenType, a.AccessToken))
}

// DigestAuth implements HTTP Digest Authentication (basic implementation)
type DigestAuth struct {
	Username string
	Password string
}

// Apply applies digest authentication to the client
func (a *DigestAuth) Apply(client *Client) *Client {
	// Note: This is a simplified implementation
	// Full digest auth requires challenge-response handling
	return client.Use(func(req *Request) error {
		// In a real implementation, this would handle the digest challenge
		// For now, we'll add it as a middleware placeholder
		req.Header("Authorization", fmt.Sprintf("Digest username=\"%s\"", a.Username))
		return nil
	})
}

// Helper functions for creating auth instances

// Basic creates a new BasicAuth instance
func Basic(username, password string) *BasicAuth {
	return &BasicAuth{Username: username, Password: password}
}

// Bearer creates a new BearerAuth instance
func Bearer(token string) *BearerAuth {
	return &BearerAuth{Token: token}
}

// APIKey creates a new APIKeyAuth instance for headers
func APIKey(key, value string) *APIKeyAuth {
	return &APIKeyAuth{Key: key, Value: value, InHeader: true}
}

// APIKeyQuery creates a new APIKeyAuth instance for query parameters
func APIKeyQuery(key, value string) *APIKeyAuth {
	return &APIKeyAuth{Key: key, Value: value, InHeader: false}
}

// OAuth2 creates a new OAuth2Auth instance
func OAuth2(accessToken string) *OAuth2Auth {
	return &OAuth2Auth{AccessToken: accessToken}
}

// OAuth2WithType creates a new OAuth2Auth instance with custom token type
func OAuth2WithType(accessToken, tokenType string) *OAuth2Auth {
	return &OAuth2Auth{AccessToken: accessToken, TokenType: tokenType}
}

// Digest creates a new DigestAuth instance
func Digest(username, password string) *DigestAuth {
	return &DigestAuth{Username: username, Password: password}
}

// Custom creates a new CustomAuth instance
func Custom(applyFunc func(*Client) *Client) *CustomAuth {
	return &CustomAuth{ApplyFunc: applyFunc}
}
