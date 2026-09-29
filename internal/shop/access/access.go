// Package access decides who may enter the organizer shop (docs/shop/PLAN.md, D4 and D12).
//
// Access is a chain of policies tried in order; the first one that handles the identity wins.
// The default chain is ExistingAccount → Allowlist → Invite → Domain. Which domains get in
// automatically, and whether they do at all, comes from app_settings, not from code.
package access

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// Source records how an account got access (shop_accounts.access_source).
type Source string

const (
	SourceDomain    Source = "domain"
	SourceAllowlist Source = "allowlist"
	SourceInvite    Source = "invite"
)

var (
	ErrEmailNotVerified = errors.New("google email is not verified")
	ErrInactive         = errors.New("shop account is inactive")
	ErrInviteInvalid    = errors.New("invite is invalid, used, revoked or expired")
	ErrDenied           = errors.New("no access policy admits this account")
)

// Identity is what Google tells us about the person logging in.
type Identity struct {
	Sub           string
	Email         string
	EmailVerified bool
	HD            string // Google Workspace domain; empty for consumer accounts
}

// Account is the part of shop_accounts the policies need.
type Account struct {
	ID           int
	Email        string
	GoogleSub    *string // nil = allowlist entry that has not logged in yet
	Active       bool
	AccessSource Source
}

// Settings drive the Domain policy.
type Settings struct {
	DomainAutoJoin bool
	AutoDomains    []string
}

// Store is the persistence the policies use. Resolve is meant to run inside one transaction,
// so an invite is never burned without its account being created.
type Store interface {
	FindBySub(sub string) (*Account, error)     // nil, nil when absent
	FindByEmail(email string) (*Account, error) // nil, nil when absent
	BindGoogleSub(accountID int, sub string) error
	// ClaimInvite atomically marks a usable invite as used and returns its ID; ok=false when the
	// invite does not exist or is used, revoked or expired.
	ClaimInvite(tokenHash string) (inviteID int, ok bool, err error)
	MarkInviteUsedBy(inviteID, accountID int) error
	CreateAccount(email, sub string, source Source) (*Account, error)
}

// Request is one login attempt.
type Request struct {
	Identity    Identity
	InviteToken string // plaintext from the invite link, optional
}

// Decision is the outcome of a successful login.
type Decision struct {
	Account *Account
	Policy  string // name of the policy that admitted the account
	Created bool   // a new shop account was created
}

// Policy handles a request or passes it on. handled=false means "not mine, try the next one".
type Policy interface {
	Name() string
	Decide(store Store, req Request, st *state) (d *Decision, handled bool, err error)
}

// state carries facts between policies within one Resolve call.
type state struct {
	inviteRejected bool
}

// DefaultChain is the policy order from the plan.
func DefaultChain(settings Settings) []Policy {
	return []Policy{ExistingAccount{}, Allowlist{}, Invite{}, Domain{Settings: settings}}
}

// Resolve runs the chain for one login attempt.
func Resolve(store Store, chain []Policy, req Request) (*Decision, error) {
	req.Identity.Email = NormalizeEmail(req.Identity.Email)
	if !req.Identity.EmailVerified {
		return nil, ErrEmailNotVerified
	}
	if req.Identity.Sub == "" || req.Identity.Email == "" {
		return nil, ErrDenied
	}

	st := &state{}
	for _, p := range chain {
		d, handled, err := p.Decide(store, req, st)
		if err != nil {
			return nil, err
		}
		if handled {
			d.Policy = p.Name()
			return d, nil
		}
	}
	if st.inviteRejected {
		return nil, ErrInviteInvalid
	}
	return nil, ErrDenied
}

// ExistingAccount admits an account already bound to this Google identity.
type ExistingAccount struct{}

func (ExistingAccount) Name() string { return "existing_account" }

func (ExistingAccount) Decide(store Store, req Request, _ *state) (*Decision, bool, error) {
	acc, err := store.FindBySub(req.Identity.Sub)
	if err != nil || acc == nil {
		return nil, false, err
	}
	if !acc.Active {
		return nil, true, ErrInactive
	}
	return &Decision{Account: acc}, true, nil
}

// Allowlist admits an e-mail an admin added in advance and binds it to the Google identity on
// first login. An e-mail already bound to a different Google identity is refused outright:
// neither an invite nor the domain may create a second account for it.
type Allowlist struct{}

func (Allowlist) Name() string { return "allowlist" }

func (Allowlist) Decide(store Store, req Request, _ *state) (*Decision, bool, error) {
	acc, err := store.FindByEmail(req.Identity.Email)
	if err != nil || acc == nil {
		return nil, false, err
	}
	if acc.GoogleSub != nil {
		return nil, true, ErrDenied
	}
	if !acc.Active {
		return nil, true, ErrInactive
	}
	if err := store.BindGoogleSub(acc.ID, req.Identity.Sub); err != nil {
		return nil, true, err
	}
	sub := req.Identity.Sub
	acc.GoogleSub = &sub
	return &Decision{Account: acc}, true, nil
}

// Invite admits anyone (any Google account) holding a valid one-time invite.
type Invite struct{}

func (Invite) Name() string { return "invite" }

func (Invite) Decide(store Store, req Request, st *state) (*Decision, bool, error) {
	if strings.TrimSpace(req.InviteToken) == "" {
		return nil, false, nil
	}
	inviteID, ok, err := store.ClaimInvite(HashInviteToken(req.InviteToken))
	if err != nil {
		return nil, false, err
	}
	if !ok {
		st.inviteRejected = true
		return nil, false, nil
	}
	acc, err := store.CreateAccount(req.Identity.Email, req.Identity.Sub, SourceInvite)
	if err != nil {
		return nil, true, err
	}
	if err := store.MarkInviteUsedBy(inviteID, acc.ID); err != nil {
		return nil, true, err
	}
	return &Decision{Account: acc, Created: true}, true, nil
}

// Domain admits Google Workspace accounts of the configured domains. It trusts the hd claim,
// not the e-mail suffix: anyone can create a consumer Google account on any address.
type Domain struct {
	Settings Settings
}

func (Domain) Name() string { return "domain" }

func (p Domain) Decide(store Store, req Request, _ *state) (*Decision, bool, error) {
	hd := strings.ToLower(strings.TrimSpace(req.Identity.HD))
	if !p.Settings.DomainAutoJoin || hd == "" || !containsFold(p.Settings.AutoDomains, hd) {
		return nil, false, nil
	}
	if !strings.HasSuffix(req.Identity.Email, "@"+hd) {
		return nil, false, nil
	}
	acc, err := store.CreateAccount(req.Identity.Email, req.Identity.Sub, SourceDomain)
	if err != nil {
		return nil, true, err
	}
	return &Decision{Account: acc, Created: true}, true, nil
}

// HashInviteToken is how invite tokens are stored (shop_invites.token_hash).
func HashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// NormalizeEmail lower-cases and trims an e-mail (shop_accounts.email is stored lower-case).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ParseDomains splits the comma-separated shop.auth.auto_domains setting.
func ParseDomains(raw string) []string {
	var out []string
	for _, d := range strings.Split(raw, ",") {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			out = append(out, d)
		}
	}
	return out
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(strings.TrimSpace(x), v) {
			return true
		}
	}
	return false
}
