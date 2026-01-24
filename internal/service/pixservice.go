package service

import (
	"net/http"
	"searchpix/internal/bb"
	"searchpix/internal/config"
	"searchpix/internal/model"
)

type PixService struct {
	client     *http.Client // ou *http.Client
	cfg        *config.Config
	tokenCache *bb.TokenCache
}

func NewPixService(client *http.Client, cfg *config.Config, cache *bb.TokenCache) *PixService {
	return &PixService{client: client, cfg: cfg, tokenCache: cache}
}

func (s *PixService) BuscarPorPeriodo(inicio, fim string) (*model.PixResponse, error) {
	token, err := bb.GetAccessToken(
		s.client,
		s.cfg.BB.OAuthURL,
		s.cfg.BB.ClientID,
		s.cfg.BB.ClientSecret,
		s.cfg.BB.Scope,
	)
	if err != nil {
		return nil, err
	}

	inicioNormalizado, fimNormalizado, err := NormalizarPeriodo(inicio, fim)
	if err != nil {
		return nil, err
	}

	return bb.ConsultarPixPorPeriodo(
		s.client,
		s.cfg.BB.ApiBaseURL,
		token.AccessToken,
		s.cfg.BB.GwAppKey,
		inicioNormalizado,
		fimNormalizado,
	)
}
