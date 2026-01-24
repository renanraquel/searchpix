package bb

import (
	"net/http"
	"sync"
	"time"
)

type TokenCache struct {
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func (c *TokenCache) GetToken(
	client *http.Client,
	tokenURL,
	clientID,
	clientSecret,
	scope string,
) (string, error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	// margem de segurança de 30s
	if c.token != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.token, nil
	}

	resp, err := GetAccessToken(
		client,
		tokenURL,
		clientID,
		clientSecret,
		scope,
	)
	if err != nil {
		return "", err
	}

	c.token = resp.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	return c.token, nil
}
