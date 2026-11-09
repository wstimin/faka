package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriteAdminPathPreservesOtherConfig(t *testing.T) {
	for _, original := range []string{
		"# keep this comment\njwt:\n  secret: '0123: # secret'\nweb:\n  admin_path: /admin\n  other: true\nredis:\n  password: 'null'\n  port: 6379\n",
		"jwt:\n  secret: private\n",
		"WEB:\n  ADMIN_PATH: /admin\n  other: true\n",
	} {
		filename := filepath.Join(t.TempDir(), "config.yml")
		if err := os.WriteFile(filename, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		var before map[string]interface{}
		if err := yaml.Unmarshal([]byte(original), &before); err != nil {
			t.Fatal(err)
		}
		if err := WriteAdminPath(filename, "/private/console-7"); err != nil {
			t.Fatal(err)
		}
		saved, err := ReadAdminPath(filename)
		if err != nil || saved != "/private/console-7" {
			t.Fatalf("saved path = %q, %v", saved, err)
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		var after map[string]interface{}
		if err := yaml.Unmarshal(data, &after); err != nil {
			t.Fatal(err)
		}
		for _, values := range []map[string]interface{}{before, after} {
			for key, value := range values {
				if strings.EqualFold(key, "web") {
					web := value.(map[string]interface{})
					for field := range web {
						if strings.EqualFold(field, "admin_path") {
							delete(web, field)
						}
					}
					if len(web) == 0 {
						delete(values, key)
					}
				}
			}
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatal("non-path config changed")
		}
		if strings.HasPrefix(original, "#") && !strings.Contains(string(data), "# keep this comment") {
			t.Fatal("lost config comment")
		}
		remaining, _ := filepath.Glob(filepath.Join(filepath.Dir(filename), ".admin-path-*.tmp"))
		if len(remaining) != 0 {
			t.Fatal("left temporary config files")
		}
	}
}

func TestWriteAdminPathRejectsMalformedConfigWithoutChangingIt(t *testing.T) {
	for _, original := range []string{"web: [broken", "web: invalid\n", "web:\n  admin_path: /a\n  admin_path: /b\n", "- list\n", "web:\n  admin_path: /admin\n---\njwt:\n  secret: hidden\n"} {
		filename := filepath.Join(t.TempDir(), "config.yml")
		if err := os.WriteFile(filename, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		if err := WriteAdminPath(filename, "/new-console"); err == nil {
			t.Fatal("expected invalid config to be rejected")
		}
		data, _ := os.ReadFile(filename)
		if string(data) != original {
			t.Fatal("invalid config was modified")
		}
	}
	if err := WriteAdminPath("", "/new-console"); err == nil {
		t.Fatal("must not create a config with missing secrets")
	}
}
