package shop

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"warehouse/internal/config"
	"warehouse/internal/oauth"
	"warehouse/internal/rate_limiter"
	"warehouse/internal/security"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every row these tests create carries a marker (emails "shoptest-…", names/labels "__TEST__…")
// because all packages share the pyrhouse_test database and run in parallel.

const (
	testShopURL  = "https://shop.test.invalid"
	testVerifier = "0123456789abcdef0123456789abcdef0123456789abcdef" // PKCE code_verifier, 48 chars
)

type fakeGoogle struct {
	users map[string]oauth.GoogleUser // by authorization code
}

func (f *fakeGoogle) ExchangeCodeWithPKCE(code, _, _ string) (*oauth.GoogleTokenResponse, error) {
	if _, ok := f.users[code]; !ok {
		return nil, fmt.Errorf("bad code")
	}
	return &oauth.GoogleTokenResponse{AccessToken: code}, nil
}

func (f *fakeGoogle) GetUser(accessToken string) (*oauth.GoogleUser, error) {
	u := f.users[accessToken]
	return &u, nil
}

type shopEnv struct {
	t          *testing.T
	db         *sql.DB
	router     *gin.Engine
	google     *fakeGoogle
	modToken   string
	modID      int
	locationID int
	location2  int
	categoryID int
	products   []int
	delivery   int
	ret        int
}

func shopTestDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

func cleanupShopData(db *sql.DB) {
	testOrders := `SELECT o.id FROM shop_orders o JOIN shop_accounts a ON a.id = o.account_id WHERE a.email LIKE 'shoptest-%'`
	_, _ = db.Exec(`DELETE FROM equipment_request_items WHERE quest_id IN (SELECT id FROM equipment_request_quests WHERE shop_order_id IN (` + testOrders + `))`)
	_, _ = db.Exec(`DELETE FROM equipment_request_quests WHERE shop_order_id IN (` + testOrders + `)`)
	_, _ = db.Exec(`DELETE FROM shop_orders WHERE id IN (` + testOrders + `)`)
	_, _ = db.Exec(`DELETE FROM shop_invites WHERE label LIKE '__TEST__%'`)
	_, _ = db.Exec(`DELETE FROM shop_accounts WHERE email LIKE 'shoptest-%'`)
	_, _ = db.Exec(`DELETE FROM shop_products WHERE name LIKE '__TEST__%'`)
	_, _ = db.Exec(`DELETE FROM shop_delivery_windows WHERE label LIKE '__TEST__%'`)
	_, _ = db.Exec(`DELETE FROM item_category WHERE label LIKE '__TEST__Shop%'`)
	_, _ = db.Exec(`DELETE FROM locations WHERE name LIKE '__TEST__Shop%'`)
	_, _ = db.Exec(`DELETE FROM users WHERE username LIKE '__test__shop%'`)
}

func setupShop(t *testing.T) *shopEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, err := sql.Open("postgres", shopTestDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	var migrated bool
	require.NoError(t, db.QueryRow(`SELECT to_regclass('shop_orders') IS NOT NULL`).Scan(&migrated))
	if !migrated {
		t.Skip("pyrhouse_test is not migrated to the shop schema")
	}
	cleanupShopData(db)
	t.Cleanup(func() { cleanupShopData(db); _ = db.Close() })

	require.NoError(t, security.Initialize(config.JWTConfig{Secret: "shop-test-secret", Expiration: time.Hour, ShopExpiration: time.Hour}))

	env := &shopEnv{t: t, db: db, google: &fakeGoogle{users: map[string]oauth.GoogleUser{}}}

	must := func(q string, args ...any) int {
		var id int
		require.NoError(t, db.QueryRow(q, args...).Scan(&id), q)
		return id
	}
	env.modID = must(`INSERT INTO users (username, role, active) VALUES ('__test__shop_mod', 'moderator', true) RETURNING id`)
	env.modToken, err = security.GenerateJWT(strconv.Itoa(env.modID), "moderator", "__test__shop_mod")
	require.NoError(t, err)

	env.locationID = must(`INSERT INTO locations (name, pavilion) VALUES ('__TEST__ShopHall', '5') RETURNING id`)
	env.location2 = must(`INSERT INTO locations (name, pavilion) VALUES ('__TEST__ShopStage', '3') RETURNING id`)
	env.categoryID = must(`INSERT INTO item_category (item_category, label, pyr_id, category_type)
		VALUES ('__test__shop_cat', '__TEST__ShopCat', 'SH99', 'stock')
		ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id`)
	env.products = []int{
		must(`INSERT INTO shop_products (name, category_id, price, max_per_order, section) VALUES ('__TEST__Laptop', $1, 150.00, 5, 'IT') RETURNING id`, env.categoryID),
		must(`INSERT INTO shop_products (name, category_id, section) VALUES ('__TEST__Kabel', $1, 'Kable') RETURNING id`, env.categoryID),
	}
	env.delivery = must(`INSERT INTO shop_delivery_windows (kind, starts_at, ends_at, label)
		VALUES ('delivery', now() + interval '10 days', now() + interval '10 days 4 hours', '__TEST__czw') RETURNING id`)
	env.ret = must(`INSERT INTO shop_delivery_windows (kind, starts_at, ends_at, label)
		VALUES ('return', now() + interval '13 days', now() + interval '13 days 4 hours', '__TEST__nd') RETURNING id`)

	env.setSettings(map[string]string{
		settingShowPrices: "true", settingOrdersOpen: "true", settingOrdersOpenUntil: "",
		settingDomainAutoJoin: "true", settingAutoDomains: "pyrkon.pl",
	})

	gin.SetMode(gin.TestMode)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, NewAuthService(repo, env.google, testShopURL), repo, testShopURL)
	h.loginLimiter = rate_limiter.NewRateLimiter(1000, time.Minute)
	r := gin.New()
	h.RegisterShopRoutes(r)
	protected := r.Group("")
	protected.Use(security.JWTMiddleware())
	h.RegisterAdminRoutes(protected)
	env.router = r
	return env
}

func (e *shopEnv) setSettings(values map[string]string) {
	for k, v := range values {
		_, err := e.db.Exec(`INSERT INTO app_settings (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, k, v)
		require.NoError(e.t, err)
	}
}

type resp struct {
	Code int
	Body map[string]any
	Raw  []byte
}

func (e *shopEnv) do(method, path, token string, body any, headers ...string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(e.t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	r := resp{Code: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(w.Body.Bytes(), &r.Body)
	return r
}

// login signs a Google identity into the shop and returns the shop token.
func (e *shopEnv) login(u oauth.GoogleUser, invite string) resp {
	code := "code-" + u.Sub
	e.google.users[code] = u
	return e.do(http.MethodPost, "/shop/auth/google/exchange", "", map[string]any{
		"code": code, "redirect_uri": testShopURL + "/auth/google/callback", "invite_token": invite,
		"code_verifier": testVerifier,
	})
}

func (e *shopEnv) domainLogin(name string) string {
	e.t.Helper()
	r := e.login(oauth.GoogleUser{Sub: "sub-" + name, Email: "shoptest-" + name + "@pyrkon.pl", EmailVerified: true, HD: "pyrkon.pl"}, "")
	require.Equal(e.t, http.StatusOK, r.Code, string(r.Raw))
	return r.Body["token"].(string)
}

func (e *shopEnv) orderBody(items ...map[string]any) map[string]any {
	if len(items) == 0 {
		items = []map[string]any{{"product_id": e.products[0], "quantity": 2}, {"product_id": e.products[1], "quantity": 4}}
	}
	return map[string]any{
		"location_id": e.locationID, "contact_name": "Jan Organizator", "contact_phone": "600100200",
		"delivery_window_id": e.delivery, "return_window_id": e.ret, "items": items,
	}
}

func num(v any) int { return int(v.(float64)) }

func TestShop_LoginPolicies(t *testing.T) {
	e := setupShop(t)

	t.Run("wrong redirect_uri", func(t *testing.T) {
		e.google.users["c1"] = oauth.GoogleUser{Sub: "s1", Email: "shoptest-a@pyrkon.pl", EmailVerified: true, HD: "pyrkon.pl"}
		r := e.do(http.MethodPost, "/shop/auth/google/exchange", "", map[string]any{"code": "c1", "redirect_uri": "https://evil.example/cb", "code_verifier": testVerifier})
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.Equal(t, "invalid_redirect_uri", r.Body["code"])
	})

	t.Run("PKCE code_verifier is required", func(t *testing.T) {
		e.google.users["c2"] = oauth.GoogleUser{Sub: "s2", Email: "shoptest-b@pyrkon.pl", EmailVerified: true, HD: "pyrkon.pl"}
		r := e.do(http.MethodPost, "/shop/auth/google/exchange", "", map[string]any{"code": "c2", "redirect_uri": testShopURL + "/auth/google/callback"})
		assert.Equal(t, http.StatusBadRequest, r.Code)
	})

	t.Run("unverified e-mail", func(t *testing.T) {
		r := e.login(oauth.GoogleUser{Sub: "sub-unv", Email: "shoptest-unv@pyrkon.pl", EmailVerified: false, HD: "pyrkon.pl"}, "")
		assert.Equal(t, "email_not_verified", r.Body["code"])
	})

	t.Run("domain address without hd (consumer account)", func(t *testing.T) {
		r := e.login(oauth.GoogleUser{Sub: "sub-nohd", Email: "shoptest-nohd@pyrkon.pl", EmailVerified: true}, "")
		assert.Equal(t, "access_denied", r.Body["code"])
	})

	t.Run("expired and revoked invites", func(t *testing.T) {
		expired := e.do(http.MethodPost, "/admin/shop/invites", e.modToken, map[string]any{"label": "__TEST__expired"})
		require.Equal(t, http.StatusCreated, expired.Code)
		_, err := e.db.Exec(`UPDATE shop_invites SET expires_at = now() - interval '1 minute' WHERE id = $1`,
			num(expired.Body["invite"].(map[string]any)["id"]))
		require.NoError(t, err)
		r := e.login(oauth.GoogleUser{Sub: "sub-exp", Email: "shoptest-exp@gmail.com", EmailVerified: true}, expired.Body["token"].(string))
		assert.Equal(t, "invite_invalid", r.Body["code"])

		revoked := e.do(http.MethodPost, "/admin/shop/invites", e.modToken, map[string]any{"label": "__TEST__revoked"})
		require.Equal(t, http.StatusCreated, revoked.Code)
		id := num(revoked.Body["invite"].(map[string]any)["id"])
		require.Equal(t, http.StatusNoContent, e.do(http.MethodDelete, fmt.Sprintf("/admin/shop/invites/%d", id), e.modToken, nil).Code)
		assert.Equal(t, http.StatusConflict, e.do(http.MethodDelete, fmt.Sprintf("/admin/shop/invites/%d", id), e.modToken, nil).Code)
		r = e.login(oauth.GoogleUser{Sub: "sub-rev", Email: "shoptest-rev@gmail.com", EmailVerified: true}, revoked.Body["token"].(string))
		assert.Equal(t, "invite_invalid", r.Body["code"])
	})

	t.Run("workspace domain gets in", func(t *testing.T) {
		r := e.login(oauth.GoogleUser{Sub: "sub-dom", Email: "shoptest-dom@pyrkon.pl", EmailVerified: true, HD: "pyrkon.pl", Name: "Dom"}, "")
		require.Equal(t, http.StatusOK, r.Code, string(r.Raw))
		acc := r.Body["account"].(map[string]any)
		assert.Equal(t, "domain", acc["access_source"])
		assert.Equal(t, "Dom", acc["display_name"])
	})

	t.Run("consumer account without invite is refused", func(t *testing.T) {
		r := e.login(oauth.GoogleUser{Sub: "sub-gmail", Email: "shoptest-gmail@gmail.com", EmailVerified: true}, "")
		assert.Equal(t, http.StatusForbidden, r.Code)
		assert.Equal(t, "access_denied", r.Body["code"])
	})

	t.Run("invite admits a consumer account once", func(t *testing.T) {
		inv := e.do(http.MethodPost, "/admin/shop/invites", e.modToken, map[string]any{"label": "__TEST__Jan"})
		require.Equal(t, http.StatusCreated, inv.Code, string(inv.Raw))
		token := inv.Body["token"].(string)
		assert.Equal(t, testShopURL+"/invite/"+token, inv.Body["url"])

		r := e.login(oauth.GoogleUser{Sub: "sub-guest", Email: "shoptest-guest@gmail.com", EmailVerified: true}, token)
		require.Equal(t, http.StatusOK, r.Code, string(r.Raw))
		assert.Equal(t, "invite", r.Body["account"].(map[string]any)["access_source"])

		again := e.login(oauth.GoogleUser{Sub: "sub-guest2", Email: "shoptest-guest2@gmail.com", EmailVerified: true}, token)
		assert.Equal(t, http.StatusForbidden, again.Code)
		assert.Equal(t, "invite_invalid", again.Body["code"])

		list := e.do(http.MethodGet, "/admin/shop/invites", e.modToken, nil)
		require.Equal(t, http.StatusOK, list.Code)
		assert.Contains(t, string(list.Raw), `"status":"used"`)
		assert.NotContains(t, string(list.Raw), token, "plaintext token must not be listed")
	})

	t.Run("allowlisted e-mail binds on first login", func(t *testing.T) {
		add := e.do(http.MethodPost, "/admin/shop/accounts", e.modToken, map[string]any{"email": "ShopTest-Listed@Gmail.com"})
		require.Equal(t, http.StatusCreated, add.Code, string(add.Raw))
		assert.Equal(t, false, add.Body["logged_in"])

		r := e.login(oauth.GoogleUser{Sub: "sub-listed", Email: "shoptest-listed@gmail.com", EmailVerified: true}, "")
		require.Equal(t, http.StatusOK, r.Code, string(r.Raw))
		assert.Equal(t, true, r.Body["account"].(map[string]any)["logged_in"])
	})

	t.Run("domain auto-join off", func(t *testing.T) {
		e.setSettings(map[string]string{settingDomainAutoJoin: "false"})
		defer e.setSettings(map[string]string{settingDomainAutoJoin: "true"})
		r := e.login(oauth.GoogleUser{Sub: "sub-dom2", Email: "shoptest-dom2@pyrkon.pl", EmailVerified: true, HD: "pyrkon.pl"}, "")
		assert.Equal(t, http.StatusForbidden, r.Code)
	})
}

func TestShop_TokenSeparation(t *testing.T) {
	e := setupShop(t)
	shopToken := e.domainLogin("sep")

	assert.Equal(t, http.StatusOK, e.do(http.MethodGet, "/shop/me", shopToken, nil).Code)
	assert.Equal(t, http.StatusUnauthorized, e.do(http.MethodGet, "/shop/me", e.modToken, nil).Code, "warehouse token on /shop")
	assert.Equal(t, http.StatusUnauthorized, e.do(http.MethodGet, "/shop/me", "", nil).Code)
	assert.Equal(t, http.StatusUnauthorized, e.do(http.MethodGet, "/admin/shop/orders", shopToken, nil).Code, "shop token on the warehouse")

	userToken, err := security.GenerateJWT(strconv.Itoa(e.modID), "user", "__test__shop_mod")
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, e.do(http.MethodGet, "/admin/shop/orders", userToken, nil).Code, "plain user in the panel")
	assert.Equal(t, http.StatusForbidden, e.do(http.MethodGet, "/admin/shop/settings", e.modToken, nil).Code, "settings are admin-only")

	t.Run("blocking an account takes effect immediately", func(t *testing.T) {
		me := e.do(http.MethodGet, "/shop/me", shopToken, nil)
		id := num(me.Body["id"])
		block := e.do(http.MethodPatch, fmt.Sprintf("/admin/shop/accounts/%d", id), e.modToken, map[string]any{"active": false})
		require.Equal(t, http.StatusOK, block.Code, string(block.Raw))

		r := e.do(http.MethodGet, "/shop/me", shopToken, nil)
		assert.Equal(t, http.StatusForbidden, r.Code)
		assert.Equal(t, "account_inactive", r.Body["code"])

		relogin := e.login(oauth.GoogleUser{Sub: "sub-sep", Email: "shoptest-sep@pyrkon.pl", EmailVerified: true, HD: "pyrkon.pl"}, "")
		assert.Equal(t, http.StatusForbidden, relogin.Code)
	})
}

func TestShop_OrderLifecycle(t *testing.T) {
	e := setupShop(t)
	token := e.domainLogin("buyer")
	other := e.domainLogin("other")

	// Submit, with a retried request carrying the same Idempotency-Key.
	created := e.do(http.MethodPost, "/shop/orders", token, e.orderBody(), "Idempotency-Key", "k-1")
	require.Equal(t, http.StatusCreated, created.Code, string(created.Raw))
	orderID := num(created.Body["id"])
	assert.Regexp(t, `^SKL-\d{4,}$`, created.Body["number"])
	assert.Equal(t, "submitted", created.Body["status"])
	assert.Equal(t, 300.0, created.Body["total"]) // 2 × 150, the cable has no price

	replay := e.do(http.MethodPost, "/shop/orders", token, e.orderBody(), "Idempotency-Key", "k-1")
	assert.Equal(t, http.StatusOK, replay.Code)
	assert.Equal(t, orderID, num(replay.Body["id"]))
	var count int
	require.NoError(t, e.db.QueryRow(`SELECT count(*) FROM shop_orders o JOIN shop_accounts a ON a.id = o.account_id WHERE a.email = 'shoptest-buyer@pyrkon.pl'`).Scan(&count))
	assert.Equal(t, 1, count)

	path := fmt.Sprintf("/shop/orders/%d", orderID)
	assert.Equal(t, http.StatusNotFound, e.do(http.MethodGet, path, other, nil).Code, "someone else's order")
	assert.Equal(t, http.StatusNotFound, e.do(http.MethodPut, path, other, merge(e.orderBody(), "version", 1)).Code)
	assert.Equal(t, http.StatusNotFound, e.do(http.MethodPost, path+"/cancel", other, map[string]any{"version": 1}).Code)

	tooLong := e.do(http.MethodPost, "/shop/orders", token, merge(e.orderBody(), "contact_phone", "123456789012345678901234567890123456789012345678901"))
	assert.Equal(t, http.StatusBadRequest, tooLong.Code)

	// Validation
	over := e.do(http.MethodPost, "/shop/orders", token, e.orderBody(map[string]any{"product_id": e.products[0], "quantity": 6}))
	assert.Equal(t, "quantity_over_limit", over.Body["code"])
	badReturn := e.do(http.MethodPost, "/shop/orders", token, merge(e.orderBody(), "return_window_id", e.delivery))
	assert.Equal(t, "invalid_return_window", badReturn.Body["code"])

	// Edit with optimistic locking
	edited := e.do(http.MethodPut, path, token, merge(e.orderBody(map[string]any{"product_id": e.products[0], "quantity": 1}), "version", 1))
	require.Equal(t, http.StatusOK, edited.Code, string(edited.Raw))
	assert.Equal(t, 2, num(edited.Body["version"]))
	stale := e.do(http.MethodPut, path, token, merge(e.orderBody(), "version", 1))
	assert.Equal(t, http.StatusConflict, stale.Code)
	assert.Equal(t, "version_conflict", stale.Body["code"])

	// Prices hidden → the API returns none
	e.setSettings(map[string]string{settingShowPrices: "false"})
	hidden := e.do(http.MethodGet, path, token, nil)
	assert.Nil(t, hidden.Body["total"])
	assert.Nil(t, hidden.Body["items"].([]any)[0].(map[string]any)["unit_price"])
	catalog := e.do(http.MethodGet, "/shop/catalog", token, nil)
	assert.NotContains(t, string(catalog.Raw), `"price":150`)
	list := e.do(http.MethodGet, "/shop/orders", token, nil)
	assert.NotContains(t, string(list.Raw), `"unit_price":150`)
	assert.NotContains(t, string(list.Raw), `"total":`+"1")
	hiddenSubmit := e.do(http.MethodPost, "/shop/orders", token, e.orderBody())
	require.Equal(t, http.StatusCreated, hiddenSubmit.Code, string(hiddenSubmit.Raw))
	assert.Nil(t, hiddenSubmit.Body["total"])
	assert.NotContains(t, string(hiddenSubmit.Raw), `"unit_price":150`)
	e.setSettings(map[string]string{settingShowPrices: "true"})

	// Confirm → quest
	adminPath := fmt.Sprintf("/admin/shop/orders/%d", orderID)
	confirmed := e.do(http.MethodPost, adminPath+"/confirm", e.modToken, map[string]any{"version": 2})
	require.Equal(t, http.StatusOK, confirmed.Code, string(confirmed.Raw))
	assert.Equal(t, "confirmed", confirmed.Body["status"])
	questID := confirmed.Body["quest_id"].(string)
	assert.Equal(t, "pending", confirmed.Body["quest_status"])

	var source, pavilion, dest string
	var locID, shopOrderID, qty, catID int
	var returnDate sql.NullString
	require.NoError(t, e.db.QueryRow(`
		SELECT q.source, q.destination_pavilion, q.destination_location, q.location_id, q.shop_order_id,
		       to_char(q.return_date, 'YYYY-MM-DD'), i.quantity, i.category_id
		FROM equipment_request_quests q JOIN equipment_request_items i ON i.quest_id = q.id
		WHERE q.quest_id = $1`, questID).Scan(&source, &pavilion, &dest, &locID, &shopOrderID, &returnDate, &qty, &catID))
	assert.Equal(t, "shop", source)
	assert.Equal(t, "5", pavilion)
	assert.Equal(t, "__TEST__ShopHall", dest)
	assert.Equal(t, e.locationID, locID)
	assert.Equal(t, orderID, shopOrderID)
	assert.True(t, returnDate.Valid)
	assert.Equal(t, 1, qty)
	assert.Equal(t, e.categoryID, catID)

	again := e.do(http.MethodPost, adminPath+"/confirm", e.modToken, map[string]any{"version": 3})
	assert.Equal(t, http.StatusConflict, again.Code)
	assert.Equal(t, "already_confirmed", again.Body["code"])

	// Confirmed orders are locked for the organizer…
	locked := e.do(http.MethodPost, path+"/cancel", token, map[string]any{"version": 3})
	assert.Equal(t, http.StatusConflict, locked.Code)
	// …but the warehouse can still move them, and the quest follows.
	moved := e.do(http.MethodPatch, adminPath, e.modToken, map[string]any{"version": 3, "location_id": e.location2})
	require.Equal(t, http.StatusOK, moved.Code, string(moved.Raw))
	require.NoError(t, e.db.QueryRow(`SELECT location_id, destination_location FROM equipment_request_quests WHERE quest_id = $1`, questID).Scan(&locID, &dest))
	assert.Equal(t, e.location2, locID)
	assert.Equal(t, "__TEST__ShopStage", dest)
	itemsAfterConfirm := e.do(http.MethodPatch, adminPath, e.modToken, map[string]any{"version": 4, "items": []map[string]any{{"product_id": e.products[1], "quantity": 1}}})
	assert.Equal(t, http.StatusConflict, itemsAfterConfirm.Code)

	detail := e.do(http.MethodGet, adminPath, e.modToken, nil)
	require.Equal(t, http.StatusOK, detail.Code)
	types := []string{}
	for _, ev := range detail.Body["events"].([]any) {
		types = append(types, ev.(map[string]any)["type"].(string))
	}
	assert.Equal(t, []string{"submitted", "updated", "confirmed", "location_changed"}, types)

	summary := e.do(http.MethodGet, "/admin/shop/orders/summary?group=day", e.modToken, nil)
	require.Equal(t, http.StatusOK, summary.Code)
	assert.Contains(t, string(summary.Raw), "__TEST__Laptop")
}

func TestShop_RejectAndCancel(t *testing.T) {
	e := setupShop(t)
	token := e.domainLogin("rc")

	a := e.do(http.MethodPost, "/shop/orders", token, e.orderBody())
	require.Equal(t, http.StatusCreated, a.Code, string(a.Raw))
	aID := num(a.Body["id"])

	noReason := e.do(http.MethodPost, fmt.Sprintf("/admin/shop/orders/%d/reject", aID), e.modToken, map[string]any{"version": 1, "reason": " "})
	assert.Equal(t, http.StatusBadRequest, noReason.Code)
	rejected := e.do(http.MethodPost, fmt.Sprintf("/admin/shop/orders/%d/reject", aID), e.modToken, map[string]any{"version": 1, "reason": "Brak sprzętu"})
	require.Equal(t, http.StatusOK, rejected.Code, string(rejected.Raw))
	mine := e.do(http.MethodGet, fmt.Sprintf("/shop/orders/%d", aID), token, nil)
	assert.Equal(t, "rejected", mine.Body["status"])
	assert.Equal(t, "Brak sprzętu", mine.Body["status_reason"])

	b := e.do(http.MethodPost, "/shop/orders", token, e.orderBody())
	bID := num(b.Body["id"])
	cancelled := e.do(http.MethodPost, fmt.Sprintf("/shop/orders/%d/cancel", bID), token, map[string]any{"version": 1})
	require.Equal(t, http.StatusOK, cancelled.Code, string(cancelled.Raw))
	assert.Equal(t, "cancelled", cancelled.Body["status"])
	confirmCancelled := e.do(http.MethodPost, fmt.Sprintf("/admin/shop/orders/%d/confirm", bID), e.modToken, map[string]any{"version": 2})
	assert.Equal(t, http.StatusConflict, confirmCancelled.Code)

	e.setSettings(map[string]string{settingOrdersOpen: "false"})
	closed := e.do(http.MethodPost, "/shop/orders", token, e.orderBody())
	assert.Equal(t, http.StatusForbidden, closed.Code)
	assert.Equal(t, "orders_closed", closed.Body["code"])
}

func merge(m map[string]any, k string, v any) map[string]any {
	out := map[string]any{}
	for kk, vv := range m {
		out[kk] = vv
	}
	out[k] = v
	return out
}
