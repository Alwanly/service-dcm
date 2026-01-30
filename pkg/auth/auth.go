package authentication

import (
	"encoding/base64"
	"strings"
)

type IBasicAuthService interface {
	Validate(username, password string) bool
	ValidateAdmin(username, password string) bool
	DecodeFromHeader(auth string) (string, string)
}

type BasicAuthTConfig struct {
	Username string

	Password string

	AdminUsername string

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
