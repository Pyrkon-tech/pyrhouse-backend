package shop

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"warehouse/internal/rate_limiter"
	"warehouse/internal/security"
	"warehouse/internal/shop/access"

	"github.com/gin-gonic/gin"
)

// inviteTTL is how long an invite link works (D17).
const inviteTTL = 48 * time.Hour

type Handler struct {
	service      *Service
	auth         *AuthService
	repo         *Repository
	shopURL      string
	loginLimiter *rate_limiter.RateLimiter
}

func NewHandler(service *Service, auth *AuthService, repo *Repository, shopURL string) *Handler {
	return &Handler{
		service: service,
		auth:    auth,
		repo:    repo,
		shopURL: strings.TrimRight(shopURL, "/"),
		// Generous per IP: organizers share the event Wi-Fi NAT. Codes are single-use and
		// invites carry 256 bits, so the limit only curbs noise and Google API quota.
		loginLimiter: rate_limiter.NewRateLimiter(30, 5*time.Minute),
	}
}

// RegisterShopRoutes registers the organizer API. It must not sit behind JWTMiddleware:
// /shop/* accepts shop tokens only (ShopAuth), warehouse tokens are rejected.
func (h *Handler) RegisterShopRoutes(router *gin.Engine) {
	g := router.Group("/shop", limitBody(1<<20))
	g.POST("/auth/google/exchange", h.login)

	a := g.Group("", ShopAuth(h.repo))
	a.GET("/me", h.me)
	a.GET("/config", h.config)
	a.GET("/catalog", h.catalog)
	a.GET("/locations", h.locations)
	a.GET("/delivery-windows", h.upcomingWindows)
	a.GET("/budget-owners", h.budgetOwners)
	a.GET("/orders", h.myOrders)
	a.GET("/orders/:id", h.myOrder)
	a.POST("/orders", h.submitOrder)
	a.PUT("/orders/:id", h.updateOrder)
	a.POST("/orders/:id/cancel", h.cancelOrder)
}

// RegisterAdminRoutes registers the warehouse panel API on a JWTMiddleware-protected group.
func (h *Handler) RegisterAdminRoutes(protected *gin.RouterGroup) {
	g := protected.Group("/admin/shop", security.Authorize("moderator"))
	g.GET("/products", h.adminProducts)
	g.POST("/products", h.createProduct)
	g.PUT("/products/:id", h.updateProduct)

	g.GET("/delivery-windows", h.adminWindows)
	g.POST("/delivery-windows", h.createWindow)
	g.PUT("/delivery-windows/:id", h.updateWindow)
	g.DELETE("/delivery-windows/:id", h.deleteWindow)

	g.GET("/orders", h.adminOrders)
	g.GET("/orders/summary", h.summary)
	g.GET("/orders/:id", h.adminOrder)
	g.POST("/orders/:id/confirm", h.confirmOrder)
	g.POST("/orders/:id/reject", h.rejectOrder)
	g.PATCH("/orders/:id", h.patchOrder)

	g.GET("/accounts", h.accounts)
	g.POST("/accounts", h.allowlistAccount)
	g.PATCH("/accounts/:id", h.setAccountActive)

	g.GET("/invites", h.invites)
	g.POST("/invites", h.createInvite)
	g.DELETE("/invites/:id", h.revokeInvite)

	g.GET("/settings", security.Authorize("admin"), h.getSettings)
	g.PUT("/settings", security.Authorize("admin"), h.putSettings)
}

// ============================================================================
// Helpers
// ============================================================================

func abort(c *gin.Context, err error) {
	var e *Error
	if errors.As(err, &e) {
		c.AbortWithStatusJSON(e.Status, gin.H{"error": e.Message, "code": e.Code})
		return
	}
	log.Printf("[shop] %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Wewnętrzny błąd serwera", "code": "internal"})
}

// limitBody caps request bodies; oversized ones fail to bind with 400.
func limitBody(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		c.Next()
	}
}

func bind(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane", "code": "invalid_body", "details": err.Error()})
		return false
	}
	return true
}

func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		abort(c, badRequest("invalid_id", "Nieprawidłowe ID"))
		return 0, false
	}
	return id, true
}

func userID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.GetString("userID"))
	if err != nil || id <= 0 {
		abort(c, newErr(http.StatusUnauthorized, "unauthorized", "Brak użytkownika"))
		return 0, false
	}
	return id, true
}

func (h *Handler) showPrices(c *gin.Context) (bool, bool) {
	s, err := h.service.Settings(c.Request.Context())
	if err != nil {
		abort(c, err)
		return false, false
	}
	return s.ShowPrices, true
}

type versionInput struct {
	Version int `json:"version" binding:"required"`
}

// ============================================================================
// Organizer
// ============================================================================

func (h *Handler) login(c *gin.Context) {
	if !h.loginLimiter.IsAllowed(c.ClientIP()) {
		abort(c, newErr(http.StatusTooManyRequests, "rate_limited", "Za dużo prób logowania — spróbuj za kilka minut"))
		return
	}
	var in loginInput
	if !bind(c, &in) {
		return
	}
	token, acc, err := h.auth.Login(c.Request.Context(), in)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "account": acc})
}

func (h *Handler) me(c *gin.Context) {
	acc, err := h.repo.accountByID(c.Request.Context(), accountID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, acc)
}

func (h *Handler) config(c *gin.Context) {
	s, err := h.service.Settings(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"show_prices":       s.ShowPrices,
		"orders_open":       s.OrdersAccepted(time.Now()),
		"orders_open_until": s.OrdersOpenUntil,
	})
}

func (h *Handler) catalog(c *gin.Context) {
	show, ok := h.showPrices(c)
	if !ok {
		return
	}
	products, err := h.repo.listProducts(c.Request.Context(), true)
	if err != nil {
		abort(c, err)
		return
	}
	for i := range products {
		products[i].CategoryName = nil // warehouse detail, not for organizers
		if !show {
			products[i].Price = nil
		}
	}
	c.JSON(http.StatusOK, products)
}

func (h *Handler) locations(c *gin.Context) {
	locs, err := h.repo.listLocations(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, locs)
}

func (h *Handler) upcomingWindows(c *gin.Context) {
	windows, err := h.repo.listWindows(c.Request.Context(), true)
	if err != nil {
		abort(c, err)
		return
	}
	for i := range windows {
		windows[i].OrdersCount = 0 // other organizers' volume is not theirs to see
	}
	c.JSON(http.StatusOK, windows)
}

func (h *Handler) budgetOwners(c *gin.Context) {
	list, err := h.repo.budgetOwners(c.Request.Context(), accountID(c))
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) myOrders(c *gin.Context) {
	show, ok := h.showPrices(c)
	if !ok {
		return
	}
	orders, err := h.service.AccountOrders(c.Request.Context(), accountID(c))
	if err != nil {
		abort(c, err)
		return
	}
	for i := range orders {
		present(&orders[i], show)
		forOrganizer(&orders[i])
	}
	c.JSON(http.StatusOK, orders)
}

func (h *Handler) respondOrder(c *gin.Context, status int, o *Order) {
	show, ok := h.showPrices(c)
	if !ok {
		return
	}
	present(o, show)
	forOrganizer(o)
	c.JSON(status, o)
}

// forOrganizer drops warehouse-side detail from an organizer's view of their order.
func forOrganizer(o *Order) {
	for i := range o.Items {
		o.Items[i].CategoryName = nil
	}
}

func (h *Handler) myOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	o, err := h.service.AccountOrder(c.Request.Context(), accountID(c), id)
	if err != nil {
		abort(c, err)
		return
	}
	h.respondOrder(c, http.StatusOK, o)
}

func (h *Handler) submitOrder(c *gin.Context) {
	var in orderInput
	if !bind(c, &in) {
		return
	}
	o, err := h.service.SubmitOrder(c.Request.Context(), accountID(c), in, c.GetHeader("Idempotency-Key"))
	if err != nil {
		abort(c, err)
		return
	}
	status := http.StatusCreated
	if o.idempotentHit {
		status = http.StatusOK
	}
	h.respondOrder(c, status, o)
}

func (h *Handler) updateOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in orderInput
	if !bind(c, &in) {
		return
	}
	if in.Version <= 0 {
		abort(c, badRequest("version_required", "Podaj version"))
		return
	}
	o, err := h.service.UpdateOrder(c.Request.Context(), accountID(c), id, in)
	if err != nil {
		abort(c, err)
		return
	}
	h.respondOrder(c, http.StatusOK, o)
}

func (h *Handler) cancelOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in versionInput
	if !bind(c, &in) {
		return
	}
	o, err := h.service.CancelOrder(c.Request.Context(), accountID(c), id, in.Version)
	if err != nil {
		abort(c, err)
		return
	}
	h.respondOrder(c, http.StatusOK, o)
}

// ============================================================================
// Warehouse panel
// ============================================================================

func (h *Handler) adminProducts(c *gin.Context) {
	products, err := h.repo.listProducts(c.Request.Context(), false)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

func (h *Handler) saveProduct(c *gin.Context, id int) {
	uid, ok := userID(c)
	if !ok {
		return
	}
	var in productInput
	if !bind(c, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Section = strings.TrimSpace(in.Section)
	in.Description = blankToNil(in.Description)
	in.ImageURL = blankToNil(in.ImageURL)
	if in.Name == "" {
		abort(c, badRequest("name_required", "Podaj nazwę produktu"))
		return
	}
	if in.Price != nil && *in.Price < 0 {
		abort(c, badRequest("invalid_price", "Cena nie może być ujemna"))
		return
	}
	if in.MaxPerOrder != nil && *in.MaxPerOrder <= 0 {
		abort(c, badRequest("invalid_max_per_order", "Limit na zamówienie musi być większy od zera"))
		return
	}
	if in.ImageURL != nil && !strings.HasPrefix(*in.ImageURL, "https://") {
		abort(c, badRequest("invalid_image_url", "Adres zdjęcia musi zaczynać się od https://"))
		return
	}
	p, err := h.repo.saveProduct(c.Request.Context(), id, in, uid)
	if err != nil {
		abort(c, err)
		return
	}
	status := http.StatusOK
	if id == 0 {
		status = http.StatusCreated
	}
	c.JSON(status, p)
}

func (h *Handler) createProduct(c *gin.Context) { h.saveProduct(c, 0) }

func (h *Handler) updateProduct(c *gin.Context) {
	if id, ok := pathID(c); ok {
		h.saveProduct(c, id)
	}
}

func (h *Handler) adminWindows(c *gin.Context) {
	windows, err := h.repo.listWindows(c.Request.Context(), false)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, windows)
}

func (h *Handler) saveWindow(c *gin.Context, id int) {
	var in windowInput
	if !bind(c, &in) {
		return
	}
	in.Label = strings.TrimSpace(in.Label)
	w, err := h.repo.saveWindow(c.Request.Context(), id, in)
	if err != nil {
		abort(c, err)
		return
	}
	status := http.StatusOK
	if id == 0 {
		status = http.StatusCreated
	}
	c.JSON(status, w)
}

func (h *Handler) createWindow(c *gin.Context) { h.saveWindow(c, 0) }

func (h *Handler) updateWindow(c *gin.Context) {
	if id, ok := pathID(c); ok {
		h.saveWindow(c, id)
	}
}

func (h *Handler) deleteWindow(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.repo.deleteWindow(c.Request.Context(), id); err != nil {
		abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) adminOrders(c *gin.Context) {
	status := c.Query("status")
	if status != "" {
		if _, known := transitions[status]; !known {
			abort(c, badRequest("invalid_status", "Nieznany status"))
			return
		}
	}
	orders, err := h.repo.listOrders(c.Request.Context(), orderFilter{Status: status})
	if err != nil {
		abort(c, err)
		return
	}
	for i := range orders {
		present(&orders[i], true)
	}
	c.JSON(http.StatusOK, orders)
}

type adminOrderView struct {
	*Order
	Events []Event `json:"events"`
}

func (h *Handler) adminOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	o, err := h.repo.orderByID(c.Request.Context(), h.repo.db, id, false)
	if err != nil {
		abort(c, err)
		return
	}
	if o == nil {
		abort(c, notFound("Zamówienie nie istnieje"))
		return
	}
	events, err := h.repo.listEvents(c.Request.Context(), id)
	if err != nil {
		abort(c, err)
		return
	}
	present(o, true)
	c.JSON(http.StatusOK, adminOrderView{Order: o, Events: events})
}

func (h *Handler) respondAdminOrder(c *gin.Context, o *Order, err error) {
	if err != nil {
		abort(c, err)
		return
	}
	present(o, true)
	c.JSON(http.StatusOK, o)
}

func (h *Handler) confirmOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid, ok := userID(c)
	if !ok {
		return
	}
	var in versionInput
	if !bind(c, &in) {
		return
	}
	o, err := h.service.ConfirmOrder(c.Request.Context(), uid, id, in.Version)
	h.respondAdminOrder(c, o, err)
}

func (h *Handler) rejectOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid, ok := userID(c)
	if !ok {
		return
	}
	var in struct {
		Version int    `json:"version" binding:"required"`
		Reason  string `json:"reason" binding:"required"`
	}
	if !bind(c, &in) {
		return
	}
	o, err := h.service.RejectOrder(c.Request.Context(), uid, id, in.Version, in.Reason)
	h.respondAdminOrder(c, o, err)
}

func (h *Handler) patchOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid, ok := userID(c)
	if !ok {
		return
	}
	var in adminPatchInput
	if !bind(c, &in) {
		return
	}
	o, err := h.service.PatchOrder(c.Request.Context(), uid, id, in)
	h.respondAdminOrder(c, o, err)
}

func (h *Handler) summary(c *gin.Context) {
	group := c.DefaultQuery("group", "product")
	if group != "product" && group != "day" && group != "location" {
		abort(c, badRequest("invalid_group", "group musi być product, day albo location"))
		return
	}
	rows, err := h.repo.summary(c.Request.Context(), group, c.Query("status") == "confirmed")
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"group": group, "rows": rows})
}

func (h *Handler) accounts(c *gin.Context) {
	list, err := h.repo.listAccounts(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) allowlistAccount(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		return
	}
	var in struct {
		Email string `json:"email" binding:"required,email,max=255"`
	}
	if !bind(c, &in) {
		return
	}
	acc, err := h.repo.createAllowlistAccount(c.Request.Context(), access.NormalizeEmail(in.Email), uid)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, acc)
}

func (h *Handler) setAccountActive(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var in struct {
		Active *bool `json:"active" binding:"required"`
	}
	if !bind(c, &in) {
		return
	}
	acc, err := h.repo.setAccountActive(c.Request.Context(), id, *in.Active)
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, acc)
}

func (h *Handler) invites(c *gin.Context) {
	list, err := h.repo.listInvites(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

// createInvite returns the plaintext token exactly once; only its hash is stored.
func (h *Handler) createInvite(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		return
	}
	var in struct {
		Label string `json:"label" binding:"required,max=255"`
	}
	if !bind(c, &in) {
		return
	}
	label := strings.TrimSpace(in.Label)
	if label == "" {
		abort(c, badRequest("label_required", "Podaj etykietę zaproszenia"))
		return
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		abort(c, err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	inv, err := h.repo.createInvite(c.Request.Context(), access.HashInviteToken(token), label, uid, time.Now().Add(inviteTTL))
	if err != nil {
		abort(c, err)
		return
	}
	resp := gin.H{"invite": inv, "token": token}
	if h.shopURL != "" {
		resp["url"] = h.shopURL + "/invite/" + token
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) revokeInvite(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.repo.revokeInvite(c.Request.Context(), id); err != nil {
		abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) getSettings(c *gin.Context) {
	s, err := h.service.Settings(c.Request.Context())
	if err != nil {
		abort(c, err)
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) putSettings(c *gin.Context) {
	var in struct {
		ShowPrices      *bool      `json:"show_prices" binding:"required"`
		OrdersOpen      *bool      `json:"orders_open" binding:"required"`
		OrdersOpenUntil *time.Time `json:"orders_open_until"`
		DomainAutoJoin  *bool      `json:"domain_auto_join" binding:"required"`
		AutoDomains     []string   `json:"auto_domains"`
	}
	if !bind(c, &in) {
		return
	}
	domains := access.ParseDomains(strings.Join(in.AutoDomains, ","))
	for _, d := range domains {
		if strings.ContainsAny(d, " @/") || !strings.Contains(d, ".") {
			abort(c, badRequest("invalid_domain", "Nieprawidłowa domena: "+d))
			return
		}
	}
	if domains == nil {
		domains = []string{}
	}
	s := Settings{
		ShowPrices:      *in.ShowPrices,
		OrdersOpen:      *in.OrdersOpen,
		OrdersOpenUntil: in.OrdersOpenUntil,
		DomainAutoJoin:  *in.DomainAutoJoin,
		AutoDomains:     domains,
	}
	if err := h.repo.saveSettings(c.Request.Context(), s); err != nil {
		abort(c, err)
		return
	}
	h.getSettings(c)
}
