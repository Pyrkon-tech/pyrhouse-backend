package shop

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// State machine — the single place that says what may happen to an order.
// ============================================================================

type action string

const (
	actEdit           action = "edit"            // organizer edits fields and items
	actCancel         action = "cancel"          // organizer withdraws the order
	actConfirm        action = "confirm"         // moderator accepts; a quest is created
	actReject         action = "reject"          // moderator refuses
	actChangeItems    action = "change_items"    // moderator corrects items
	actChangeLocation action = "change_location" // moderator corrects the location (propagates to the quest)
)

var transitions = map[string]map[action]bool{
	StatusSubmitted: {actEdit: true, actCancel: true, actConfirm: true, actReject: true, actChangeItems: true, actChangeLocation: true},
	// A confirmed order is locked (a change means a new order), except that the warehouse may
	// still move it to the right location.
	StatusConfirmed: {actChangeLocation: true},
	StatusRejected:  {},
	StatusCancelled: {},
}

func canDo(status string, a action) bool {
	return transitions[status][a]
}

func checkTransition(o *Order, a action, version int) error {
	if !canDo(o.Status, a) {
		if a == actConfirm && o.Status == StatusConfirmed {
			return conflict("already_confirmed", "Zamówienie jest już potwierdzone")
		}
		return conflict("invalid_status", fmt.Sprintf("Tej operacji nie można wykonać na zamówieniu w statusie %q", o.Status))
	}
	if version != o.Version {
		return conflict("version_conflict", "Zamówienie zostało w międzyczasie zmienione — odśwież i spróbuj ponownie")
	}
	return nil
}

// ============================================================================
// Inputs and validation
// ============================================================================

type itemInput struct {
	ProductID int `json:"product_id" binding:"required"`
	Quantity  int `json:"quantity" binding:"required,gt=0"`
}

type orderInput struct {
	LocationID       int         `json:"location_id" binding:"required"`
	LocationNote     *string     `json:"location_note" binding:"omitempty,max=1000"`
	ContactName      string      `json:"contact_name" binding:"max=255"` // empty = the account's Google name
	BudgetOwner      *string     `json:"budget_owner" binding:"omitempty,max=255"`
	DeliveryWindowID int         `json:"delivery_window_id" binding:"required"`
	ReturnWindowID   int         `json:"return_window_id" binding:"required"`
	ReturnDate       *string     `json:"return_date"` // YYYY-MM-DD, optional, within the return window
	Notes            *string     `json:"notes" binding:"omitempty,max=4000"`
	Items            []itemInput `json:"items" binding:"required,min=1,max=200,dive"`
	Version          int         `json:"version"` // required when editing
}

// validateWindows checks the delivery and return windows an organizer picked.
func validateWindows(delivery, ret *Window, returnDate *string, now time.Time) error {
	if delivery == nil || !delivery.Active || delivery.Kind != WindowDelivery {
		return badRequest("invalid_delivery_window", "Wybrane okno dostawy jest niedostępne")
	}
	if !delivery.StartsAt.After(now) {
		return badRequest("delivery_window_past", "Okno dostawy już się rozpoczęło")
	}
	if ret == nil || !ret.Active || ret.Kind != WindowReturn {
		return badRequest("invalid_return_window", "Wybrane okno zwrotu jest niedostępne")
	}
	if ret.StartsAt.Before(delivery.EndsAt) {
		return badRequest("return_before_delivery", "Zwrot musi być po dostawie")
	}
	if returnDate != nil {
		d, err := time.ParseInLocation("2006-01-02", *returnDate, warsaw)
		if err != nil {
			return badRequest("invalid_return_date", "Data zwrotu musi mieć format RRRR-MM-DD")
		}
		first := dayStart(ret.StartsAt)
		last := dayStart(ret.EndsAt)
		if d.Before(first) || d.After(last) {
			return badRequest("return_date_outside_window", "Data zwrotu musi mieścić się w oknie zwrotu")
		}
	}
	return nil
}

func dayStart(t time.Time) time.Time {
	t = t.In(warsaw)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, warsaw)
}

// buildItems validates requested items against the catalog and snapshots name, category and
// price. strict applies organizer rules (active products only, max_per_order); moderators may
// override both.
func buildItems(in []itemInput, products map[int]Product, strict bool) ([]OrderItem, error) {
	if len(in) == 0 {
		return nil, badRequest("no_items", "Zamówienie musi mieć co najmniej jedną pozycję")
	}
	seen := map[int]bool{}
	items := make([]OrderItem, 0, len(in))
	for _, it := range in {
		if it.Quantity <= 0 {
			return nil, badRequest("invalid_quantity", "Ilość musi być większa od zera")
		}
		if seen[it.ProductID] {
			return nil, badRequest("duplicate_product", "Produkt występuje w zamówieniu więcej niż raz")
		}
		seen[it.ProductID] = true

		p, ok := products[it.ProductID]
		if !ok {
			return nil, badRequest("product_not_found", fmt.Sprintf("Produkt %d nie istnieje", it.ProductID))
		}
		if strict && !p.Active {
			return nil, badRequest("product_inactive", fmt.Sprintf("Produkt %q jest niedostępny", p.Name))
		}
		if strict && p.MaxPerOrder != nil && it.Quantity > *p.MaxPerOrder {
			return nil, badRequest("quantity_over_limit", fmt.Sprintf("Produkt %q: maksymalnie %d szt. w zamówieniu", p.Name, *p.MaxPerOrder))
		}
		items = append(items, OrderItem{
			ProductID:   p.ID,
			ProductName: p.Name,
			CategoryID:  p.CategoryID,
			Quantity:    it.Quantity,
			UnitPrice:   p.Price,
		})
	}
	return items, nil
}

func blankToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// ============================================================================
// Service
// ============================================================================

// Notifier is told about decisions on an order after they are committed. MVP: nothing is sent
// (moderators see the pending counter, organizers check the shop). A mailer plugs in here.
type Notifier interface {
	OrderDecided(ctx context.Context, o *Order)
}

type noopNotifier struct{}

func (noopNotifier) OrderDecided(context.Context, *Order) {}

type Service struct {
	repo     *Repository
	now      func() time.Time
	Notifier Notifier
	// OnQuestsChanged is called after a quest is created or moved (quest board SSE refresh).
	OnQuestsChanged func()
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo, now: time.Now, Notifier: noopNotifier{}}
}

func (s *Service) questsChanged() {
	if s.OnQuestsChanged != nil {
		go s.OnQuestsChanged()
	}
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	return s.repo.loadSettings(ctx, s.repo.db)
}

// validateOrganizerOrder runs every organizer-side rule and returns what to store.
func (s *Service) validateOrganizerOrder(ctx context.Context, tx *sql.Tx, accountID int, in orderInput) (orderFields, []OrderItem, error) {
	settings, err := s.repo.loadSettings(ctx, tx)
	if err != nil {
		return orderFields{}, nil, err
	}
	now := s.now()
	if !settings.OrdersAccepted(now) {
		return orderFields{}, nil, newErr(http.StatusForbidden, "orders_closed", "Przyjmowanie zamówień jest zamknięte")
	}

	f := orderFields{
		LocationID:       in.LocationID,
		LocationNote:     blankToNil(in.LocationNote),
		ContactName:      strings.TrimSpace(in.ContactName),
		BudgetOwner:      blankToNil(in.BudgetOwner),
		DeliveryWindowID: in.DeliveryWindowID,
		ReturnWindowID:   in.ReturnWindowID,
		ReturnDate:       blankToNil(in.ReturnDate),
		Notes:            blankToNil(in.Notes),
	}
	if f.ContactName == "" {
		if f.ContactName, err = s.repo.accountContactName(ctx, tx, accountID); err != nil {
			return f, nil, err
		}
	}

	loc, err := s.repo.locationByID(ctx, tx, f.LocationID)
	if err != nil {
		return f, nil, err
	}
	if loc == nil {
		return f, nil, badRequest("location_not_found", "Wybrana lokalizacja nie istnieje")
	}

	delivery, err := s.repo.windowByID(ctx, tx, f.DeliveryWindowID)
	if err != nil {
		return f, nil, err
	}
	ret, err := s.repo.windowByID(ctx, tx, f.ReturnWindowID)
	if err != nil {
		return f, nil, err
	}
	if err := validateWindows(delivery, ret, f.ReturnDate, now); err != nil {
		return f, nil, err
	}

	ids := make([]int, 0, len(in.Items))
	for _, it := range in.Items {
		ids = append(ids, it.ProductID)
	}
	products, err := s.repo.productsByID(ctx, tx, ids)
	if err != nil {
		return f, nil, err
	}
	items, err := buildItems(in.Items, products, true)
	return f, items, err
}

// ---------------------------------------------------------------------------
// Organizer
// ---------------------------------------------------------------------------

// SubmitOrder creates an order. The same Idempotency-Key returns the original order instead of
// creating a second one (double click, retry after a timeout).
func (s *Service) SubmitOrder(ctx context.Context, accountID int, in orderInput, idempotencyKey string) (*Order, error) {
	var key *string
	if k := strings.TrimSpace(idempotencyKey); k != "" {
		if len(k) > 100 {
			return nil, badRequest("invalid_idempotency_key", "Idempotency-Key może mieć najwyżej 100 znaków")
		}
		key = &k
		if id, err := s.repo.orderIDByIdempotencyKey(ctx, s.repo.db, accountID, k); err != nil {
			return nil, err
		} else if id != 0 {
			return s.replayed(ctx, id)
		}
	}

	var orderID int
	err := s.repo.inTx(ctx, func(tx *sql.Tx) error {
		f, items, err := s.validateOrganizerOrder(ctx, tx, accountID, in)
		if err != nil {
			return err
		}
		orderID, err = s.repo.insertOrder(ctx, tx, accountID, f, key)
		if err != nil {
			return err
		}
		if err := s.repo.replaceItems(ctx, tx, orderID, items); err != nil {
			return err
		}
		return s.repo.insertEvent(ctx, tx, orderID, ActorAccount, accountID, "submitted", nil)
	})
	if key != nil && isUniqueViolation(err) {
		// A concurrent request with the same key won the race.
		id, lookupErr := s.repo.orderIDByIdempotencyKey(ctx, s.repo.db, accountID, *key)
		if lookupErr == nil && id != 0 {
			return s.replayed(ctx, id)
		}
	}
	if err != nil {
		return nil, err
	}
	log.Printf("[shop] order %d submitted by account %d", orderID, accountID)
	return s.repo.orderByID(ctx, s.repo.db, orderID, false)
}

func (s *Service) replayed(ctx context.Context, id int) (*Order, error) {
	o, err := s.repo.orderByID(ctx, s.repo.db, id, false)
	if err != nil || o == nil {
		return o, err
	}
	o.idempotentHit = true
	return o, nil
}

// ownOrder locks an order and checks it belongs to the account; someone else's order is a 404.
func (s *Service) ownOrder(ctx context.Context, tx *sql.Tx, accountID, orderID int) (*Order, error) {
	o, err := s.repo.orderByID(ctx, tx, orderID, true)
	if err != nil {
		return nil, err
	}
	if o == nil || o.AccountID != accountID {
		return nil, notFound("Zamówienie nie istnieje")
	}
	return o, nil
}

func (s *Service) UpdateOrder(ctx context.Context, accountID, orderID int, in orderInput) (*Order, error) {
	err := s.repo.inTx(ctx, func(tx *sql.Tx) error {
		o, err := s.ownOrder(ctx, tx, accountID, orderID)
		if err != nil {
			return err
		}
		if err := checkTransition(o, actEdit, in.Version); err != nil {
			return err
		}
		f, items, err := s.validateOrganizerOrder(ctx, tx, accountID, in)
		if err != nil {
			return err
		}
		if err := s.repo.updateOrderFields(ctx, tx, orderID, f); err != nil {
			return err
		}
		if err := s.repo.replaceItems(ctx, tx, orderID, items); err != nil {
			return err
		}
		return s.repo.insertEvent(ctx, tx, orderID, ActorAccount, accountID, "updated", nil)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.orderByID(ctx, s.repo.db, orderID, false)
}

func (s *Service) CancelOrder(ctx context.Context, accountID, orderID, version int) (*Order, error) {
	err := s.repo.inTx(ctx, func(tx *sql.Tx) error {
		o, err := s.ownOrder(ctx, tx, accountID, orderID)
		if err != nil {
			return err
		}
		if err := checkTransition(o, actCancel, version); err != nil {
			return err
		}
		if err := s.repo.setStatus(ctx, tx, orderID, StatusCancelled, nil, nil); err != nil {
			return err
		}
		return s.repo.insertEvent(ctx, tx, orderID, ActorAccount, accountID, "cancelled", nil)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.orderByID(ctx, s.repo.db, orderID, false)
}

func (s *Service) AccountOrders(ctx context.Context, accountID int) ([]Order, error) {
	return s.repo.listOrders(ctx, orderFilter{AccountID: &accountID})
}

func (s *Service) AccountOrder(ctx context.Context, accountID, orderID int) (*Order, error) {
	o, err := s.repo.orderByID(ctx, s.repo.db, orderID, false)
	if err != nil {
		return nil, err
	}
	if o == nil || o.AccountID != accountID {
		return nil, notFound("Zamówienie nie istnieje")
	}
	return o, nil
}

// ---------------------------------------------------------------------------
// Warehouse (moderator)
// ---------------------------------------------------------------------------

func (s *Service) adminOrder(ctx context.Context, tx *sql.Tx, orderID int) (*Order, error) {
	o, err := s.repo.orderByID(ctx, tx, orderID, true)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, notFound("Zamówienie nie istnieje")
	}
	return o, nil
}

// ConfirmOrder locks the order and turns it into a quest, in one transaction.
func (s *Service) ConfirmOrder(ctx context.Context, userID, orderID, version int) (*Order, error) {
	err := s.repo.inTx(ctx, func(tx *sql.Tx) error {
		o, err := s.adminOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if err := checkTransition(o, actConfirm, version); err != nil {
			return err
		}
		if len(o.Items) == 0 {
			return conflict("no_items", "Zamówienie nie ma pozycji")
		}
		if err := s.repo.setStatus(ctx, tx, orderID, StatusConfirmed, nil, &userID); err != nil {
			return err
		}
		questID, err := s.repo.createQuest(ctx, tx, o)
		if err != nil {
			return err
		}
		return s.repo.insertEvent(ctx, tx, orderID, ActorUser, userID, "confirmed", map[string]any{"quest_id": questID})
	})
	if err != nil {
		return nil, err
	}
	log.Printf("[shop] order %d confirmed by user %d", orderID, userID)
	s.questsChanged()
	return s.decided(ctx, orderID)
}

func (s *Service) RejectOrder(ctx context.Context, userID, orderID, version int, reason string) (*Order, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, badRequest("reason_required", "Podaj powód odrzucenia")
	}
	err := s.repo.inTx(ctx, func(tx *sql.Tx) error {
		o, err := s.adminOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if err := checkTransition(o, actReject, version); err != nil {
			return err
		}
		if err := s.repo.setStatus(ctx, tx, orderID, StatusRejected, &reason, &userID); err != nil {
			return err
		}
		return s.repo.insertEvent(ctx, tx, orderID, ActorUser, userID, "rejected", map[string]any{"reason": reason})
	})
	if err != nil {
		return nil, err
	}
	return s.decided(ctx, orderID)
}

type adminPatchInput struct {
	Version      int          `json:"version" binding:"required"`
	LocationID   *int         `json:"location_id"`
	LocationNote *string      `json:"location_note" binding:"omitempty,max=1000"`
	Items        *[]itemInput `json:"items" binding:"omitempty,max=200,dive"`
}

// PatchOrder lets the warehouse correct an order: items while it is submitted, the location
// also after confirmation (the quest follows in the same transaction).
func (s *Service) PatchOrder(ctx context.Context, userID, orderID int, in adminPatchInput) (*Order, error) {
	if in.LocationID == nil && in.Items == nil {
		return nil, badRequest("nothing_to_change", "Podaj location_id albo items")
	}
	movedQuest := false
	err := s.repo.inTx(ctx, func(tx *sql.Tx) error {
		o, err := s.adminOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if in.Version != o.Version {
			return conflict("version_conflict", "Zamówienie zostało w międzyczasie zmienione — odśwież i spróbuj ponownie")
		}

		if in.Items != nil {
			if err := checkTransition(o, actChangeItems, in.Version); err != nil {
				return err
			}
			ids := make([]int, 0, len(*in.Items))
			for _, it := range *in.Items {
				ids = append(ids, it.ProductID)
			}
			products, err := s.repo.productsByID(ctx, tx, ids)
			if err != nil {
				return err
			}
			items, err := buildItems(*in.Items, products, false)
			if err != nil {
				return err
			}
			if err := s.repo.replaceItems(ctx, tx, orderID, items); err != nil {
				return err
			}
			if err := s.repo.insertEvent(ctx, tx, orderID, ActorUser, userID, "items_changed", nil); err != nil {
				return err
			}
		}

		if in.LocationID != nil {
			if err := checkTransition(o, actChangeLocation, in.Version); err != nil {
				return err
			}
			loc, err := s.repo.locationByID(ctx, tx, *in.LocationID)
			if err != nil {
				return err
			}
			if loc == nil {
				return badRequest("location_not_found", "Wybrana lokalizacja nie istnieje")
			}
			note := o.LocationNote
			if in.LocationNote != nil {
				note = blankToNil(in.LocationNote)
			}
			if err := s.repo.setLocation(ctx, tx, orderID, loc.ID, note); err != nil {
				return err
			}
			if o.Status == StatusConfirmed {
				if err := s.repo.moveQuest(ctx, tx, orderID, *loc); err != nil {
					return err
				}
				movedQuest = true
			}
			if err := s.repo.insertEvent(ctx, tx, orderID, ActorUser, userID, "location_changed",
				map[string]any{"from": o.Location.ID, "to": loc.ID}); err != nil {
				return err
			}
		}
		return s.repo.bumpVersion(ctx, tx, orderID)
	})
	if err != nil {
		return nil, err
	}
	if movedQuest {
		s.questsChanged()
	}
	return s.repo.orderByID(ctx, s.repo.db, orderID, false)
}

// decided reloads an order after confirm/reject and hands it to the notifier.
func (s *Service) decided(ctx context.Context, orderID int) (*Order, error) {
	o, err := s.repo.orderByID(ctx, s.repo.db, orderID, false)
	if err != nil || o == nil {
		return o, err
	}
	if s.Notifier != nil {
		s.Notifier.OrderDecided(ctx, o)
	}
	return o, nil
}

// ---------------------------------------------------------------------------
// Presentation
// ---------------------------------------------------------------------------

// present fills totals and, for organizers with prices hidden, strips every price (D8: the API
// must not return them, not just the UI).
func present(o *Order, showPrices bool) {
	if !showPrices {
		for i := range o.Items {
			o.Items[i].UnitPrice = nil
		}
		o.Total = nil
		return
	}
	var total float64
	priced := false
	for _, it := range o.Items {
		if it.UnitPrice != nil {
			total += *it.UnitPrice * float64(it.Quantity)
			priced = true
		}
	}
	if priced {
		o.Total = &total
	}
}
