package bb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"searchpix/internal/model"
)

func ConsultarPixPorPeriodo(
	client *http.Client,
	baseURL,
	token,
	gwDevAppKey,
	inicio,
	fim string,
) (*model.PixResponse, error) {

	u, err := url.Parse(baseURL + "/pix/v2/pix")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("inicio", inicio)
	q.Set("fim", fim)
	q.Set("gw-dev-app-key", gwDevAppKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var bbErr model.BBErrorResponse

		err := json.NewDecoder(resp.Body).Decode(&bbErr)
		if err != nil {
			return nil, fmt.Errorf("erro ao consultar BB (status %d)", resp.StatusCode)
		}

		return nil, fmt.Errorf(bbErr.Detail)
	}

	var result model.PixResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
