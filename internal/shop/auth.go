package shop

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"warehouse/internal/oauth"
	"warehouse/internal/security"
	"warehouse/internal/shop/access"

	"github.com/gin-gonic/gin"
)

// GoogleAuth is the part of oauth.GoogleOAuth the shop login needs (fakeable in tests).
type GoogleAuth interface {
	ExchangeCodeWithURI(code, redirectURI string) (*oauth.GoogleTokenResponse, error)
	GetUser(accessToken string) (*oauth.GoogleUser, error)
}

// callbackPath is where the shop frontend receives the Google redirect.
const callbackPath = "/auth/google/callback"

type AuthService struct {
	repo    *Repository
	google  GoogleAuth
	shopURL string
}

func NewAuthService(repo *Repository, google GoogleAuth, shopURL string) *AuthService {
	return &AuthService{repo: repo, google: google, shopURL: strings.TrimRight(shopURL, "/")}
}

type loginInput struct {
	Code        string `json:"code" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
	InviteToken string `json:"invite_token"`
}

// Login exchanges a Google authorization code for a shop token, running the access policy chain.
func (s *AuthService) Login(ctx context.Context, in loginInput) (string, *Account, error) {
	if s.google == nil || s.shopURL == "" {
		return "", nil, newErr(http.StatusServiceUnavailable, "shop_login_unavailable", "Logowanie do sklepu nie jest skonfigurowane")
	}
	// Only the shop's own callback may receive codes exchanged here.
	if in.RedirectURI != s.shopURL+callbackPath {
		return "", nil, badRequest("invalid_redirect_uri", "Niedozwolony redirect_uri")
	}

	token, err := s.google.ExchangeCodeWithURI(in.Code, in.RedirectURI)
	if err != nil {
		return "", nil, newErr(http.StatusBadRequest, "google_exchange_failed", "Nie udało się zalogować przez Google")
	}
	gu, err := s.google.GetUser(token.AccessToken)
	if err != nil {
		return "", nil, newErr(http.StatusBadGateway, "google_userinfo_failed", "Nie udało się pobrać danych konta Google")
	}

	var accountID int
	var policy string
	err = s.repo.inTx(ctx, func(tx *sql.Tx) error {
		settings, err := s.repo.loadSettings(ctx, tx)
		if err != nil {
			return err
		}
		d, err := access.Resolve(accessStore{ctx: ctx, tx: tx},
			access.DefaultChain(access.Settings{DomainAutoJoin: settings.DomainAutoJoin, AutoDomains: settings.AutoDomains}),
			access.Request{
				Identity: access.Identity{
					Sub:           gu.Sub,
					Email:         gu.Email,
					EmailVerified: gu.EmailVerified,
					HD:            gu.HD,
				},
				InviteToken: in.InviteToken,
			})
		if err != nil {
			return err
		}
		accountID, policy = d.Account.ID, d.Policy
		return s.repo.touchLogin(ctx, tx, accountID, gu.Name, gu.Picture)
	})
	if err != nil {
		if mapped := mapAccessError(err); mapped != nil {
			log.Printf("[shop] login refused for %s: %v", access.NormalizeEmail(gu.Email), err)
			return "", nil, mapped
		}
		return "", nil, err
	}

	jwt, err := security.GenerateShopJWT(accountID)
	if err != nil {
		return "", nil, err
	}
	acc, err := s.repo.accountByID(ctx, accountID)
	if err != nil {
		return "", nil, err
	}
	log.Printf("[shop] account %d logged in (policy %s)", accountID, policy)
	return jwt, acc, nil
}

func mapAccessError(err error) *Error {
	switch {
	case errors.Is(err, access.ErrEmailNotVerified):
		return newErr(http.StatusForbidden, "email_not_verified", "Adres e-mail konta Google nie jest zweryfikowany")
	case errors.Is(err, access.ErrInactive):
		return newErr(http.StatusForbidden, "account_inactive", "Konto w sklepie jest zablokowane — skontaktuj się z magazynem")
	case errors.Is(err, access.ErrInviteInvalid):
		return newErr(http.StatusForbidden, "invite_invalid", "Zaproszenie jest nieważne, wykorzystane albo wygasło — poproś o nowe")
	case errors.Is(err, access.ErrDenied):
		return newErr(http.StatusForbidden, "access_denied", "To konto nie ma dostępu do sklepu — poproś magazyn o zaproszenie")
	}
	return nil
}

const ctxAccountID = "shopAccountID"

// ShopAuth authenticates organizer requests: only shop tokens (aud=pyrhouse-shop) pass, and the
// account is re-read on every request so blocking it takes effect immediately.
func ShopAuth(repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if raw == "" {
			abort(c, newErr(http.StatusUnauthorized, "unauthorized", "Brak tokenu"))
			return
		}
		accountID, err := security.ParseShopToken(raw)
		if err != nil {
			abort(c, newErr(http.StatusUnauthorized, "unauthorized", "Nieprawidłowy token"))
			return
		}
		acc, err := repo.accountByID(c.Request.Context(), accountID)
		if err != nil {
			abort(c, err)
			return
		}
		if acc == nil {
			abort(c, newErr(http.StatusUnauthorized, "unauthorized", "Konto nie istnieje"))
			return
		}
		if !acc.Active {
			abort(c, newErr(http.StatusForbidden, "account_inactive", "Konto w sklepie jest zablokowane"))
			return
		}
		c.Set(ctxAccountID, acc.ID)
		c.Next()
	}
}

func accountID(c *gin.Context) int {
	return c.GetInt(ctxAccountID)
}
