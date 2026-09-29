package shop

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func errCode(t *testing.T, err error) string {
	t.Helper()
	var e *Error
	require.True(t, errors.As(err, &e), "expected *shop.Error, got %v", err)
	return e.Code
}

func TestTransitions(t *testing.T) {
	all := []action{actEdit, actCancel, actConfirm, actReject, actChangeItems, actChangeLocation}
	allowed := map[string][]action{
		StatusSubmitted: all,
		StatusConfirmed: {actChangeLocation},
		StatusRejected:  {},
		StatusCancelled: {},
	}

	for status, ok := range allowed {
		for _, a := range all {
			want := false
			for _, x := range ok {
				if x == a {
					want = true
				}
			}
			t.Run(status+"/"+string(a), func(t *testing.T) {
				assert.Equal(t, want, canDo(status, a))
			})
		}
	}
	assert.False(t, canDo("unknown", actEdit))
}

func TestCheckTransition(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		version  int
		act      action
		wantCode string
	}{
		{"allowed with matching version", StatusSubmitted, 3, actConfirm, ""},
		{"stale version", StatusSubmitted, 2, actConfirm, "version_conflict"},
		{"second confirm", StatusConfirmed, 3, actConfirm, "already_confirmed"},
		{"edit after confirm", StatusConfirmed, 3, actEdit, "invalid_status"},
		{"cancel after reject", StatusRejected, 3, actCancel, "invalid_status"},
		{"location after confirm", StatusConfirmed, 3, actChangeLocation, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTransition(&Order{Status: tt.status, Version: 3}, tt.act, tt.version)
			if tt.wantCode == "" {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, tt.wantCode, errCode(t, err))
			assert.Equal(t, http.StatusConflict, err.(*Error).Status)
		})
	}
}

func warsawTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, warsaw)
	if err != nil {
		panic(err)
	}
	return t
}

func TestValidateWindows(t *testing.T) {
	now := warsawTime("2027-07-01 12:00")
	delivery := &Window{Kind: WindowDelivery, Active: true, StartsAt: warsawTime("2027-07-15 16:00"), EndsAt: warsawTime("2027-07-15 20:00")}
	ret := &Window{Kind: WindowReturn, Active: true, StartsAt: warsawTime("2027-07-18 18:00"), EndsAt: warsawTime("2027-07-19 02:00")}
	with := func(w *Window, f func(*Window)) *Window { c := *w; f(&c); return &c }
	date := func(s string) *string { return &s }

	tests := []struct {
		name       string
		delivery   *Window
		ret        *Window
		returnDate *string
		wantCode   string
	}{
		{"valid", delivery, ret, nil, ""},
		{"return date on the window's first day", delivery, ret, date("2027-07-18"), ""},
		{"return date on the day a night window ends", delivery, ret, date("2027-07-19"), ""},
		{"return date before the window", delivery, ret, date("2027-07-17"), "return_date_outside_window"},
		{"return date after the window", delivery, ret, date("2027-07-20"), "return_date_outside_window"},
		{"malformed return date", delivery, ret, date("18.07.2027"), "invalid_return_date"},
		{"missing delivery window", nil, ret, nil, "invalid_delivery_window"},
		{"inactive delivery window", with(delivery, func(w *Window) { w.Active = false }), ret, nil, "invalid_delivery_window"},
		{"return window given as delivery", ret, ret, nil, "invalid_delivery_window"},
		{"delivery already started", with(delivery, func(w *Window) { w.StartsAt = now.Add(-time.Hour) }), ret, nil, "delivery_window_past"},
		{"missing return window", delivery, nil, nil, "invalid_return_window"},
		{"delivery window given as return", delivery, delivery, nil, "invalid_return_window"},
		{"inactive return window", delivery, with(ret, func(w *Window) { w.Active = false }), nil, "invalid_return_window"},
		{"return overlaps delivery", delivery, with(ret, func(w *Window) { w.StartsAt = warsawTime("2027-07-15 19:00") }), nil, "return_before_delivery"},
		{"return right after delivery", delivery, with(ret, func(w *Window) { w.StartsAt = delivery.EndsAt }), nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWindows(tt.delivery, tt.ret, tt.returnDate, now)
			if tt.wantCode == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.wantCode, errCode(t, err))
			}
		})
	}
}

func TestBuildItems(t *testing.T) {
	price := 12.5
	limit := 2
	products := map[int]Product{
		1: {ID: 1, Name: "Laptop", CategoryID: 10, Price: &price, Active: true},
		2: {ID: 2, Name: "Przedłużacz", CategoryID: 20, MaxPerOrder: &limit, Active: true},
		3: {ID: 3, Name: "Stary rzutnik", CategoryID: 30, Active: false},
	}

	tests := []struct {
		name     string
		in       []itemInput
		strict   bool
		wantCode string
	}{
		{"valid", []itemInput{{1, 3}, {2, 2}}, true, ""},
		{"empty", nil, true, "no_items"},
		{"zero quantity", []itemInput{{1, 0}}, true, "invalid_quantity"},
		{"duplicate product", []itemInput{{1, 1}, {1, 2}}, true, "duplicate_product"},
		{"unknown product", []itemInput{{99, 1}}, true, "product_not_found"},
		{"inactive product", []itemInput{{3, 1}}, true, "product_inactive"},
		{"over the limit", []itemInput{{2, 3}}, true, "quantity_over_limit"},
		{"moderator may use an inactive product", []itemInput{{3, 1}}, false, ""},
		{"moderator may exceed the limit", []itemInput{{2, 5}}, false, ""},
		{"moderator still cannot use an unknown product", []itemInput{{99, 1}}, false, "product_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := buildItems(tt.in, products, tt.strict)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, errCode(t, err))
				return
			}
			require.NoError(t, err)
			require.Len(t, items, len(tt.in))
			for i, it := range items {
				p := products[tt.in[i].ProductID]
				assert.Equal(t, p.Name, it.ProductName)
				assert.Equal(t, p.CategoryID, it.CategoryID)
				assert.Equal(t, p.Price, it.UnitPrice)
			}
		})
	}
}

func TestPresent(t *testing.T) {
	price := 10.0
	newOrder := func() *Order {
		return &Order{Items: []OrderItem{
			{ProductID: 1, Quantity: 3, UnitPrice: &price},
			{ProductID: 2, Quantity: 5}, // no price
		}}
	}

	shown := newOrder()
	present(shown, true)
	require.NotNil(t, shown.Total)
	assert.Equal(t, 30.0, *shown.Total)
	assert.NotNil(t, shown.Items[0].UnitPrice)

	hidden := newOrder()
	present(hidden, false)
	assert.Nil(t, hidden.Total)
	for _, it := range hidden.Items {
		assert.Nil(t, it.UnitPrice)
	}

	unpriced := &Order{Items: []OrderItem{{ProductID: 2, Quantity: 1}}}
	present(unpriced, true)
	assert.Nil(t, unpriced.Total)
}

func TestOrdersAccepted(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	assert.False(t, Settings{OrdersOpen: false}.OrdersAccepted(now))
	assert.True(t, Settings{OrdersOpen: true}.OrdersAccepted(now))
	assert.True(t, Settings{OrdersOpen: true, OrdersOpenUntil: &future}.OrdersAccepted(now))
	assert.False(t, Settings{OrdersOpen: true, OrdersOpenUntil: &past}.OrdersAccepted(now))
}
