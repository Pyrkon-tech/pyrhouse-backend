package routes

import (
	"database/sql"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"warehouse/internal/config"
	"warehouse/internal/di"
)

// Routes deliberately left out of the OpenAPI document.
var undocumentedRoutes = map[string]string{}

var ginParam = regexp.MustCompile(`:[^/]+|\*[^/]+`)
var specParam = regexp.MustCompile(`\{[^}]+\}`)

func normalizeRoute(method, path string) string {
	path = ginParam.ReplaceAllString(path, "{}")
	path = specParam.ReplaceAllString(path, "{}")
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return method + " " + path
}

// TestOpenAPICoversAllRoutes keeps docs/openapi.yaml (the contract the frontend types are generated from) in
// step with the routes Gin actually serves: every route must be documented and every documented operation
// must exist.
func TestOpenAPICoversAllRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// sql.Open does not connect; constructors only keep the handle.
	db, err := sql.Open("postgres", "postgres://localhost:1/none?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		// OAuth handlers register routes only when configured.
		Discord: config.DiscordConfig{ClientID: "x", ClientSecret: "x"},
		Google:  config.GoogleConfig{ClientID: "x", ClientSecret: "x"},
	}
	router := gin.New()
	container := di.NewAppContainer(db, cfg)
	RegisterPublicRoutes(router, container)
	RegisterProtectedRoutes(router, container)
	RegisterUtilityRoutes(router)

	served := map[string]bool{}
	for _, r := range router.Routes() {
		served[normalizeRoute(r.Method, r.Path)] = true
	}

	raw, err := os.ReadFile("../../docs/openapi.yaml")
	require.NoError(t, err)
	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &spec))

	documented := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			switch method {
			case "get", "post", "put", "patch", "delete":
				documented[normalizeRoute(strings.ToUpper(method), path)] = true
			}
		}
	}

	var missing, stale []string
	for r := range served {
		if !documented[r] && undocumentedRoutes[r] == "" {
			missing = append(missing, r)
		}
	}
	for r := range documented {
		if !served[r] {
			stale = append(stale, r)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	assert.Empty(t, missing, "routes served but not in docs/openapi.yaml")
	assert.Empty(t, stale, "operations in docs/openapi.yaml that no route serves")
}
