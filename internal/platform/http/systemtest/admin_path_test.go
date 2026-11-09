package systemtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/config"
	systemhttp "github.com/dujiao-next/internal/platform/http/system"
	"github.com/gin-gonic/gin"
)

func TestAdminPathSaveAndPendingState(t *testing.T) {
	t.Setenv("WEB_ADMIN_PATH", "")
	gin.SetMode(gin.TestMode)
	filename := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(filename, []byte("web:\n  admin_path: /admin\njwt:\n  secret: hidden\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{FilePath: filename, Web: config.WebConfig{AdminPath: "/admin"}}
	router := gin.New()
	authorized := router.Group("/api/v1/admin", func(c *gin.Context) { c.Set("admin_id", uint(1)); c.Next() })
	systemhttp.RegisterAdminRoutes(authorized, systemhttp.NewAdminHandler(nil, cfg))
	request := func(method, body string) map[string]interface{} {
		t.Helper()
		req := httptest.NewRequest(method, "/api/v1/admin/system/admin-path", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		var payload map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	for _, invalid := range []string{"/", "/api/private", "/assets", "/products/secret", "/me", "/orders/private", "/terms", "/privacy", "/.well-known", "/bad/../path", "/new/", "//external.example", "/new?x=1", "/" + strings.Repeat("a", 201)} {
		body, _ := json.Marshal(map[string]string{"path": invalid})
		if payload := request(http.MethodPut, string(body)); payload["status_code"] != float64(400) {
			t.Fatalf("unsafe path accepted: %q: %+v", invalid, payload)
		}
	}
	t.Setenv("WEB_ADMIN_PATH", "/fixed")
	if payload := request(http.MethodPut, `{"path":"/private-console"}`); payload["status_code"] != float64(400) {
		t.Fatal("environment override must prevent misleading save")
	}
	t.Setenv("WEB_ADMIN_PATH", "")
	if payload := request(http.MethodPut, `{"path":"/private-console"}`); payload["status_code"] != float64(0) {
		t.Fatalf("save failed: %+v", payload)
	}
	if cfg.Web.AdminPath != "/admin" {
		t.Fatal("running routing config must not change before restart")
	}
	payload := request(http.MethodGet, "")
	data := payload["data"].(map[string]interface{})
	if data["current_path"] != "/admin" || data["saved_path"] != "/private-console" || data["restart_required"] != true {
		t.Fatalf("wrong pending state: %+v", data)
	}
	if _, leaked := data["jwt"]; leaked {
		t.Fatal("must not expose config secrets")
	}
	if payload := request(http.MethodPut, `{"path":"/admin"}`); payload["status_code"] != float64(0) {
		t.Fatal("must be able to cancel a pending change")
	}
	data = request(http.MethodGet, "")["data"].(map[string]interface{})
	if data["restart_required"] != false {
		t.Fatal("restoring the running path must cancel the pending change")
	}
	request(http.MethodPut, `{"path":"/private-console"}`)
	cfg.Web.AdminPath = "/private-console" // Simulate the next process loading the saved config.
	data = request(http.MethodGet, "")["data"].(map[string]interface{})
	if data["restart_required"] != false {
		t.Fatal("restart state must clear in the new process")
	}
	cfg.FilePath = filepath.Join(t.TempDir(), "missing.yml")
	if payload := request(http.MethodPut, `{"path":"/private-console"}`); payload["status_code"] != float64(500) {
		t.Fatal("missing config must fail without creating a replacement")
	}
}
