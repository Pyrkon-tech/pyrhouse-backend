package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"warehouse/internal/config"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

type GoogleOAuth struct {
	config config.GoogleConfig
}

type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type GoogleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

func NewGoogleOAuth(cfg config.GoogleConfig) *GoogleOAuth {
	return &GoogleOAuth{config: cfg}
}

func (g *GoogleOAuth) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {g.config.ClientID},
		"redirect_uri":  {g.config.RedirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"online"},
	}
	return googleAuthURL + "?" + params.Encode()
}

func (g *GoogleOAuth) ExchangeCodeWithURI(code, redirectURI string) (*GoogleTokenResponse, error) {
	data := url.Values{
		"client_id":     {g.config.ClientID},
		"client_secret": {g.config.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.Post(googleTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google token error: %s", body)
	}

	var token GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (g *GoogleOAuth) GetUser(accessToken string) (*GoogleUser, error) {
	req, err := http.NewRequest("GET", googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo error: %s", body)
	}

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (g *GoogleOAuth) IsConfigured() bool {
	return g.config.ClientID != "" && g.config.ClientSecret != ""
}

func (g *GoogleOAuth) GetFrontendURL() string {
	return g.config.FrontendURL
}
