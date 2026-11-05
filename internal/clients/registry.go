// Package clients holds the declarative client registry: which MCP clients
// exist, how to detect them, where their config lives, and which format
// handler edits it. Adding a client of a known format is a registry entry,
// not code (the Docker MCP Gateway pattern).
package clients

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed registry.yml
var registryYAML []byte

// Formats the M1 editors understand. codex-toml is print-only for now
// (a TOML editor is a dependency decision, not a rushed one).
const (
	FormatMCPJSON   = "mcp-json"
	FormatCodexTOML = "codex-toml"
)

type ClientSpec struct {
	Key          string   `yaml:"key"`
	DisplayName  string   `yaml:"displayName"`
	Format       string   `yaml:"format"`
	Writable     bool     `yaml:"writable"`
	InstallCheck []string `yaml:"installCheck"`
	ConfigPaths  []string `yaml:"configPaths"`
}

type registryFile struct {
	Clients []ClientSpec `yaml:"clients"`
}

// Load returns the embedded registry.
func Load() ([]ClientSpec, error) {
	var file registryFile
	if err := yaml.Unmarshal(registryYAML, &file); err != nil {
		return nil, fmt.Errorf("parse client registry: %w", err)
	}
	return file.Clients, nil
}

// Find returns the spec for a client key.
func Find(key string) (ClientSpec, error) {
	specs, err := Load()
	if err != nil {
		return ClientSpec{}, err
	}
	for _, spec := range specs {
		if spec.Key == key {
			return spec, nil
		}
	}
	return ClientSpec{}, fmt.Errorf("unknown client %q (registry knows: %s)", key, knownKeys(specs))
}

// DetectInstalled reports which registry clients look installed (any
// install-check path exists, ~ expanded per platform).
func DetectInstalled() ([]ClientSpec, error) {
	specs, err := Load()
	if err != nil {
		return nil, err
	}
	installed := []ClientSpec{}
	for _, spec := range specs {
		if len(spec.InstallCheck) == 0 || pathExistsAny(expandAll(spec.InstallCheck)) {
			installed = append(installed, spec)
		}
	}
	return installed, nil
}

// ExistingConfigPath returns the first config path that exists for the
// client on this platform, or "" when none does.
func (spec ClientSpec) ExistingConfigPath() string {
	for _, path := range expandAll(spec.ConfigPaths) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func expandAll(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, expandHome(path))
	}
	return out
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
		}
	}
	return strings.ReplaceAll(path, "${GOOS}", runtime.GOOS)
}

func pathExistsAny(paths []string) bool {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func knownKeys(specs []ClientSpec) string {
	keys := make([]string, len(specs))
	for i, spec := range specs {
		keys[i] = spec.Key
	}
	return strings.Join(keys, ", ")
}
