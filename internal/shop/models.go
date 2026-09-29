// Package shop is the organizer shop backend (docs/shop/PLAN.md in the workspace repo):
// organizer endpoints under /shop (ShopAuth, shop tokens) and the warehouse panel under
// /admin/shop (JWTMiddleware + roles). A confirmed order becomes an equipment request quest.
package shop

import (
	"net/http"
	"time"
	_ "time/tzdata" // Europe/Warsaw must resolve even where the OS has no zoneinfo
)

// Order statuses (shop_orders.status).
const (
	StatusSubmitted = "submitted"
	StatusConfirmed = "confirmed"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
)

// Delivery window kinds (shop_delivery_windows.kind).
const (
	WindowDelivery = "delivery"
	WindowReturn   = "return"
)

// Event actor kinds (shop_order_events.actor_kind).
const (
	ActorAccount = "account"
	ActorUser    = "user"
)

// warsaw is the event's time zone: window days and return dates are Polish calendar days.
var warsaw = mustLoadLocation("Europe/Warsaw")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type Account struct {
	ID           int        `json:"id"`
	Email        string     `json:"email"`
	DisplayName  *string    `json:"display_name"`
	AvatarURL    *string    `json:"avatar_url"`
	Active       bool       `json:"active"`
	AccessSource string     `json:"access_source"`
	LoggedIn     bool       `json:"logged_in"` // false = allowlist entry that has not logged in yet
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	OrdersCount  int        `json:"orders_count"`
}

type Invite struct {
	ID        int        `json:"id"`
	Label     string     `json:"label"`
	CreatedBy *int       `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	UsedBy    *string    `json:"used_by_email"`
	RevokedAt *time.Time `json:"revoked_at"`
	Status    string     `json:"status"` // active | used | revoked | expired
}

type Product struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	ImageURL     *string   `json:"image_url"`
	Section      string    `json:"section"`
	SortOrder    int       `json:"sort_order"`
	CategoryID   int       `json:"category_id"`
	CategoryName *string   `json:"category_name"`
	Price        *float64  `json:"price"` // nil = no price, or prices hidden from organizers
	MaxPerOrder  *int      `json:"max_per_order"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Window struct {
	ID       int       `json:"id"`
	Kind     string    `json:"kind"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Label    string    `json:"label"`
	Active   bool      `json:"active"`
	// OrdersCount counts submitted and confirmed orders using the window (panel only; 0 for organizers).
	OrdersCount int `json:"orders_count"`
}

type Location struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Pavilion *string `json:"pavilion"`
}

type OrderItem struct {
	ProductID    int      `json:"product_id"`
	ProductName  string   `json:"product_name"`
	CategoryID   int      `json:"category_id"`
	CategoryName *string  `json:"category_name"` // panel only; null for organizers
	Quantity     int      `json:"quantity"`
	UnitPrice    *float64 `json:"unit_price"` // snapshot; nil when the product had no price or prices are hidden
}

type Order struct {
	ID            int         `json:"id"`
	Number        string      `json:"number"`
	AccountID     int         `json:"account_id"`
	AccountEmail  string      `json:"account_email"`
	AccountName   *string     `json:"account_name"`
	Location      Location    `json:"location"`
	LocationNote  *string     `json:"location_note"`
	ContactName   string      `json:"contact_name"`
	BudgetOwner   *string     `json:"budget_owner"`
	Delivery      Window      `json:"delivery_window"`
	Return        Window      `json:"return_window"`
	ReturnDate    *string     `json:"return_date"` // YYYY-MM-DD
	Notes         *string     `json:"notes"`
	Status        string      `json:"status"`
	StatusReason  *string     `json:"status_reason"`
	DecidedAt     *time.Time  `json:"decided_at"`
	Version       int         `json:"version"`
	Items         []OrderItem `json:"items"`
	Total         *float64    `json:"total"` // sum of priced items; nil when prices are hidden
	QuestID       *string     `json:"quest_id"`
	QuestStatus   *string     `json:"quest_status"` // fulfillment after confirmation, derived from the quest
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	decidedBy     *int
	idempotentHit bool
}

type Event struct {
	ActorKind string         `json:"actor_kind"`
	ActorID   int            `json:"actor_id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	At        time.Time      `json:"at"`
}

// Settings are the shop.* app_settings.
type Settings struct {
	ShowPrices      bool       `json:"show_prices"`
	OrdersOpen      bool       `json:"orders_open"`
	OrdersOpenUntil *time.Time `json:"orders_open_until"`
	DomainAutoJoin  bool       `json:"domain_auto_join"`
	AutoDomains     []string   `json:"auto_domains"`
}

// OrdersAccepted reports whether organizers may submit or edit orders right now.
func (s Settings) OrdersAccepted(now time.Time) bool {
	return s.OrdersOpen && (s.OrdersOpenUntil == nil || now.Before(*s.OrdersOpenUntil))
}

// Error is a failure with an HTTP status and a stable code the frontends can switch on.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func newErr(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

func badRequest(code, msg string) *Error { return newErr(http.StatusBadRequest, code, msg) }
func notFound(msg string) *Error         { return newErr(http.StatusNotFound, "not_found", msg) }
func conflict(code, msg string) *Error   { return newErr(http.StatusConflict, code, msg) }
