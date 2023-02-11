package core

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/resync"
	"github.com/pelletier/go-toml/v2"
)

// How many parent directories to traverse before considering a directory as not a nt repository
const maxDepth = 10

// Default .nt/config content
const DefaultConfig = `
[core]
extensions=["md", "markdown"]

[search.quotes]
q="-#ignore @kind:quote"
name="Favorite Quotes"
`

// Default .ntignore content
const DefaultIgnore = `
build/
README.md
`

var (
	// Lazy-load configuration and ensure a single read
	configOnce      resync.Once
	configSingleton *Config
)

// Note: Fields must be public for toml package to unmarshall
type ConfigFile struct {
	Core struct {
		Extensions []string
	}
	Search map[string]struct {
		Q    string
		Name string
	}
}

// SupportExtension checks if the given file extension must be considered.
func (f *ConfigFile) SupportExtension(path string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".") // ".md" => "md"
	for _, extension := range f.Core.Extensions {
		if strings.EqualFold(extension, ext) { // case-insensitive
			return true
		}
	}
	return false
}

type IgnoreFile struct {
	Entries []GlobPath
}

func (f *IgnoreFile) Include(path string) bool {
	for _, entry := range f.Entries {
		if entry.Match(path) {
			log.Printf("Match %s: %s", entry, path) // FIXME remove or debug
			return false
		}
	}
	return true
}

type GlobPath string

func (g GlobPath) Match(path string) bool {
	// TODO Go standard library doesn't support the same Git syntax (ex: ** is missing).
	// Compare https://git-scm.com/docs/gitignore with https://go.dev/src/path/filepath/match.go
	match, err := filepath.Match(string(g), path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid glob pattern %q: %v\n", string(g), err)
		os.Exit(1)
	}
	return match
}

type VerboseLevel int

const (
	VerboseOff VerboseLevel = iota
	VerboseInfo
	VerboseDebug
	VerboseTrace
)

type Config struct {
	// Absolute top directory containing the .nt sub-directory
	RootDirectory string

	// .nt/config content
	ConfigFile ConfigFile

	// .ntignore content
	IgnoreFile IgnoreFile

	// Logs verbosity
	Verbose VerboseLevel
}

func CurrentConfig() *Config {
	configOnce.Do(func() {
		var err error
		configSingleton, err = ReadConfigFromDirectory(currentHome())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to read current configuration: %v\n", err)
			os.Exit(1)
		}
		if configSingleton == nil {
			fmt.Fprintln(os.Stderr, "fatal: not a NoteTaker repository (or any of the parent directories): .nt")
			os.Exit(1)
		}
	})
	return configSingleton
}

// SetVerboseLevel overrides the default verbose level
func (c *Config) SetVerboseLevel(level VerboseLevel) *Config {
	c.Verbose = level
	return c
}

func (c *Config) Info() bool {
	return c.Verbose >= VerboseInfo
}

func (c *Config) Debug() bool {
	return c.Verbose >= VerboseDebug
}

func (c *Config) Trace() bool {
	return c.Verbose >= VerboseTrace
}

func currentHome() string {
	// Supports overriding the root directory mainly for testing purposes.
	// For example, when developing the CLI, it's convenient to try command
	// without installing the binary. Ex:
	//
	//   $ env NT_HOME ./examples go run main.go build
	if path, ok := os.LookupEnv("NT_HOME"); ok {
		abspath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to evaluate $NT_HOME")
			os.Exit(1)
		}
		if _, err := os.Stat(abspath); os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Path in $NT_HOME undefined")
			os.Exit(1)
		}
		return abspath
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to determine current directory: %v\n", err)
		os.Exit(1)
	}
	return cwd
}

// ReadConfigFromDirectory loads the configuration by searching for a .nt directory in the given directory
// or any parent directories. It fails if a directory already exists.
func ReadConfigFromDirectory(path string) (*Config, error) {
	rootPath := path
	i := 0 // Safeguard to not go up too far
	for {
		i++
		if i > maxDepth {
			return nil, nil
		}
		ntPath := filepath.Join(rootPath, ".nt")
		_, err := os.Stat(ntPath)
		if os.IsNotExist(err) {
			if len(strings.Split(rootPath, string(os.PathSeparator))) <= 2 {
				// Root directory detected
				return nil, nil
			}
			rootPath = filepath.Clean(filepath.Join(rootPath, ".."))
		} else if err != nil {
			return nil, fmt.Errorf("error while searching for configuration directory: %v", err)
		} else {
			break
		}
	}

	// Check for .nt/config
	ntConfigPath := filepath.Join(rootPath, ".nt", "config")
	_, err := os.Stat(ntConfigPath)
	var configFile *ConfigFile
	if os.IsNotExist(err) {
		configFile, err = parseConfigFile(DefaultConfig)
		if err != nil {
			return nil, fmt.Errorf("default configuration is broken: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check for .nt/config file: %v", err)
	} else {
		content, err := os.ReadFile(ntConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read .nt/config file: %v", err)
		}
		configFile, err = parseConfigFile(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse .nt/config file: %v", err)
		}
	}

	// Check for .ntignore
	ntignorePath := filepath.Join(rootPath, ".ntignore")
	_, err = os.Stat(ntignorePath)
	var ignoreFile *IgnoreFile
	if os.IsNotExist(err) {
		ignoreFile, err = parseIgnoreFile(DefaultIgnore)
		if err != nil {
			return nil, fmt.Errorf("default configuration is broken: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check for .ntignore file: %v", err)
	} else {
		content, err := os.ReadFile(ntignorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read .ntignore file: %v", err)
		}
		ignoreFile, err = parseIgnoreFile(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse .ntignore file: %v", err)
		}
	}

	return &Config{
		RootDirectory: rootPath,
		ConfigFile:    *configFile,
		IgnoreFile:    *ignoreFile,
		Verbose:       VerboseOff,
	}, nil
}

func parseConfigFile(content string) (*ConfigFile, error) {
	r := strings.NewReader(content)
	d := toml.NewDecoder(r)
	d.DisallowUnknownFields()
	var result ConfigFile
	err := d.Decode(&result)
	return &result, err
}

func parseIgnoreFile(content string) (*IgnoreFile, error) {
	var result IgnoreFile
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			// ignore comment
			continue
		}
		var entry = GlobPath(line)
		result.Entries = append(result.Entries, entry)
	}
	return &result, nil
}

// InitConfigFromDirectory creates the .nt configuration directory with default files including .ntignore.
func InitConfigFromDirectory(path string) (*Config, error) {
	currentConfig, err := ReadConfigFromDirectory(path)
	if err != nil {
		return nil, err
	}
	if currentConfig != nil {
		// Do not override current configuration
		return nil, fmt.Errorf("current configuration detected")
	}

	// Create .nt directory
	ntPath := filepath.Join(path, ".nt")
	err = os.Mkdir(ntPath, 0755)
	if err != nil {
		return nil, err
	}

	// Init .nt/config file
	ntConfigPath := filepath.Join(ntPath, "config")
	err = os.WriteFile(ntConfigPath, []byte(DefaultConfig), 0644)
	if err != nil {
		return nil, err
	}

	// Init .ntignore file
	ntIgnorePath := filepath.Join(path, ".ntignore")
	_, err = os.Stat(ntIgnorePath)
	if os.IsNotExist(err) { // Do not override existing file!
		err = os.WriteFile(ntIgnorePath, []byte(DefaultIgnore), 0644)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Reread configuration
	return ReadConfigFromDirectory(path)
}
