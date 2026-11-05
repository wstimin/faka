package installer

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/i18n"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func installationForm() url.Values {
	return url.Values{"token": {strings.Repeat("a", 64)}, "redis_host": {"127.0.0.1"}, "redis_port": {"6379"}, "redis_password": {`redis:# '"\ secret`}, "admin_user": {"owner"}, "admin_password": {"StrongPassword99"}, "confirm_password": {"StrongPassword99"}, "site_url": {"https://shop.example.com/"}}
}

func newWizard(t *testing.T) *wizard {
	t.Helper()
	return &wizard{token: strings.Repeat("a", 64), done: make(chan struct{}), redisCheck: func(context.Context, Input) error { return nil }, options: Options{Dir: t.TempDir(), Address: "0.0.0.0:8081", Logger: log.New(io.Discard, "", 0), Initialize: func(_ context.Context, path string, _ Input) error {
		return os.WriteFile(path, []byte("test database"), 0600)
	}}}
}

func submit(s *wizard, form url.Values, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "http://shop.example.com/install", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

func TestWizardProtectsInstallationAndCommitsOnce(t *testing.T) {
	s := newWizard(t)
	calls := 0
	s.options.Initialize = func(_ context.Context, path string, in Input) error {
		calls++
		if in.SiteURL != "https://shop.example.com" || in.AdminPassword != "StrongPassword99" {
			t.Fatal("input lost")
		}
		return os.WriteFile(path, []byte("closed database"), 0600)
	}
	get := httptest.NewRecorder()
	s.ServeHTTP(get, httptest.NewRequest("GET", "http://shop.example.com/install", nil))
	if strings.Contains(get.Body.String(), s.token) || get.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatal("unsafe installer response")
	}
	if !strings.Contains(get.Body.String(), "安装向导") {
		t.Fatal(get.Body.String())
	}
	for _, scenario := range []string{"wrong token", "cross site", "null origin", "weak password", "mismatch", "invalid URL", "invalid port", "oversized", "newline"} {
		t.Run(scenario, func(t *testing.T) {
			form := installationForm()
			origin := "http://shop.example.com"
			switch scenario {
			case "wrong token":
				form.Set("token", "wrong")
			case "cross site":
				origin = "https://evil.example.com"
			case "null origin":
				origin = "null"
			case "weak password":
				form.Set("admin_password", "admin123")
			case "mismatch":
				form.Set("confirm_password", "DifferentPassword99")
			case "invalid URL":
				form.Set("site_url", "https://user:pass@example.com/path")
			case "invalid port":
				form.Set("redis_port", "0")
			case "oversized":
				form.Set("redis_password", strings.Repeat("z", 20000))
			case "newline":
				form.Set("admin_user", "owner\nadmin")
			}
			w := submit(s, form, origin)
			if calls != 0 || s.complete {
				t.Fatal("invalid submission reached initialization")
			}
			if scenario == "cross site" || scenario == "null origin" || scenario == "wrong token" {
				if w.Code != 403 {
					t.Fatal(w.Code)
				}
			}
			if strings.Contains(w.Body.String(), "StrongPassword99") || strings.Contains(w.Body.String(), form.Get("redis_password")) && form.Get("redis_password") != "" {
				t.Fatal("password reflected")
			}
		})
	}
	w := submit(s, installationForm(), "http://shop.example.com")
	if w.Code != 200 || calls != 1 || !s.complete || !strings.Contains(w.Body.String(), "安装完成") || !strings.Contains(w.Body.String(), "waitForStore") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(s.options.Dir, "db", "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "StrongPassword99") {
		t.Fatal("plaintext administrator password persisted")
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	redis := config["redis"].(map[string]interface{})
	queue := config["queue"].(map[string]interface{})
	if redis["password"] != installationForm().Get("redis_password") || queue["password"] != redis["password"] || redis["db"] != 0 || queue["db"] != 1 {
		t.Fatal("Redis configuration changed")
	}
	keys := []string{config["app"].(map[string]interface{})["secret_key"].(string), config["jwt"].(map[string]interface{})["secret"].(string), config["user_jwt"].(map[string]interface{})["secret"].(string)}
	if len(keys[0]) != 64 || keys[0] == keys[1] || keys[0] == keys[2] || keys[1] == keys[2] {
		t.Fatal("keys not independently generated")
	}
	if needed, err := Needed(s.options.Dir); err != nil || needed {
		t.Fatal("committed installation not locked", err)
	}
	if w := submit(s, installationForm(), "http://shop.example.com"); w.Code != 409 || calls != 1 {
		t.Fatal("reinstallation allowed")
	}
	t.Chdir(s.options.Dir)
	viper.Reset()
	t.Cleanup(viper.Reset)
	cfg := appconfig.Load()
	if cfg.Server.Port != "8081" || cfg.Database.Driver != "sqlite" || cfg.Redis.Password != installationForm().Get("redis_password") || cfg.Queue.DB != 1 || cfg.Bootstrap.DefaultAdminPassword != "" || cfg.App.SecretKey != keys[0] {
		t.Fatal("normal application cannot load the committed installation")
	}
}

func TestFailedInitializationLeavesNoInstallationAndCanRetry(t *testing.T) {
	for _, scenario := range []string{"Redis unavailable", "database unavailable"} {
		t.Run(scenario, func(t *testing.T) {
			s := newWizard(t)
			if scenario == "Redis unavailable" {
				s.redisCheck = func(context.Context, Input) error { return errors.New("setup.redis_failed") }
			} else {
				s.options.Initialize = func(_ context.Context, path string, _ Input) error {
					_ = os.WriteFile(path, []byte("partial"), 0600)
					return errors.New("migration failed")
				}
			}
			w := submit(s, installationForm(), "http://shop.example.com")
			if s.complete || !strings.Contains(w.Body.String(), "失败") {
				t.Fatal(w.Body.String())
			}
			if needed, err := Needed(s.options.Dir); err != nil || !needed {
				t.Fatal("failure left a partial installation", err)
			}
			entries, _ := os.ReadDir(s.options.Dir)
			if len(entries) != 0 {
				t.Fatal("staging data left after failure", entries)
			}
			s.redisCheck = func(context.Context, Input) error { return nil }
			s.options.Initialize = func(_ context.Context, path string, _ Input) error { return os.WriteFile(path, []byte("ready"), 0600) }
			if w := submit(s, installationForm(), "http://shop.example.com"); !s.complete {
				t.Fatal(w.Body.String())
			}
		})
	}
}

func TestExistingSitesAndAuthorizationArePreserved(t *testing.T) {
	t.Setenv("APP_SECRET_KEY", "")
	for _, path := range []string{"config.yml", "config.yaml", "etc/config.yml", "db/config.yml"} {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, path)
			_ = os.MkdirAll(filepath.Dir(dest), 0700)
			_ = os.WriteFile(dest, []byte("existing"), 0600)
			if needed, err := Needed(dir); err != nil || needed {
				t.Fatal("existing site exposed setup", err)
			}
		})
	}
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, "db"), 0700)
	if _, err := Needed(dir); err == nil {
		t.Fatal("orphan database allowed installation")
	}
	dir = t.TempDir()
	first, err := tokenFor(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := tokenFor(dir)
	if err != nil || first != second {
		t.Fatal("token changed after restart")
	}
}

func TestInstallerTranslationsAndEscaping(t *testing.T) {
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleTW, i18n.LocaleEN} {
		s := newWizard(t)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "http://shop.example.com/install?lang="+locale, nil))
		if strings.Contains(w.Body.String(), "setup.") || !strings.Contains(w.Body.String(), i18n.T(locale, "setup.title")) {
			t.Fatal("missing translation", locale, w.Body.String())
		}
	}
	s := newWizard(t)
	form := installationForm()
	form.Set("admin_user", `<script>alert(1)</script>`)
	form.Set("confirm_password", "mismatch")
	w := submit(s, form, "http://shop.example.com")
	if strings.Contains(w.Body.String(), `value="<script>`) || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("input not escaped")
	}
}
