package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func generateTokenID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "tok_" + hex.EncodeToString(b)
}

func generateTokenValue() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (c *Config) CreateToken(name string) (string, error) {
	value, err := generateTokenValue()
	if err != nil {
		return "", err
	}

	token := Token{
		ID:        generateTokenID(),
		Name:      name,
		Value:     value,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		LastUsedAt: "",
	}

	c.Tokens = append(c.Tokens, token)

	if err := c.Save(); err != nil {
		return "", fmt.Errorf("saving config: %w", err)
	}

	return value, nil
}

func (c *Config) ValidateToken(value string) bool {
	// Simple direct comparison - tokens are stored in plaintext in the config file
	for i := range c.Tokens {
		if c.Tokens[i].Value == value {
			c.Tokens[i].LastUsedAt = time.Now().UTC().Format(time.RFC3339)
			_ = c.Save()
			return true
		}
	}
	return false
}

func (c *Config) RevokeToken(id string) bool {
	for i, t := range c.Tokens {
		if t.ID == id {
			c.Tokens = append(c.Tokens[:i], c.Tokens[i+1:]...)
			_ = c.Save()
			return true
		}
	}
	return false
}

func (c *Config) ListTokens() []Token {
	result := make([]Token, len(c.Tokens))
	copy(result, c.Tokens)
	return result
}

func (c *Config) EnsureDefaultToken() (string, bool, error) {
	if len(c.Tokens) > 0 {
		return "", false, nil
	}

	value, err := c.CreateToken("default")
	if err != nil {
		return "", false, err
	}

	return value, true, nil
}
