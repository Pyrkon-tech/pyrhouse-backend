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
	discordAuthURL  = "https://discord.com/oauth2/authorize"
	discordTokenURL = "https://discord.com/api/oauth2/token"
	discordUserURL  = "https://discord.com/api/users/@me"
)

type DiscordOAuth struct {
	config config.DiscordConfig
}

type DiscordUser struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	Avatar        *string `json:"avatar"`
	Email         *string `json:"email"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func NewDiscordOAuth(cfg config.DiscordConfig) *DiscordOAuth {
	return &DiscordOAuth{config: cfg}
}

// GetAuthURL zwraca URL do autoryzacji Discord
func (d *DiscordOAuth) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {d.config.ClientID},
		"redirect_uri":  {d.config.RedirectURI},
		"response_type": {"code"},
		"scope":         {"identify email"},
		"state":         {state},
	}
	return discordAuthURL + "?" + params.Encode()
}

// ExchangeCode wymienia authorization code na access token
func (d *DiscordOAuth) ExchangeCode(code string) (*TokenResponse, error) {
	data := url.Values{
		"client_id":     {d.config.ClientID},
		"client_secret": {d.config.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {d.config.RedirectURI},
	}

	resp, err := http.Post(discordTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord token error: %s", body)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// GetUser pobiera dane użytkownika z Discord API
func (d *DiscordOAuth) GetUser(accessToken string) (*DiscordUser, error) {
	req, err := http.NewRequest("GET", discordUserURL, nil)
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
		return nil, fmt.Errorf("discord user error: %s", body)
	}

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAvatarURL generuje URL do avatara Discord
func (d *DiscordOAuth) GetAvatarURL(user *DiscordUser) string {
	if user.Avatar == nil {
		return ""
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, *user.Avatar)
}

// IsConfigured sprawdza czy Discord OAuth jest skonfigurowany
func (d *DiscordOAuth) IsConfigured() bool {
	return d.config.ClientID != "" && d.config.ClientSecret != "" && d.config.RedirectURI != ""
}
