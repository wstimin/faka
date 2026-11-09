package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var adminPathFileMu sync.Mutex

// ReadAdminPath reads the saved value without changing the running server's routes.
func ReadAdminPath(filename string) (string, error) {
	adminPathFileMu.Lock()
	defer adminPathFileMu.Unlock()
	if filename == "" {
		return "", errors.New("no active config file")
	}
	reader := viper.New()
	reader.SetConfigFile(filename)
	reader.SetConfigType("yaml")
	reader.SetDefault("web.admin_path", "/admin")
	if err := reader.ReadInConfig(); err != nil {
		return "", err
	}
	return reader.GetString("web.admin_path"), nil
}

// WriteAdminPath patches only web.admin_path. The caller must validate the path.
// A temp file in the same directory keeps the existing config intact until rename.
func WriteAdminPath(filename, adminPath string) error {
	adminPathFileMu.Lock()
	defer adminPathFileMu.Unlock()
	resolved, document, err := readAdminPathDocument(filename)
	if err != nil {
		return err
	}
	root := document.Content[0]
	web, err := configMappingValue(root, "web")
	if err != nil {
		return err
	}
	if web == nil {
		web = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "web"}, web)
	}
	entry, err := configMappingValue(web, "admin_path")
	if err != nil {
		return err
	}
	if entry == nil {
		entry = &yaml.Node{}
		web.Content = append(web.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "admin_path"}, entry)
	}
	entry.Kind, entry.Tag, entry.Value, entry.Style = yaml.ScalarNode, "!!str", adminPath, yaml.DoubleQuotedStyle
	entry.Content, entry.Alias = nil, nil
	data, err := yaml.Marshal(document)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(resolved), ".admin-path-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), resolved)
}

func readAdminPathDocument(filename string) (string, *yaml.Node, error) {
	if filename == "" {
		return "", nil, errors.New("no active config file")
	}
	resolved, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", nil, err
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err = decoder.Decode(&document); err != nil {
		return "", nil, err
	}
	var extra yaml.Node
	if err = decoder.Decode(&extra); err != io.EOF {
		return "", nil, errors.New("config must contain exactly one YAML document")
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return "", nil, errors.New("config must be a YAML mapping")
	}
	// Decode also rejects duplicate keys before touching a config containing secrets.
	var values map[string]interface{}
	if err = document.Decode(&values); err != nil {
		return "", nil, err
	}
	return resolved, &document, nil
}

func configMappingValue(mapping *yaml.Node, key string) (*yaml.Node, error) {
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s parent must be a mapping", key)
	}
	var found *yaml.Node
	for i := 0; i < len(mapping.Content); i += 2 {
		if strings.EqualFold(mapping.Content[i].Value, key) {
			if found != nil {
				return nil, fmt.Errorf("duplicate config key %s", key)
			}
			found = mapping.Content[i+1]
		}
	}
	return found, nil
}
