package bb

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func GetAccessToken(
	client *http.Client,
	tokenURL,
	clientID,
	clientSecret,
	scope string,
) (*TokenResponse, error) {

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", scope)

	req, err := http.NewRequest(
		http.MethodPost,
		tokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}

	basicAuth := base64.StdEncoding.EncodeToString(
		[]byte(clientID + ":" + clientSecret),
	)

	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}
