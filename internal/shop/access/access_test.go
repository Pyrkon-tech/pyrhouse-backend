package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInvite struct {
	id     int
	usable bool
	usedBy int
}

type fakeStore struct {
	accounts []*Account
	invites  map[string]*fakeInvite // by token hash
	nextID   int
}

func (s *fakeStore) FindBySub(sub string) (*Account, error) {
	for _, a := range s.accounts {
		if a.GoogleSub != nil && *a.GoogleSub == sub {
			return a, nil
		}
	}
	return nil, nil
}

func (s *fakeStore) FindByEmail(email string) (*Account, error) {
	for _, a := range s.accounts {
		if a.Email == email {
			return a, nil
		}
	}
	return nil, nil
}

func (s *fakeStore) BindGoogleSub(accountID int, sub string) error {
	for _, a := range s.accounts {
		if a.ID == accountID {
			a.GoogleSub = &sub
		}
	}
	return nil
}

func (s *fakeStore) ClaimInvite(tokenHash string) (int, bool, error) {
	inv, ok := s.invites[tokenHash]
	if !ok || !inv.usable {
		return 0, false, nil
	}
	inv.usable = false
	return inv.id, true, nil
}

func (s *fakeStore) MarkInviteUsedBy(inviteID, accountID int) error {
	for _, inv := range s.invites {
		if inv.id == inviteID {
			inv.usedBy = accountID
		}
	}
	return nil
}

func (s *fakeStore) CreateAccount(email, sub string, source Source) (*Account, error) {
	s.nextID++
	a := &Account{ID: 100 + s.nextID, Email: email, GoogleSub: &sub, Active: true, AccessSource: source}
	s.accounts = append(s.accounts, a)
	return a, nil
}

func strPtr(s string) *string { return &s }

const (
	validInvite   = "valid-invite-token"
	usedInvite    = "used-invite-token"
	missingInvite = "no-such-invite"
)

func newStore() *fakeStore {
	return &fakeStore{
		accounts: []*Account{
			{ID: 1, Email: "known@gmail.com", GoogleSub: strPtr("sub-known"), Active: true, AccessSource: SourceInvite},
			{ID: 2, Email: "blocked@pyrkon.pl", GoogleSub: strPtr("sub-blocked"), Active: false, AccessSource: SourceDomain},
			{ID: 3, Email: "listed@gmail.com", Active: true, AccessSource: SourceAllowlist},
			{ID: 4, Email: "listed-blocked@gmail.com", Active: false, AccessSource: SourceAllowlist},
		},
		invites: map[string]*fakeInvite{
			HashInviteToken(validInvite): {id: 10, usable: true},
			HashInviteToken(usedInvite):  {id: 11, usable: false},
		},
	}
}

var pyrkonSettings = Settings{DomainAutoJoin: true, AutoDomains: []string{"pyrkon.pl"}}

func TestResolve(t *testing.T) {
	gmail := func(sub, email string) Identity {
		return Identity{Sub: sub, Email: email, EmailVerified: true}
	}
	workspace := func(sub, email, hd string) Identity {
		return Identity{Sub: sub, Email: email, EmailVerified: true, HD: hd}
	}

	tests := []struct {
		name        string
		settings    Settings
		req         Request
		wantErr     error
		wantPolicy  string
		wantCreated bool
		wantSource  Source
		check       func(t *testing.T, s *fakeStore)
	}{
		{
			name:     "unverified email is refused before any policy",
			settings: pyrkonSettings,
			req:      Request{Identity: Identity{Sub: "sub-known", Email: "known@gmail.com", EmailVerified: false}},
			wantErr:  ErrEmailNotVerified,
		},
		{
			name:       "existing active account",
			settings:   pyrkonSettings,
			req:        Request{Identity: gmail("sub-known", "known@gmail.com")},
			wantPolicy: "existing_account",
		},
		{
			name:     "existing inactive account is blocked even with a valid invite",
			settings: pyrkonSettings,
			req:      Request{Identity: workspace("sub-blocked", "blocked@pyrkon.pl", "pyrkon.pl"), InviteToken: validInvite},
			wantErr:  ErrInactive,
			check: func(t *testing.T, s *fakeStore) {
				assert.True(t, s.invites[HashInviteToken(validInvite)].usable, "invite must not be consumed")
			},
		},
		{
			name:       "existing account does not consume an invite",
			settings:   pyrkonSettings,
			req:        Request{Identity: gmail("sub-known", "known@gmail.com"), InviteToken: validInvite},
			wantPolicy: "existing_account",
			check: func(t *testing.T, s *fakeStore) {
				assert.True(t, s.invites[HashInviteToken(validInvite)].usable)
			},
		},
		{
			name:       "allowlisted e-mail binds the Google identity on first login",
			settings:   pyrkonSettings,
			req:        Request{Identity: gmail("sub-new", "Listed@Gmail.com ")},
			wantPolicy: "allowlist",
			check: func(t *testing.T, s *fakeStore) {
				acc, _ := s.FindBySub("sub-new")
				require.NotNil(t, acc)
				assert.Equal(t, 3, acc.ID)
			},
		},
		{
			name:     "allowlisted but deactivated before first login",
			settings: pyrkonSettings,
			req:      Request{Identity: gmail("sub-new", "listed-blocked@gmail.com")},
			wantErr:  ErrInactive,
		},
		{
			name:     "e-mail bound to another Google identity is refused, no duplicate account",
			settings: pyrkonSettings,
			req:      Request{Identity: gmail("sub-other", "known@gmail.com"), InviteToken: validInvite},
			wantErr:  ErrDenied,
			check: func(t *testing.T, s *fakeStore) {
				assert.Len(t, s.accounts, 4)
				assert.True(t, s.invites[HashInviteToken(validInvite)].usable)
			},
		},
		{
			name:        "valid invite admits a consumer account and is consumed",
			settings:    pyrkonSettings,
			req:         Request{Identity: gmail("sub-new", "guest@gmail.com"), InviteToken: validInvite},
			wantPolicy:  "invite",
			wantCreated: true,
			wantSource:  SourceInvite,
			check: func(t *testing.T, s *fakeStore) {
				inv := s.invites[HashInviteToken(validInvite)]
				assert.False(t, inv.usable)
				assert.NotZero(t, inv.usedBy)
			},
		},
		{
			name:     "used invite without another way in",
			settings: pyrkonSettings,
			req:      Request{Identity: gmail("sub-new", "guest@gmail.com"), InviteToken: usedInvite},
			wantErr:  ErrInviteInvalid,
		},
		{
			name:     "unknown invite without another way in",
			settings: pyrkonSettings,
			req:      Request{Identity: gmail("sub-new", "guest@gmail.com"), InviteToken: missingInvite},
			wantErr:  ErrInviteInvalid,
		},
		{
			name:        "used invite but the domain lets the person in",
			settings:    pyrkonSettings,
			req:         Request{Identity: workspace("sub-new", "jan@pyrkon.pl", "pyrkon.pl"), InviteToken: usedInvite},
			wantPolicy:  "domain",
			wantCreated: true,
			wantSource:  SourceDomain,
		},
		{
			name:        "workspace account of an auto domain",
			settings:    pyrkonSettings,
			req:         Request{Identity: workspace("sub-new", "jan@pyrkon.pl", "PYRKON.PL")},
			wantPolicy:  "domain",
			wantCreated: true,
			wantSource:  SourceDomain,
		},
		{
			name:     "domain auto-join switched off",
			settings: Settings{DomainAutoJoin: false, AutoDomains: []string{"pyrkon.pl"}},
			req:      Request{Identity: workspace("sub-new", "jan@pyrkon.pl", "pyrkon.pl")},
			wantErr:  ErrDenied,
		},
		{
			name:     "consumer account on a domain address has no hd and is refused",
			settings: pyrkonSettings,
			req:      Request{Identity: gmail("sub-new", "jan@pyrkon.pl")},
			wantErr:  ErrDenied,
		},
		{
			name:     "hd of a domain that is not on the list",
			settings: pyrkonSettings,
			req:      Request{Identity: workspace("sub-new", "jan@other.pl", "other.pl")},
			wantErr:  ErrDenied,
		},
		{
			name:     "hd does not match the e-mail domain",
			settings: pyrkonSettings,
			req:      Request{Identity: workspace("sub-new", "jan@other.pl", "pyrkon.pl")},
			wantErr:  ErrDenied,
		},
		{
			name:     "plain consumer account",
			settings: pyrkonSettings,
			req:      Request{Identity: gmail("sub-new", "random@gmail.com")},
			wantErr:  ErrDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore()
			d, err := Resolve(store, DefaultChain(tt.settings), tt.req)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, d)
			} else {
				require.NoError(t, err)
				require.NotNil(t, d)
				assert.Equal(t, tt.wantPolicy, d.Policy)
				assert.Equal(t, tt.wantCreated, d.Created)
				assert.True(t, d.Account.Active)
				require.NotNil(t, d.Account.GoogleSub)
				assert.Equal(t, tt.req.Identity.Sub, *d.Account.GoogleSub)
				if tt.wantSource != "" {
					assert.Equal(t, tt.wantSource, d.Account.AccessSource)
				}
			}
			if tt.check != nil {
				tt.check(t, store)
			}
		})
	}
}

func TestParseDomains(t *testing.T) {
	assert.Equal(t, []string{"pyrkon.pl", "example.org"}, ParseDomains(" Pyrkon.pl, ,example.org "))
	assert.Nil(t, ParseDomains(""))
}
