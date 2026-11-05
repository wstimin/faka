// Package installer serves the first-run wizard before the application opens its database.
package installer

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/shared/passwordpolicy"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

//go:embed setup.html
var setupHTML string

var setupPage = template.Must(template.New("setup").Parse(setupHTML))

type Input struct {
	RedisHost, RedisPassword, AdminUser, AdminPassword, SiteURL string
	RedisPort                                                   int
}

type Options struct {
	Dir, Address string
	Logger       *log.Logger
	// Initialize must create and close a fresh SQLite database at databasePath.
	Initialize func(ctx context.Context, databasePath string, input Input) error
}

// Needed preserves configuration discovery used by legacy installations.
func Needed(dir string) (bool, error) {
	for _, base := range []string{dir, filepath.Join(dir, ".."), filepath.Join(dir, "etc"), filepath.Join(dir, "db")} {
		for _, name := range []string{"config.yml", "config.yaml"} {
			if _, err := os.Stat(filepath.Join(base, name)); err == nil {
				return false, nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return false, err
			}
		}
	}
	if os.Getenv("APP_SECRET_KEY") != "" {
		return false, nil
	}
	// An orphaned database must never become a new administrator's installation.
	if _, err := os.Stat(filepath.Join(dir, "db")); err == nil {
		return false, errors.New("database directory exists without configuration; restore its configuration before starting")
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, nil
}

func secret() (string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func tokenFor(dir string) (string, error) {
	path := filepath.Join(dir, ".install-token")
	if data, err := os.ReadFile(path); err == nil {
		token := strings.TrimSpace(string(data))
		decoded, err := hex.DecodeString(token)
		if err != nil || len(decoded) != 32 {
			return "", errors.New("invalid .install-token file")
		}
		return token, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	token, err := secret()
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return tokenFor(dir)
	}
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(token)
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	return token, closeErr
}

func (in *Input) validate() error {
	in.RedisHost = strings.TrimSpace(in.RedisHost)
	in.AdminUser = strings.TrimSpace(in.AdminUser)
	in.SiteURL = strings.TrimRight(strings.TrimSpace(in.SiteURL), "/")
	for _, value := range []string{in.RedisHost, in.RedisPassword, in.AdminUser, in.AdminPassword, in.SiteURL} {
		if strings.ContainsAny(value, "\r\n\x00") || len(value) > 2048 {
			return errors.New("setup.invalid_input")
		}
	}
	if in.RedisHost == "" || strings.ContainsAny(in.RedisHost, "/\\@?# ") || in.RedisPort < 1 || in.RedisPort > 65535 {
		return errors.New("setup.invalid_redis")
	}
	if in.AdminUser == "" || len(in.AdminUser) > 128 || len(in.AdminPassword) > 72 || in.AdminPassword != strings.TrimSpace(in.AdminPassword) {
		return errors.New("setup.invalid_admin")
	}
	if err := passwordpolicy.Validate(passwordpolicy.Policy{MinLength: 8, RequireUpper: true, RequireLower: true, RequireNumber: true}, in.AdminPassword); err != nil {
		return errors.New("setup.invalid_admin")
	}
	u, err := url.Parse(in.SiteURL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return errors.New("setup.invalid_url")
	}
	return nil
}

func checkRedis(ctx context.Context, in Input) error {
	for _, db := range []int{0, 1} {
		client := redis.NewClient(&redis.Options{Addr: net.JoinHostPort(in.RedisHost, strconv.Itoa(in.RedisPort)), Password: in.RedisPassword, DB: db, MaxRetries: -1, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second})
		err := client.Ping(ctx).Err()
		_ = client.Close()
		if err != nil {
			return errors.New("setup.redis_failed")
		}
	}
	return nil
}

func configuration(in Input, address string) ([]byte, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	appKey, err := secret()
	if err != nil {
		return nil, err
	}
	adminKey, err := secret()
	if err != nil {
		return nil, err
	}
	userKey, err := secret()
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(map[string]interface{}{
		"app":       map[string]interface{}{"secret_key": appKey},
		"server":    map[string]interface{}{"host": host, "port": port, "mode": "release", "trusted_proxies": []string{"127.0.0.1/32", "::1/128"}},
		"database":  map[string]interface{}{"driver": "sqlite", "dsn": "./db/dujiao.db", "pool": map[string]interface{}{"max_open_conns": 1, "max_idle_conns": 1}},
		"jwt":       map[string]interface{}{"secret": adminKey, "expire_hours": 24},
		"user_jwt":  map[string]interface{}{"secret": userKey, "expire_hours": 24, "remember_me_expire_hours": 168},
		"bootstrap": map[string]interface{}{"default_admin_username": in.AdminUser},
		"redis":     map[string]interface{}{"enabled": true, "host": in.RedisHost, "port": in.RedisPort, "password": in.RedisPassword, "db": 0},
		"queue":     map[string]interface{}{"enabled": true, "host": in.RedisHost, "port": in.RedisPort, "password": in.RedisPassword, "db": 1},
		"cors":      map[string]interface{}{"allowed_origins": []string{in.SiteURL}},
		"email":     map[string]interface{}{"enabled": false},
		"web":       map[string]interface{}{"admin_path": "/admin"},
	})
}

type wizard struct {
	options    Options
	token      string
	done       chan struct{}
	mu         sync.Mutex
	complete   bool
	redisCheck func(context.Context, Input) error
}

func (s *wizard) install(ctx context.Context, in Input) error {
	needed, err := Needed(s.options.Dir)
	if err != nil || !needed {
		return errors.New("setup.locked")
	}
	if err := s.redisCheck(ctx, in); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(s.options.Dir, ".install-stage-")
	if err != nil {
		return errors.New("setup.save_failed")
	}
	defer os.RemoveAll(staging)
	data, err := configuration(in, s.options.Address)
	if err != nil {
		return errors.New("setup.save_failed")
	}
	if err := os.WriteFile(filepath.Join(staging, "config.yml"), data, 0600); err != nil {
		return errors.New("setup.save_failed")
	}
	if err := s.options.Initialize(ctx, filepath.Join(staging, "dujiao.db"), in); err != nil {
		s.options.Logger.Printf("installation initialization failed: %v", err)
		return errors.New("setup.database_failed")
	}
	needed, err = Needed(s.options.Dir)
	if err != nil || !needed {
		return errors.New("setup.locked")
	}
	// Publish the closed database and its configuration together, never a half-installation.
	if err := os.Rename(staging, filepath.Join(s.options.Dir, "db")); err != nil {
		return errors.New("setup.save_failed")
	}
	return nil
}

func (s *wizard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nonce, err := secret()
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-"+nonce+"'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	if r.URL.Path != "/" && r.URL.Path != "/install" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	locale := r.URL.Query().Get("lang")
	if locale != i18n.LocaleTW && locale != i18n.LocaleEN {
		locale = i18n.LocaleZH
	}
	translate := func(key string) string { return i18n.T(locale, "setup."+key) }
	data := struct {
		T                                                                     func(string) string
		Locale, Token, Error, SiteURL, RedisHost, RedisPort, AdminUser, Nonce string
		Done                                                                  bool
	}{T: translate, Locale: locale, SiteURL: "http://" + r.Host, RedisHost: "127.0.0.1", RedisPort: "6379", AdminUser: "admin", Nonce: nonce}
	if r.TLS != nil {
		data.SiteURL = "https://" + r.Host
	}
	token := r.URL.Query().Get("token")
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if err := r.ParseForm(); err != nil {
			http.Error(w, translate("invalid_input"), 400)
			return
		}
		token = r.PostForm.Get("token")
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.EqualFold(u.Host, r.Host) {
				http.Error(w, translate("cross_site"), 403)
				return
			}
		}
	}
	valid := len(token) == 64 && subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) == 1
	if valid {
		data.Token = token
	}
	if r.Method == http.MethodPost {
		if !valid {
			w.WriteHeader(403)
			data.Error = translate("invalid_token")
		} else {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.complete {
				http.Error(w, translate("locked"), 409)
				return
			}
			port, _ := strconv.Atoi(r.PostForm.Get("redis_port"))
			in := Input{RedisHost: r.PostForm.Get("redis_host"), RedisPort: port, RedisPassword: r.PostForm.Get("redis_password"), AdminUser: r.PostForm.Get("admin_user"), AdminPassword: r.PostForm.Get("admin_password"), SiteURL: r.PostForm.Get("site_url")}
			err := in.validate()
			if err == nil && in.AdminPassword != r.PostForm.Get("confirm_password") {
				err = errors.New("setup.password_mismatch")
			}
			if err == nil {
				ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
				err = s.install(ctx, in)
				cancel()
			}
			data.RedisHost, data.RedisPort, data.AdminUser, data.SiteURL = in.RedisHost, strconv.Itoa(in.RedisPort), in.AdminUser, in.SiteURL
			if err != nil {
				data.Error = i18n.T(locale, err.Error())
			} else {
				s.complete, data.Done = true, true
				close(s.done)
			}
		}
	}
	_ = setupPage.Execute(w, data)
}

// Run returns only after a new installation has committed, or immediately for existing sites.
func Run(ctx context.Context, options Options) error {
	if options.Dir == "" {
		var err error
		options.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	needed, err := Needed(options.Dir)
	if err != nil || !needed {
		return err
	}
	if options.Logger == nil {
		options.Logger = log.Default()
	}
	if options.Initialize == nil {
		return errors.New("installer initializer is required")
	}
	if options.Address == "" {
		host, port := os.Getenv("SERVER_HOST"), os.Getenv("SERVER_PORT")
		if host == "" {
			host = "0.0.0.0"
		}
		if port == "" {
			port = "8081"
		}
		options.Address = net.JoinHostPort(host, port)
	}
	token, err := tokenFor(options.Dir)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", options.Address)
	if err != nil {
		return err
	}
	_, port, _ := net.SplitHostPort(options.Address)
	options.Logger.Printf("Dujiao Next setup: http://SERVER_IP:%s/install?token=%s", port, token)
	s := &wizard{options: options, token: token, done: make(chan struct{}), redisCheck: checkRedis}
	server := &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 130 * time.Second, IdleTimeout: 30 * time.Second}
	failures := make(chan error, 1)
	go func() { failures <- server.Serve(listener) }()
	select {
	case <-ctx.Done():
		_ = server.Close()
		return ctx.Err()
	case err := <-failures:
		return fmt.Errorf("installer server stopped: %w", err)
	case <-s.done:
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(options.Dir, ".install-token"))
		return nil
	}
}
