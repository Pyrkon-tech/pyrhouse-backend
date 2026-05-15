package reservations

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"warehouse/internal/auditlog"
	"warehouse/internal/inventory/assets"
	"warehouse/internal/origins"
	"warehouse/internal/repository"
)

func testDBURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:pyrpyr@localhost:15432/pyrhouse_test?sslmode=disable"
}

type resFixtures struct {
	laptopCategoryID int
	laptopPyrID      string
	plainOriginSlug  string
}

func setupResTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("postgres", testDBURL())
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}
	doCleanup := func() {
		_, _ = db.Exec("DELETE FROM pyr_code_reservations WHERE pyr_code LIKE 'PYR-TSLP%'")
		_, _ = db.Exec("DELETE FROM items WHERE item_serial LIKE '__TEST__%' OR pyr_code LIKE 'PYR-TSLP%'")
		_, _ = db.Exec("DELETE FROM item_category WHERE item_category = '__test__tslaptop'")
	}
	doCleanup() // pre-clean stale data from previous runs
	cleanup := func() {
		doCleanup()
		_ = db.Close()
	}
	return db, cleanup
}

func createResFixtures(t *testing.T, db *sql.DB) resFixtures {
	t.Helper()

	var catID int
	require.NoError(t, db.QueryRow(
		"INSERT INTO item_category (item_category, label, pyr_id, category_type) VALUES ('__test__tslaptop', '__TEST__TSLaptop', 'TSLP', 'asset') ON CONFLICT (item_category) DO UPDATE SET label = EXCLUDED.label RETURNING id",
	).Scan(&catID))

	var plainSlug string
	require.NoError(t, db.QueryRow("SELECT slug FROM origins WHERE active = true AND allow_suffix = false LIMIT 1").Scan(&plainSlug))

	return resFixtures{
		laptopCategoryID: catID,
		laptopPyrID:      "TSLP",
		plainOriginSlug:  plainSlug,
	}
}

func newResRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Next()
	})

	repo := repository.NewRepository(db)
	assetRepo := assets.NewRepository(repo)
	al := auditlog.NewAuditLog(auditlog.NewRepository(repo))
	originsRepo := origins.NewRepository(repo)
	originSvc := origins.NewService(originsRepo)
	resRepo := NewRepository(repo)
	resSvc := NewService(resRepo, assetRepo, repo, originSvc, al)
	h := NewHandler(resSvc)
	h.RegisterRoutes(r.Group("/"))

	// also wire asset handler so we can POST /assets in number-continuity tests
	ah := assets.NewAssetHandler(repo, assetRepo, al, originSvc)
	ah.RegisterRoutes(r.Group("/"))

	return r
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func doReserve(t *testing.T, r *gin.Engine, categoryID, qty int) []PyrCodeReservation {
	t.Helper()
	body := toJSON(t, map[string]any{"category_id": categoryID, "quantity": qty})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/assets/reservations", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp struct {
		Reservations []PyrCodeReservation `json:"reservations"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp.Reservations
}

func doClaim(t *testing.T, r *gin.Engine, origin string, locationID int, items []map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body := toJSON(t, map[string]any{
		"origin":      origin,
		"location_id": locationID,
		"items":       items,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/assets/reservations/claim", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func doAddAsset(t *testing.T, r *gin.Engine, categoryID int, serial, origin string) *httptest.ResponseRecorder {
	t.Helper()
	body := toJSON(t, map[string]any{
		"category_id": categoryID,
		"serial":      serial,
		"location_id": 1,
		"origin":      origin,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/assets", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// TestReserve_Basic — reserve N codes, verify all are in DB and unclaimed.
func TestReserve_Basic(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 3)
	require.Len(t, reservations, 3)

	for _, res := range reservations {
		assert.NotEmpty(t, res.PyrCode)
		assert.Equal(t, fix.laptopCategoryID, res.CategoryID)
		assert.Nil(t, res.ClaimedAt)
	}

	// verify via GET
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/assets/reservations?category_id=%d", fix.laptopCategoryID), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var got []PyrCodeReservation
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Len(t, got, 3)
}

// TestReserve_FromEmpty_Claim_ThenAddNormal — reserve from scratch, claim, then add normal asset (should get next number).
func TestReserve_FromEmpty_Claim_ThenAddNormal(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 2)
	require.Len(t, reservations, 2)

	pyrCode1 := reservations[0].PyrCode
	pyrCode2 := reservations[1].PyrCode

	// claim both
	w := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": pyrCode1, "serial": "__TEST__res-serial-001"},
		{"pyr_code": pyrCode2, "serial": "__TEST__res-serial-002"},
	})
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// add normal asset — must get TSLP3
	wNormal := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-serial-003", fix.plainOriginSlug)
	assert.Equal(t, http.StatusCreated, wNormal.Code, wNormal.Body.String())
	var normalResp map[string]any
	require.NoError(t, json.NewDecoder(wNormal.Body).Decode(&normalResp))
	assert.Equal(t, "PYR-TSLP3", normalResp["pyrcode"], "normal asset after claiming TSLP1+2 should be TSLP3")
}

// TestReserve_ExistingAssets_Claim_ThenAddNormal — existing assets, reserve more, claim, add normal.
func TestReserve_ExistingAssets_Claim_ThenAddNormal(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	// add 2 assets manually (TSLP1, TSLP2)
	w1 := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-exist-001", fix.plainOriginSlug)
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())
	w2 := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-exist-002", fix.plainOriginSlug)
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())

	// reserve 2 more (should get TSLP3, TSLP4)
	reservations := doReserve(t, r, fix.laptopCategoryID, 2)
	require.Len(t, reservations, 2)
	assert.Equal(t, "PYR-TSLP3", reservations[0].PyrCode)
	assert.Equal(t, "PYR-TSLP4", reservations[1].PyrCode)

	// claim both reservations
	w := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": reservations[0].PyrCode, "serial": "__TEST__res-exist-003"},
		{"pyr_code": reservations[1].PyrCode, "serial": "__TEST__res-exist-004"},
	})
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// add another normal asset — must be TSLP5
	wNormal := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-exist-005", fix.plainOriginSlug)
	assert.Equal(t, http.StatusCreated, wNormal.Code, wNormal.Body.String())
	var normalResp map[string]any
	require.NoError(t, json.NewDecoder(wNormal.Body).Decode(&normalResp))
	assert.Equal(t, "PYR-TSLP5", normalResp["pyrcode"])
}

// TestReserve_AddNormal_DeleteReservation_AddNormal — reserve blocks numbers even after delete if items exist.
func TestReserve_AddNormal_DeleteReservation_AddNormal(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	// reserve TSLP1, TSLP2
	reservations := doReserve(t, r, fix.laptopCategoryID, 2)
	require.Len(t, reservations, 2)

	// add normal asset — generator sees MAX(items=0, reservations=2)=2 → TSLP3
	wNormal := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-del-001", fix.plainOriginSlug)
	require.Equal(t, http.StatusCreated, wNormal.Code, wNormal.Body.String())
	var normalResp map[string]any
	require.NoError(t, json.NewDecoder(wNormal.Body).Decode(&normalResp))
	assert.Equal(t, "PYR-TSLP3", normalResp["pyrcode"])

	// delete reservations
	delBody := toJSON(t, map[string]any{"pyr_codes": []string{reservations[0].PyrCode, reservations[1].PyrCode}})
	wDel := httptest.NewRecorder()
	reqDel, _ := http.NewRequest(http.MethodDelete, "/assets/reservations", delBody)
	reqDel.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wDel, reqDel)
	assert.Equal(t, http.StatusOK, wDel.Code)

	// add another normal asset — items has TSLP3, reservations empty → MAX=3 → TSLP4
	wNormal2 := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-del-002", fix.plainOriginSlug)
	require.Equal(t, http.StatusCreated, wNormal2.Code, wNormal2.Body.String())
	var normalResp2 map[string]any
	require.NoError(t, json.NewDecoder(wNormal2.Body).Decode(&normalResp2))
	assert.Equal(t, "PYR-TSLP4", normalResp2["pyrcode"], "TSLP1/2 are NOT recycled since TSLP3 exists in items")
}

// TestReserve_DeleteWithoutClaim_NumberRecycled — deleting reservations before any asset is created recycles numbers.
func TestReserve_DeleteWithoutClaim_NumberRecycled(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	// reserve TSLP1, TSLP2 (no existing assets)
	reservations := doReserve(t, r, fix.laptopCategoryID, 2)
	require.Len(t, reservations, 2)

	// delete all reservations without claiming
	delBody := toJSON(t, map[string]any{"pyr_codes": []string{reservations[0].PyrCode, reservations[1].PyrCode}})
	wDel := httptest.NewRecorder()
	reqDel, _ := http.NewRequest(http.MethodDelete, "/assets/reservations", delBody)
	reqDel.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wDel, reqDel)
	assert.Equal(t, http.StatusOK, wDel.Code)

	// add asset — both tables empty → MAX=0 → TSLP1 is recycled
	wNormal := doAddAsset(t, r, fix.laptopCategoryID, "__TEST__res-recycle-001", fix.plainOriginSlug)
	require.Equal(t, http.StatusCreated, wNormal.Code, wNormal.Body.String())
	var normalResp map[string]any
	require.NoError(t, json.NewDecoder(wNormal.Body).Decode(&normalResp))
	assert.Equal(t, "PYR-TSLP1", normalResp["pyrcode"], "TSLP1 recycled — no items or reservations existed")
}

// TestGetReservations_ReturnsOnlyUnclaimed — reserve 3, claim 1, GET free returns 2.
func TestGetReservations_ReturnsOnlyUnclaimed(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 3)
	require.Len(t, reservations, 3)

	// claim one
	w := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": reservations[0].PyrCode, "serial": "__TEST__get-serial-001"},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// GET free (default)
	wGet := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/assets/reservations?category_id=%d", fix.laptopCategoryID), nil)
	r.ServeHTTP(wGet, req)
	assert.Equal(t, http.StatusOK, wGet.Code)
	var free []PyrCodeReservation
	require.NoError(t, json.NewDecoder(wGet.Body).Decode(&free))
	assert.Len(t, free, 2)

	// GET claimed
	wClaimed := httptest.NewRecorder()
	reqC, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/assets/reservations?category_id=%d&status=claimed", fix.laptopCategoryID), nil)
	r.ServeHTTP(wClaimed, reqC)
	assert.Equal(t, http.StatusOK, wClaimed.Code)
	var claimed []PyrCodeReservation
	require.NoError(t, json.NewDecoder(wClaimed.Body).Decode(&claimed))
	assert.Len(t, claimed, 1)

	// GET all
	wAll := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/assets/reservations?category_id=%d&status=all", fix.laptopCategoryID), nil)
	r.ServeHTTP(wAll, reqA)
	assert.Equal(t, http.StatusOK, wAll.Code)
	var all []PyrCodeReservation
	require.NoError(t, json.NewDecoder(wAll.Body).Decode(&all))
	assert.Len(t, all, 3)
}

// TestDelete_ByPyrCodes — reserve 3, delete 2 by pyr_code, 1 remains.
func TestDelete_ByPyrCodes(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 3)
	require.Len(t, reservations, 3)

	delBody := toJSON(t, map[string]any{"pyr_codes": []string{reservations[0].PyrCode, reservations[1].PyrCode}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/assets/reservations", delBody)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["deleted"])

	var remaining int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM pyr_code_reservations WHERE category_id = $1 AND claimed_at IS NULL", fix.laptopCategoryID).Scan(&remaining))
	assert.Equal(t, 1, remaining)
}

// TestDelete_ByIDs — reserve 3, delete by id.
func TestDelete_ByIDs(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 3)
	require.Len(t, reservations, 3)

	delBody := toJSON(t, map[string]any{"ids": []int{reservations[0].ID, reservations[2].ID}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/assets/reservations", delBody)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["deleted"])
}

// TestDelete_ClaimedReturns409 — trying to delete a claimed reservation yields 409.
func TestDelete_ClaimedReturns409(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 2)
	require.Len(t, reservations, 2)

	// claim first one
	w := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": reservations[0].PyrCode, "serial": "__TEST__del-conflict-001"},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// try to delete claimed code
	delBody := toJSON(t, map[string]any{"pyr_codes": []string{reservations[0].PyrCode}})
	wDel := httptest.NewRecorder()
	reqDel, _ := http.NewRequest(http.MethodDelete, "/assets/reservations", delBody)
	reqDel.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wDel, reqDel)
	assert.Equal(t, http.StatusConflict, wDel.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(wDel.Body).Decode(&resp))
	assert.Contains(t, resp, "claimed_pyr_codes")
}

// TestClaim_BulkAllAtOnce — reserve 3, claim all 3 with serials.
func TestClaim_BulkAllAtOnce(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 3)
	require.Len(t, reservations, 3)

	items := []map[string]any{
		{"pyr_code": reservations[0].PyrCode, "serial": "__TEST__claim-bulk-001"},
		{"pyr_code": reservations[1].PyrCode, "serial": "__TEST__claim-bulk-002"},
		{"pyr_code": reservations[2].PyrCode, "serial": "__TEST__claim-bulk-003"},
	}
	w := doClaim(t, r, fix.plainOriginSlug, 1, items)
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	created, ok := resp["created"].([]any)
	require.True(t, ok)
	assert.Len(t, created, 3)
}

// TestClaim_WithoutSerial — serial: nil is allowed.
func TestClaim_WithoutSerial(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 1)
	require.Len(t, reservations, 1)

	// omit serial field entirely
	items := []map[string]any{{"pyr_code": reservations[0].PyrCode}}
	w := doClaim(t, r, fix.plainOriginSlug, 1, items)
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}

// TestClaim_AlreadyClaimed_Returns400 — claiming an already-claimed code returns 400.
func TestClaim_AlreadyClaimed_Returns400(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 1)
	require.Len(t, reservations, 1)
	pyrCode := reservations[0].PyrCode

	// claim once
	w1 := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": pyrCode, "serial": "__TEST__double-claim-001"},
	})
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

	// claim again — should fail
	w2 := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": pyrCode, "serial": "__TEST__double-claim-002"},
	})
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

// TestClaim_UnknownPyrCode_Returns400 — code not in reservations table.
func TestClaim_UnknownPyrCode_Returns400(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	_ = fix
	w := doClaim(t, r, fix.plainOriginSlug, 1, []map[string]any{
		{"pyr_code": "PYR-TSLP99999", "serial": "__TEST__unknown-001"},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp, "details")
}

// TestClaim_PartialUnknown_RollsBackAll — 2 valid + 1 unknown → nothing created.
func TestClaim_PartialUnknown_RollsBackAll(t *testing.T) {
	db, cleanup := setupResTestDB(t)
	defer cleanup()
	fix := createResFixtures(t, db)
	r := newResRouter(db)

	reservations := doReserve(t, r, fix.laptopCategoryID, 2)
	require.Len(t, reservations, 2)

	items := []map[string]any{
		{"pyr_code": reservations[0].PyrCode, "serial": "__TEST__partial-001"},
		{"pyr_code": reservations[1].PyrCode, "serial": "__TEST__partial-002"},
		{"pyr_code": "PYR-TSLP99998", "serial": "__TEST__partial-003"},
	}
	w := doClaim(t, r, fix.plainOriginSlug, 1, items)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// verify nothing was persisted
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM items WHERE item_serial LIKE '__TEST__partial-%'").Scan(&count))
	assert.Equal(t, 0, count, "transaction must have rolled back")
}
