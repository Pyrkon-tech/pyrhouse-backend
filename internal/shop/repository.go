package shop

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"warehouse/internal/shop/access"
)

// querier is satisfied by *sql.DB and *sql.Tx, so reads work inside and outside a transaction.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ============================================================================
// Settings
// ============================================================================

const (
	settingShowPrices      = "shop.show_prices"
	settingOrdersOpen      = "shop.orders_open"
	settingOrdersOpenUntil = "shop.orders_open_until"
	settingDomainAutoJoin  = "shop.auth.domain_auto_join"
	settingAutoDomains     = "shop.auth.auto_domains"
)

func (r *Repository) loadSettings(ctx context.Context, q querier) (Settings, error) {
	rows, err := q.QueryContext(ctx, `SELECT key, value FROM app_settings WHERE key LIKE 'shop.%'`)
	if err != nil {
		return Settings{}, err
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return Settings{}, err
		}
		values[k] = v
	}
	if err := rows.Err(); err != nil {
		return Settings{}, err
	}

	// Missing or malformed values fall back to the safe side: prices hidden, orders closed,
	// no automatic domain access.
	s := Settings{
		ShowPrices:     values[settingShowPrices] == "true",
		OrdersOpen:     values[settingOrdersOpen] == "true",
		DomainAutoJoin: values[settingDomainAutoJoin] == "true",
		AutoDomains:    access.ParseDomains(values[settingAutoDomains]),
	}
	if raw := strings.TrimSpace(values[settingOrdersOpenUntil]); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			s.OrdersOpenUntil = &t
		} else {
			s.OrdersOpen = false
		}
	}
	if s.AutoDomains == nil {
		s.AutoDomains = []string{}
	}
	return s, nil
}

func (r *Repository) saveSettings(ctx context.Context, s Settings) error {
	until := ""
	if s.OrdersOpenUntil != nil {
		until = s.OrdersOpenUntil.UTC().Format(time.RFC3339)
	}
	values := map[string]string{
		settingShowPrices:      strconv.FormatBool(s.ShowPrices),
		settingOrdersOpen:      strconv.FormatBool(s.OrdersOpen),
		settingOrdersOpenUntil: until,
		settingDomainAutoJoin:  strconv.FormatBool(s.DomainAutoJoin),
		settingAutoDomains:     strings.Join(s.AutoDomains, ","),
	}
	return r.inTx(ctx, func(tx *sql.Tx) error {
		for k, v := range values {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO app_settings (key, value, updated_at) VALUES ($1, $2, now())
				ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// ============================================================================
// Accounts
// ============================================================================

const accountColumns = `id, email, display_name, avatar_url, active, access_source, google_sub IS NOT NULL, created_at, last_login_at`

func scanAccount(row interface{ Scan(...any) error }) (*Account, error) {
	var a Account
	if err := row.Scan(&a.ID, &a.Email, &a.DisplayName, &a.AvatarURL, &a.Active, &a.AccessSource, &a.LoggedIn, &a.CreatedAt, &a.LastLoginAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) accountByID(ctx context.Context, id int) (*Account, error) {
	a, err := scanAccount(r.db.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM shop_accounts WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *Repository) listAccounts(ctx context.Context) ([]Account, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+accountColumns+` FROM shop_accounts ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *Repository) createAllowlistAccount(ctx context.Context, email string, createdBy int) (*Account, error) {
	a, err := scanAccount(r.db.QueryRowContext(ctx, `
		INSERT INTO shop_accounts (email, access_source, created_by) VALUES ($1, 'allowlist', $2)
		RETURNING `+accountColumns, email, createdBy))
	if isUniqueViolation(err) {
		return nil, conflict("account_exists", "Konto z tym adresem e-mail już istnieje")
	}
	return a, err
}

func (r *Repository) setAccountActive(ctx context.Context, id int, active bool) (*Account, error) {
	a, err := scanAccount(r.db.QueryRowContext(ctx,
		`UPDATE shop_accounts SET active = $2 WHERE id = $1 RETURNING `+accountColumns, id, active))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("Konto nie istnieje")
	}
	return a, err
}

func (r *Repository) touchLogin(ctx context.Context, q querier, id int, name, avatar string) error {
	_, err := q.ExecContext(ctx, `
		UPDATE shop_accounts
		SET last_login_at = now(), display_name = NULLIF($2, ''), avatar_url = NULLIF($3, '')
		WHERE id = $1`, id, name, avatar)
	return err
}

// accessStore adapts a transaction to access.Store.
type accessStore struct {
	ctx context.Context
	tx  *sql.Tx
}

func (s accessStore) find(where string, arg any) (*access.Account, error) {
	var a access.Account
	var source string
	err := s.tx.QueryRowContext(s.ctx,
		`SELECT id, email, google_sub, active, access_source FROM shop_accounts WHERE `+where+` FOR UPDATE`, arg).
		Scan(&a.ID, &a.Email, &a.GoogleSub, &a.Active, &source)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	a.AccessSource = access.Source(source)
	return &a, err
}

func (s accessStore) FindBySub(sub string) (*access.Account, error) {
	return s.find("google_sub = $1", sub)
}

func (s accessStore) FindByEmail(email string) (*access.Account, error) {
	return s.find("email = $1", email)
}

func (s accessStore) BindGoogleSub(accountID int, sub string) error {
	_, err := s.tx.ExecContext(s.ctx, `UPDATE shop_accounts SET google_sub = $2 WHERE id = $1 AND google_sub IS NULL`, accountID, sub)
	return err
}

func (s accessStore) ClaimInvite(tokenHash string) (int, bool, error) {
	var id int
	err := s.tx.QueryRowContext(s.ctx, `
		UPDATE shop_invites SET used_at = now()
		WHERE token_hash = $1 AND used_at IS NULL AND revoked_at IS NULL AND expires_at > now()
		RETURNING id`, tokenHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}

func (s accessStore) MarkInviteUsedBy(inviteID, accountID int) error {
	_, err := s.tx.ExecContext(s.ctx, `UPDATE shop_invites SET used_by_account_id = $2 WHERE id = $1`, inviteID, accountID)
	return err
}

func (s accessStore) CreateAccount(email, sub string, source access.Source) (*access.Account, error) {
	a := access.Account{Email: email, GoogleSub: &sub, Active: true, AccessSource: source}
	err := s.tx.QueryRowContext(s.ctx, `
		INSERT INTO shop_accounts (email, google_sub, access_source) VALUES ($1, $2, $3) RETURNING id`,
		email, sub, string(source)).Scan(&a.ID)
	if isUniqueViolation(err) {
		// A parallel first login (double click, retried callback) created the account first.
		return nil, errLoginRace
	}
	return &a, err
}

var errLoginRace = errors.New("concurrent first login")

// ============================================================================
// Invites
// ============================================================================

func (r *Repository) createInvite(ctx context.Context, tokenHash, label string, createdBy int, expiresAt time.Time) (*Invite, error) {
	var inv Invite
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO shop_invites (token_hash, label, created_by, expires_at) VALUES ($1, $2, $3, $4)
		RETURNING id, label, created_by, created_at, expires_at`, tokenHash, label, createdBy, expiresAt).
		Scan(&inv.ID, &inv.Label, &inv.CreatedBy, &inv.CreatedAt, &inv.ExpiresAt)
	inv.Status = "active"
	return &inv, err
}

func (r *Repository) listInvites(ctx context.Context) ([]Invite, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.label, i.created_by, i.created_at, i.expires_at, i.used_at, a.email, i.revoked_at,
		       CASE WHEN i.used_at IS NOT NULL THEN 'used'
		            WHEN i.revoked_at IS NOT NULL THEN 'revoked'
		            WHEN i.expires_at <= now() THEN 'expired'
		            ELSE 'active' END
		FROM shop_invites i
		LEFT JOIN shop_accounts a ON a.id = i.used_by_account_id
		ORDER BY i.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invite{}
	for rows.Next() {
		var inv Invite
		if err := rows.Scan(&inv.ID, &inv.Label, &inv.CreatedBy, &inv.CreatedAt, &inv.ExpiresAt, &inv.UsedAt, &inv.UsedBy, &inv.RevokedAt, &inv.Status); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *Repository) revokeInvite(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE shop_invites SET revoked_at = now() WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists bool
		_ = r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM shop_invites WHERE id = $1)`, id).Scan(&exists)
		if !exists {
			return notFound("Zaproszenie nie istnieje")
		}
		return conflict("invite_closed", "Zaproszenie zostało już wykorzystane albo unieważnione")
	}
	return nil
}

// ============================================================================
// Catalog: products, windows, locations
// ============================================================================

const productColumns = `p.id, p.name, p.description, p.image_url, p.section, p.sort_order, p.category_id, c.label,
	p.price::float8, p.max_per_order, p.active, p.created_at, p.updated_at`

func scanProduct(row interface{ Scan(...any) error }) (*Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.Section, &p.SortOrder, &p.CategoryID, &p.CategoryName,
		&p.Price, &p.MaxPerOrder, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

func (r *Repository) listProducts(ctx context.Context, activeOnly bool) ([]Product, error) {
	where := ""
	if activeOnly {
		where = "WHERE p.active"
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+productColumns+`
		FROM shop_products p LEFT JOIN item_category c ON c.id = p.category_id
		`+where+`
		ORDER BY p.section, p.sort_order, p.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) productsByID(ctx context.Context, q querier, ids []int) (map[int]Product, error) {
	out := map[int]Product{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx, `
		SELECT `+productColumns+`
		FROM shop_products p LEFT JOIN item_category c ON c.id = p.category_id
		WHERE p.id = ANY($1)`, intArray(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out[p.ID] = *p
	}
	return out, rows.Err()
}

type productInput struct {
	Name        string   `json:"name" binding:"required,max=255"`
	Description *string  `json:"description" binding:"omitempty,max=4000"`
	ImageURL    *string  `json:"image_url" binding:"omitempty,max=2048"`
	Section     string   `json:"section" binding:"max=100"`
	SortOrder   int      `json:"sort_order"`
	CategoryID  int      `json:"category_id" binding:"required"`
	Price       *float64 `json:"price"`
	MaxPerOrder *int     `json:"max_per_order"`
	Active      *bool    `json:"active"`
}

func (r *Repository) saveProduct(ctx context.Context, id int, in productInput, userID int) (*Product, error) {
	active := in.Active == nil || *in.Active
	var newID int
	var err error
	if id == 0 {
		err = r.db.QueryRowContext(ctx, `
			INSERT INTO shop_products (name, description, image_url, section, sort_order, category_id, price, max_per_order, active, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
			in.Name, in.Description, in.ImageURL, in.Section, in.SortOrder, in.CategoryID, in.Price, in.MaxPerOrder, active, userID).
			Scan(&newID)
	} else {
		err = r.db.QueryRowContext(ctx, `
			UPDATE shop_products SET name = $2, description = $3, image_url = $4, section = $5, sort_order = $6,
			       category_id = $7, price = $8, max_per_order = $9, active = $10, updated_at = now()
			WHERE id = $1 RETURNING id`,
			id, in.Name, in.Description, in.ImageURL, in.Section, in.SortOrder, in.CategoryID, in.Price, in.MaxPerOrder, active).
			Scan(&newID)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("Produkt nie istnieje")
	}
	if isForeignKeyViolation(err) {
		return nil, badRequest("category_not_found", "Kategoria nie istnieje")
	}
	if err != nil {
		return nil, err
	}
	p, err := scanProduct(r.db.QueryRowContext(ctx, `
		SELECT `+productColumns+` FROM shop_products p LEFT JOIN item_category c ON c.id = p.category_id WHERE p.id = $1`, newID))
	return p, err
}

const windowColumns = `id, kind, starts_at, ends_at, label, active`

func scanWindow(row interface{ Scan(...any) error }) (*Window, error) {
	var w Window
	err := row.Scan(&w.ID, &w.Kind, &w.StartsAt, &w.EndsAt, &w.Label, &w.Active)
	return &w, err
}

// listWindows returns windows ordered by start. upcomingOnly keeps active windows that have not ended.
func (r *Repository) listWindows(ctx context.Context, upcomingOnly bool) ([]Window, error) {
	where := ""
	if upcomingOnly {
		where = "WHERE active AND ends_at > now()"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+windowColumns+` FROM shop_delivery_windows `+where+` ORDER BY starts_at, kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Window{}
	for rows.Next() {
		w, err := scanWindow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func (r *Repository) windowByID(ctx context.Context, q querier, id int) (*Window, error) {
	w, err := scanWindow(q.QueryRowContext(ctx, `SELECT `+windowColumns+` FROM shop_delivery_windows WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return w, err
}

type windowInput struct {
	Kind     string    `json:"kind" binding:"required,oneof=delivery return"`
	StartsAt time.Time `json:"starts_at" binding:"required"`
	EndsAt   time.Time `json:"ends_at" binding:"required"`
	Label    string    `json:"label" binding:"max=255"`
	Active   *bool     `json:"active"`
}

func (r *Repository) saveWindow(ctx context.Context, id int, in windowInput) (*Window, error) {
	if !in.EndsAt.After(in.StartsAt) {
		return nil, badRequest("invalid_window", "Koniec okna musi być po jego początku")
	}
	active := in.Active == nil || *in.Active
	var w *Window
	var err error
	if id == 0 {
		w, err = scanWindow(r.db.QueryRowContext(ctx, `
			INSERT INTO shop_delivery_windows (kind, starts_at, ends_at, label, active) VALUES ($1, $2, $3, $4, $5)
			RETURNING `+windowColumns, in.Kind, in.StartsAt, in.EndsAt, in.Label, active))
	} else {
		w, err = scanWindow(r.db.QueryRowContext(ctx, `
			UPDATE shop_delivery_windows SET kind = $2, starts_at = $3, ends_at = $4, label = $5, active = $6
			WHERE id = $1 RETURNING `+windowColumns, id, in.Kind, in.StartsAt, in.EndsAt, in.Label, active))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("Okno nie istnieje")
	}
	return w, err
}

func (r *Repository) deleteWindow(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM shop_delivery_windows WHERE id = $1`, id)
	if isForeignKeyViolation(err) {
		return conflict("window_in_use", "Okno jest użyte w zamówieniach — zamiast usuwać, wyłącz je")
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return notFound("Okno nie istnieje")
	}
	return nil
}

func (r *Repository) listLocations(ctx context.Context) ([]Location, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(name, ''), pavilion FROM locations ORDER BY pavilion NULLS LAST, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Location{}
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Pavilion); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) locationByID(ctx context.Context, q querier, id int) (*Location, error) {
	var l Location
	err := q.QueryRowContext(ctx, `SELECT id, COALESCE(name, ''), pavilion FROM locations WHERE id = $1`, id).
		Scan(&l.ID, &l.Name, &l.Pavilion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &l, err
}

// ============================================================================
// Orders
// ============================================================================

const orderSelect = `
	SELECT o.id, o.number, o.account_id, a.email,
	       l.id, COALESCE(l.name, ''), l.pavilion, o.location_note,
	       o.contact_name, o.contact_phone, o.budget_owner,
	       dw.id, dw.kind, dw.starts_at, dw.ends_at, dw.label, dw.active,
	       rw.id, rw.kind, rw.starts_at, rw.ends_at, rw.label, rw.active,
	       to_char(o.return_date, 'YYYY-MM-DD'), o.notes, o.status, o.status_reason, o.decided_by, o.decided_at,
	       o.version, o.created_at, o.updated_at, q.quest_id, q.status
	FROM shop_orders o
	JOIN shop_accounts a ON a.id = o.account_id
	JOIN locations l ON l.id = o.location_id
	JOIN shop_delivery_windows dw ON dw.id = o.delivery_window_id
	JOIN shop_delivery_windows rw ON rw.id = o.return_window_id
	LEFT JOIN equipment_request_quests q ON q.shop_order_id = o.id`

func scanOrder(row interface{ Scan(...any) error }) (*Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.Number, &o.AccountID, &o.AccountEmail,
		&o.Location.ID, &o.Location.Name, &o.Location.Pavilion, &o.LocationNote,
		&o.ContactName, &o.ContactPhone, &o.BudgetOwner,
		&o.Delivery.ID, &o.Delivery.Kind, &o.Delivery.StartsAt, &o.Delivery.EndsAt, &o.Delivery.Label, &o.Delivery.Active,
		&o.Return.ID, &o.Return.Kind, &o.Return.StartsAt, &o.Return.EndsAt, &o.Return.Label, &o.Return.Active,
		&o.ReturnDate, &o.Notes, &o.Status, &o.StatusReason, &o.decidedBy, &o.DecidedAt,
		&o.Version, &o.CreatedAt, &o.UpdatedAt, &o.QuestID, &o.QuestStatus)
	return &o, err
}

// orderByID loads an order with its items. With lock=true the order row is locked for update
// (q must then be a transaction).
func (r *Repository) orderByID(ctx context.Context, q querier, id int, lock bool) (*Order, error) {
	if lock {
		// FOR UPDATE cannot lock the nullable side of the quest outer join; lock the order row alone first.
		var dummy int
		err := q.QueryRowContext(ctx, `SELECT id FROM shop_orders WHERE id = $1 FOR UPDATE`, id).Scan(&dummy)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	o, err := scanOrder(q.QueryRowContext(ctx, orderSelect+` WHERE o.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := r.orderItems(ctx, q, []int{o.ID})
	if err != nil {
		return nil, err
	}
	o.Items = items[o.ID]
	if o.Items == nil {
		o.Items = []OrderItem{}
	}
	return o, nil
}

type orderFilter struct {
	AccountID *int
	Status    string
}

func (r *Repository) listOrders(ctx context.Context, f orderFilter) ([]Order, error) {
	var conds []string
	var args []any
	if f.AccountID != nil {
		args = append(args, *f.AccountID)
		conds = append(conds, fmt.Sprintf("o.account_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("o.status = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	rows, err := r.db.QueryContext(ctx, orderSelect+where+` ORDER BY o.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []Order{}
	var ids []int
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
		ids = append(ids, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items, err := r.orderItems(ctx, r.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].Items = items[orders[i].ID]
		if orders[i].Items == nil {
			orders[i].Items = []OrderItem{}
		}
	}
	return orders, nil
}

func (r *Repository) orderItems(ctx context.Context, q querier, orderIDs []int) (map[int][]OrderItem, error) {
	out := map[int][]OrderItem{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := q.QueryContext(ctx, `
		SELECT order_id, product_id, product_name, category_id, quantity, unit_price::float8
		FROM shop_order_items WHERE order_id = ANY($1) ORDER BY id`, intArray(orderIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int
		var it OrderItem
		if err := rows.Scan(&orderID, &it.ProductID, &it.ProductName, &it.CategoryID, &it.Quantity, &it.UnitPrice); err != nil {
			return nil, err
		}
		out[orderID] = append(out[orderID], it)
	}
	return out, rows.Err()
}

func (r *Repository) orderIDByIdempotencyKey(ctx context.Context, q querier, accountID int, key string) (int, error) {
	var id int
	err := q.QueryRowContext(ctx, `SELECT id FROM shop_orders WHERE account_id = $1 AND idempotency_key = $2`, accountID, key).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// orderFields are the organizer-editable parts of an order.
type orderFields struct {
	LocationID       int
	LocationNote     *string
	ContactName      string
	ContactPhone     *string
	BudgetOwner      *string
	DeliveryWindowID int
	ReturnWindowID   int
	ReturnDate       *string
	Notes            *string
}

func (r *Repository) insertOrder(ctx context.Context, tx *sql.Tx, accountID int, f orderFields, idempotencyKey *string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO shop_orders (account_id, location_id, location_note, contact_name, contact_phone, budget_owner,
		                         delivery_window_id, return_window_id, return_date, notes, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		accountID, f.LocationID, f.LocationNote, f.ContactName, f.ContactPhone, f.BudgetOwner,
		f.DeliveryWindowID, f.ReturnWindowID, f.ReturnDate, f.Notes, idempotencyKey).Scan(&id)
	return id, err
}

func (r *Repository) updateOrderFields(ctx context.Context, tx *sql.Tx, id int, f orderFields) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE shop_orders SET location_id = $2, location_note = $3, contact_name = $4, contact_phone = $5,
		       budget_owner = $6, delivery_window_id = $7, return_window_id = $8, return_date = $9, notes = $10,
		       version = version + 1, updated_at = now()
		WHERE id = $1`,
		id, f.LocationID, f.LocationNote, f.ContactName, f.ContactPhone, f.BudgetOwner,
		f.DeliveryWindowID, f.ReturnWindowID, f.ReturnDate, f.Notes)
	return err
}

func (r *Repository) replaceItems(ctx context.Context, tx *sql.Tx, orderID int, items []OrderItem) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM shop_order_items WHERE order_id = $1`, orderID); err != nil {
		return err
	}
	for _, it := range items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO shop_order_items (order_id, product_id, quantity, product_name, category_id, unit_price)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			orderID, it.ProductID, it.Quantity, it.ProductName, it.CategoryID, it.UnitPrice); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) setStatus(ctx context.Context, tx *sql.Tx, id int, status string, reason *string, decidedBy *int) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE shop_orders SET status = $2, status_reason = $3, decided_by = $4,
		       decided_at = CASE WHEN $4::int IS NULL THEN decided_at ELSE now() END,
		       version = version + 1, updated_at = now()
		WHERE id = $1`, id, status, reason, decidedBy)
	return err
}

// setLocation does not bump the version; the caller bumps it once per change set.
func (r *Repository) setLocation(ctx context.Context, tx *sql.Tx, id, locationID int, note *string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE shop_orders SET location_id = $2, location_note = $3, updated_at = now()
		WHERE id = $1`, id, locationID, note)
	return err
}

func (r *Repository) bumpVersion(ctx context.Context, tx *sql.Tx, id int) error {
	_, err := tx.ExecContext(ctx, `UPDATE shop_orders SET version = version + 1, updated_at = now() WHERE id = $1`, id)
	return err
}

func (r *Repository) insertEvent(ctx context.Context, tx *sql.Tx, orderID int, actorKind string, actorID int, typ string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO shop_order_events (order_id, actor_kind, actor_id, type, payload) VALUES ($1, $2, $3, $4, $5)`,
		orderID, actorKind, actorID, typ, raw)
	return err
}

func (r *Repository) listEvents(ctx context.Context, orderID int) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT actor_kind, actor_id, type, payload, at FROM shop_order_events WHERE order_id = $1 ORDER BY at, id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var raw []byte
		if err := rows.Scan(&e.ActorKind, &e.ActorID, &e.Type, &raw, &e.At); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &e.Payload)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ============================================================================
// Quest created by confirming an order
// ============================================================================

// questDestination mirrors what quests store for a location.
func questDestination(l Location) (pavilion, name string) {
	if l.Pavilion != nil {
		pavilion = *l.Pavilion
	}
	return pavilion, l.Name
}

func (r *Repository) createQuest(ctx context.Context, tx *sql.Tx, o *Order) (string, error) {
	questID := fmt.Sprintf("quest-shop-%d", o.ID)
	pavilion, name := questDestination(o.Location)

	start := o.Delivery.StartsAt.In(warsaw)
	pickup := start.Format("15:04") + "–" + o.Delivery.EndsAt.In(warsaw).Format("15:04")
	returnDate := o.Return.StartsAt.In(warsaw).Format("2006-01-02")
	if o.ReturnDate != nil {
		returnDate = *o.ReturnDate
	}

	var questDBID int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO equipment_request_quests
			(quest_key, quest_id, destination_pavilion, destination_location, recipient, delivery_date, pickup_time,
			 budget_owner, status, location_id, location_resolved, source, return_date, shop_order_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', $9, true, 'shop', $10, $11)
		RETURNING id`,
		fmt.Sprintf("shop:%d", o.ID), questID, pavilion, name, o.ContactName, start.Format("2006-01-02"), pickup,
		o.BudgetOwner, o.Location.ID, returnDate, o.ID).Scan(&questDBID)
	if err != nil {
		return "", fmt.Errorf("create quest: %w", err)
	}

	for _, it := range o.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO equipment_request_items (quest_id, item_name, quantity, category_id, budget_owner)
			VALUES ($1, $2, $3, $4, $5)`, questDBID, it.ProductName, it.Quantity, it.CategoryID, o.BudgetOwner); err != nil {
			return "", fmt.Errorf("create quest item: %w", err)
		}
	}
	return questID, nil
}

func (r *Repository) moveQuest(ctx context.Context, tx *sql.Tx, orderID int, l Location) error {
	pavilion, name := questDestination(l)
	_, err := tx.ExecContext(ctx, `
		UPDATE equipment_request_quests
		SET location_id = $2, location_resolved = true, destination_pavilion = $3, destination_location = $4
		WHERE shop_order_id = $1`, orderID, l.ID, pavilion, name)
	return err
}

// ============================================================================
// Summary
// ============================================================================

type SummaryRow struct {
	Key         string `json:"key"`   // group key: product ID, YYYY-MM-DD or location ID
	Label       string `json:"label"` // human-readable group
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Orders      int    `json:"orders"`
}

// summary aggregates submitted and confirmed orders (what the warehouse has to prepare).
func (r *Repository) summary(ctx context.Context, group string) ([]SummaryRow, error) {
	var keyExpr, labelExpr string
	switch group {
	case "day":
		keyExpr = `to_char(dw.starts_at AT TIME ZONE 'Europe/Warsaw', 'YYYY-MM-DD')`
		labelExpr = keyExpr
	case "location":
		keyExpr = `l.id::text`
		labelExpr = `COALESCE(l.pavilion || ' — ', '') || COALESCE(l.name, '')`
	default:
		keyExpr = `i.product_id::text`
		labelExpr = `i.product_name`
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+keyExpr+` AS k, `+labelExpr+` AS lbl, i.product_id, i.product_name,
		       SUM(i.quantity)::int, COUNT(DISTINCT o.id)::int
		FROM shop_order_items i
		JOIN shop_orders o ON o.id = i.order_id
		JOIN shop_delivery_windows dw ON dw.id = o.delivery_window_id
		JOIN locations l ON l.id = o.location_id
		WHERE o.status IN ('submitted', 'confirmed')
		GROUP BY 1, 2, 3, 4
		ORDER BY 1, 4`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SummaryRow{}
	for rows.Next() {
		var s SummaryRow
		if err := rows.Scan(&s.Key, &s.Label, &s.ProductID, &s.ProductName, &s.Quantity, &s.Orders); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ============================================================================
// Helpers
// ============================================================================

// intArray renders a Postgres int[] literal; database/sql has no native array support.
func intArray(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func pqCode(err error) string {
	if err == nil {
		return ""
	}
	var coded interface{ SQLState() string }
	if errors.As(err, &coded) {
		return coded.SQLState()
	}
	return ""
}

func isUniqueViolation(err error) bool     { return pqCode(err) == "23505" }
func isForeignKeyViolation(err error) bool { return pqCode(err) == "23503" }
