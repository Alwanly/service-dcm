package authentication

import (
	"encoding/base64"
	"strings"
)

type IBasicAuthService interface {
	// BasicAuth authenticates the user using basic auth.
	//
	// Parameters:
	//   - username: username
	//   - password: password
	//
	// Returns:
	//   - bool: true if the user is authenticated, false otherwise
	Validate(username, password string) bool

	// BasicAuth authenticates the user using basic auth.
	//
	// Parameters:
	//   - username: username
	//   - password: password
	//
	// Returns:
	//   - bool: true if the user is authenticated, false otherwise
	ValidateAdmin(username, password string) bool

	// DecodeBasicAuth decodes the basic auth header.
	//
	// Parameters:
	//   - auth: basic auth header
	//
	// Returns:
	//   - string: username
	//   - string: password
	DecodeFromHeader(auth string) (string, string)
}

type BasicAuthTConfig struct {
	// Username
	Username string

	// Password
	Password string

	// AdminUsername
	AdminUsername string

	// AdminPassword
	AdminPassword string
}

type basicAuth struct {
	username      string
	password      string
	adminUsername string
	adminPassword string
}

func NewBasicAuthService(config *BasicAuthTConfig) IBasicAuthService {
	return &basicAuth{
		username:      config.Username,
		password:      config.Password,
		adminUsername: config.AdminUsername,
		adminPassword: config.AdminPassword,
	}
}

func (b *basicAuth) Validate(username, password string) bool {
	return b.username == username && b.password == password
}

func (b *basicAuth) DecodeFromHeader(auth string) (string, string) {
	encoded := strings.TrimPrefix(auth, "Basic ")

	// Decode the Base64 string
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ""
	}

	// Split the decoded string into username and password
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", ""
	}

	return parts[0], parts[1]
}

func (b *basicAuth) ValidateAdmin(username, password string) bool {
	return b.adminUsername == username && b.adminPassword == password
}
