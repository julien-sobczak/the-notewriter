package core

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/medias"
	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// How many parent directories to traverse before considering a directory as not a nt repository
const maxDepth = 10

// SRS
const (
	DefaultSRSBoostFactor = 100
	DefaultSRSAlgorithm   = "Anki2"
	DefaultSRSEaseFactor  = 2.5
)

// Default .nt/config content
const DefaultConfig = `
[core]
extensions=["md", "markdown"]

[medias]
command="ffmpeg"
parallel=1
preset="medium"

[search.quotes]
q="-#ignore @kind:quote"
name="Favorite Quotes"
`

// Default .nt/.gitignore content
const DefaultGitIgnore = `
/database.db
/database.db-journal
/objects/
/index
/refs/
`

// Default .ntignore content
const DefaultIgnore = `
build/
README.md
`

const DefaultLint = `
# No rules by default

schemas:

  - name: Hooks
    attributes:
    - name: hook
      type: string[]
      inherit: false

  - name: Tags
    attributes:
      - name: tags
        type: string[]

  - name: Relations
    attributes:
      - name: source
        inherit: false
      - name: references
        type: string[]
      - name: inspirations
        type: string[]
`

// Edit website/docs/guides/linter.md after for updating this list

var (
	// Lazy-load configuration and ensure a single read
	configOnce      resync.Once
	configSingleton *Config

	converterOnce      resync.Once
	converterSingleton medias.Converter
)

// Note: Fields must be public for toml package to unmarshall
type ConfigFile struct {
	Core      ConfigCore
	Medias    ConfigMedias
	Remote    ConfigRemote
	Deck      map[string]*ConfigDeck
	Search    map[string]*ConfigSearch
	Reference map[string]*ConfigReference
}
type ConfigCore struct {
	Extensions []string
}
type ConfigMedias struct {
	Command  string
	Parallel int
	Preset   string
}
type ConfigRemote struct {
	Type string // fs or s3
	// fs-specific attributes
	Dir string
	// s3-specific attributes
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Secure     bool
	// Storj-specific attributes
	AccessGrant string
	// + reuse BucketName
}
type ConfigDeck struct {
	Name  string
	Query string
	// General attributes
	BoostFactor         int // How passionate I am on this topic (100 = neutral, 80 = challenging, 120 = smooth)
	NewFlashcardsPerDay int // How many new flashcards to add every day (= 0 no more cards for now)
	MaxFlashcardsPerDay int // How many flashcards (including new) to review every day (= 0 no limit, review what is due)
	// Specific attributes
	Algorithm         string         // Anki2
	AlgorithmSettings map[string]any // SRS-specific attributes
}
type ConfigSearch struct {
	Q    string
	Name string
}
type ConfigReference struct {
	Title    string // Ex: "A book"
	Manager  string // Ex: "zotero"
	Path     string // Ex: "references/books"
	Template string // Ex: "# {{.Title}}\n"
}

// SetParallel overrides the value in config file.
func (c *Config) SetParallel(value int) {
	c.ConfigFile.Medias.Parallel = value
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

// ConfigureFSRemote defines a local remote using the file system.
func (f *ConfigFile) ConfigureFSRemote(dir string) *ConfigFile {
	f.Remote = ConfigRemote{
		Type: "fs",
		Dir:  dir,
	}
	return f
}

// ConfigureS3Remote defines a remote using a S3 backend.
func (f *ConfigFile) ConfigureS3Remote(bucketName, accessKey, secretKey string) *ConfigFile {
	f.Remote = ConfigRemote{
		Type:       "s3",
		BucketName: bucketName,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
	}
	return f
}

type IgnoreFile struct {
	Entries PathSpecs
}

func (i *IgnoreFile) MustExcludeFile(path string, dir bool) bool {
	path = strings.Trim(path, "/")
	if dir {
		path += "/"
	}
	return i.Entries.Match(path)
}

// A pathspec is a pattern used to limit paths in "nt" commands ("nt add", "nt diff", etc.)
// and thus limit the scope of operations to some subset of the tree or worktree.
// Pathspecs are used in .ntignore and .nt/lint files and can be prefixed by !.
type PathSpec string

func (p PathSpec) Negate() bool {
	return strings.HasPrefix(string(p), "!")
}

func (p PathSpec) Expr() string {
	return strings.TrimPrefix(string(p), "!")
}

// Match tests a given path. NB: Directories must have a trailing /.
func (p PathSpec) Match(path string) bool {
	// The Go standard library doesn't support the same Git syntax (ex: ** is missing).
	// Compare https://git-scm.com/docs/gitignore with https://go.dev/src/path/filepath/match.go
	// We fallback to a custom implementation.

	if runtime.GOOS == "windows" {
		path = filepath.ToSlash(path)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	expr := p.Expr()
	leadingSlash := strings.HasPrefix(expr, "/")
	trailingSlash := strings.HasSuffix(expr, "/")
	// Adapt slightly the expression to have a correct regex (ex: "projects/" => `/projects/.*?` to match "projects/index.md" but not "myprojects/"")
	if !leadingSlash {
		expr = "/" + expr
	}
	if trailingSlash {
		expr = expr + "**/"
	}

	parts := strings.Split(expr, "**/")
	var partsPatterns []string
	for _, part := range parts {
		subparts := strings.Split(part, "*")
		partsPatterns = append(partsPatterns, strings.Join(subparts, "[^/]*?")) // * => [^/]*
	}
	pattern := strings.Join(partsPatterns, ".*?") // ** => .*?

	if leadingSlash {
		pattern = "^" + pattern
	}

	rePattern, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid glob pattern %q: %v\n", p, err)
		os.Exit(1)
	}

	return rePattern.MatchString(path)
}

type PathSpecs []PathSpec

// AnyPath is a special pathspec that matches any path.
// Useful when no path spec is specified (ex: `nt add`).
var AnyPath = PathSpecs{"."}

// MatchAll tests if the path specs match any path.
func (p PathSpecs) MatchAll() bool {
	anyClause := false
	atLeastOneNegateClause := false
	for _, pathSpec := range p {
		if pathSpec == "." {
			anyClause = true
		}
		if pathSpec.Negate() {
			atLeastOneNegateClause = true
		}
	}
	return anyClause && !atLeastOneNegateClause
}

// Match tests if a file path satisfies the conditions.
func (p PathSpecs) Match(path string) bool {
	foundMatch := false
	for _, entry := range p {
		// Test all lines to find a match (if a line match = the path must be included)
		if entry.Match(path) {
			if entry.Negate() {
				// An exclusion matched, the file must no longer be included.
				return false
			}
			foundMatch = true
		}
	}
	return foundMatch
}

type LintFile struct {
	Rules []ConfigLintRule `yaml:"rules"`

	Schemas []ConfigLintSchema `yaml:"schemas"`
}

type ConfigLintRule struct {

	// Name of the rule. Must exists in the registry of rules.
	Name string `yaml:"name"`

	// Severity of the rule: "error", "warning". Default to "error".
	Severity string `yaml:"severity"`

	// Optional arguments for the rule.
	Args []string `yaml:"args"`

	// PathRestrictions returns on which paths to evaluate the rule.
	// Glob expressions are supported and ! as prefix indicated to exclude.
	Includes PathSpecs `yaml:"includes"`
}

type ConfigLintSchema struct {
	// Name of the schema used when reporting violations.
	Name       string                       `yaml:"name"`
	Kind       string                       `yaml:"kind"`
	Path       string                       `yaml:"path"`
	Attributes []*ConfigLintSchemaAttribute `yaml:"attributes"`
}
type ConfigLintSchemaAttribute struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Aliases  []string `yaml:"aliases"`
	Pattern  string   `yaml:"pattern"`
	Required *bool    `yaml:"required"`
	Inherit  *bool    `yaml:"inherit"`
}

func (a ConfigLintSchemaAttribute) String() string {
	var specs []string
	if a.Type != "" {
		specs = append(specs, a.Type)
	}
	if a.Pattern != "" {
		specs = append(specs, a.Pattern)
	}
	if *a.Required {
		specs = append(specs, "required")
	}
	if *a.Inherit {
		specs = append(specs, "inherit")
	}
	return strings.Join(specs, ",")
}

func (c ConfigLintSchema) MatchesPath(path string) bool {
	// TODO support glob patterns instead?
	if c.Path == "" {
		// No path defined = apply to all files
		return true
	}
	return strings.HasPrefix(c.Path, path)
}

// TODO refacto move these methods below to attributes.go to avoid having too much logic inside config.go????

// IsInheritableAttribute returns if an attribute can be inherited between files/notes.
func (l *LintFile) IsInheritableAttribute(attributeName string, filePath string) bool {
	for _, schema := range l.Schemas {
		if !schema.MatchesPath(filePath) {
			continue
		}
		for _, attribute := range schema.Attributes {
			if attribute.Name == attributeName {
				return *attribute.Inherit
			}
		}
	}
	return true // Inheritable by default to limit schemas to write
}

// Severity returns the severity of a lint rule.
func (l *LintFile) Severity(name string) string {
	for _, rule := range l.Rules {
		if rule.Name == name {
			return rule.Severity
		}
	}
	return "error" // must not happen but default to error
}

// GetAttributeDefinition returns the attribute definition to use.
func (l *LintFile) GetAttributeDefinition(name string, filter func(schema ConfigLintSchema) bool) *ConfigLintSchemaAttribute {
	// We must find the most specific definition.
	//
	// Ex:
	// schemas:
	// - name: Attributes
	//   attributes:
	//   - name: author
	//     type: string
	//
	// - name: Books
	//   path: references/books/
	//   attributes:
	//   - name: author
	//     required: true
	//
	// We must use the second schema when both apply.

	var matchingSchemas []ConfigLintSchema
	for _, schema := range l.Schemas {
		if !filter(schema) {
			continue
		}
		if slices.ContainsFunc(schema.Attributes, func(a *ConfigLintSchemaAttribute) bool {
			return a.Name == name
		}) {
			matchingSchemas = append(matchingSchemas, schema)
		}
	}
	if len(matchingSchemas) == 0 {
		// Not explicitely defined in schemas
		return nil
	}

	// Sort from most specific to least specific
	slices.SortFunc(matchingSchemas, func(a, b ConfigLintSchema) int {
		// Most specific path first
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		return 1 // The last must win but SortFunc is not stable...
	})

	schemaToUse := matchingSchemas[0]
	for _, definition := range schemaToUse.Attributes {
		if definition.Name == name {
			return definition
		}
	}

	return nil
}

/* Main config */

type Config struct {
	// Absolute top directory containing the .nt sub-directory
	RootDirectory string

	// .nt/config content
	ConfigFile ConfigFile

	// .nt/lint content
	LintFile LintFile

	// .ntignore content
	IgnoreFile IgnoreFile

	// Temporary directory to generate blob files locally
	tempDir string

	// Toggle this flag to skip some side-effects
	DryRun bool
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
			fmt.Fprintln(os.Stderr, "fatal: not a NoteWriter repository (or any of the parent directories): .nt")
			os.Exit(1)
		}
	})
	return configSingleton
}

// TempDir returns the privileged temporary directory to use when generating temporary files.
func (c *Config) TempDir() string {
	if c.tempDir == "" {
		dir, err := os.MkdirTemp("", "the-notewriter")
		if err != nil {
			log.Fatalf("Unable to init temp dir: %v", err)
		}
		c.tempDir = dir
	}
	return c.tempDir
	// FIXME call defer os.RemoveAll(CurrentConfig().TempDir()) from tests?
}

// Converter returns the convertor to use when creating blobs from media files.
func (c *Config) Converter() medias.Converter {
	converterOnce.Do(func() {
		var err error
		mediaConfig := c.ConfigFile.Medias
		switch mediaConfig.Command {
		case "":
			fallthrough
		case "ffmpeg":
			preset := mediaConfig.Preset
			converterSingleton, err = medias.NewFFmpegConverter(preset)
			if err != nil {
				log.Fatal(err)
			}
			converterSingleton.OnPreGeneration(func(cmd string, args ...string) {
				CurrentLogger().Debugf("Running command %q", cmd+" "+strings.Join(args, " "))
			})
		case "random":
			converterSingleton = medias.NewRandomConverter()
		default:
			log.Fatalf("Unsupported converter %q", c.ConfigFile.Medias.Command)
		}
	})
	return converterSingleton
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
	// We muse ignore the .nt directory in user home directory
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	rootPath := path
	i := 0 // Safeguard to not go up too far
	for {
		i++
		if i > maxDepth {
			return nil, nil
		}
		if rootPath == homePath {
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
	_, err = os.Stat(ntConfigPath)
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

	// Check for .nt/lint
	ntLintConfigPath := filepath.Join(rootPath, ".nt", "lint")
	_, err = os.Stat(ntLintConfigPath)
	var lintFile *LintFile
	if os.IsNotExist(err) {
		lintFile, err = parseLintFile(DefaultLint)
		if err != nil {
			return nil, fmt.Errorf("default lint configuration is broken: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check for .nt/lint file: %v", err)
	} else {
		content, err := os.ReadFile(ntLintConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read .nt/lint file: %v", err)
		}
		lintFile, err = parseLintFile(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse .nt/lint file: %v", err)
		}
	}

	// Check for .ntignore
	ntignorePath := filepath.Join(rootPath, ".ntignore")
	_, err = os.Stat(ntignorePath)
	var ignoreFile *IgnoreFile
	if os.IsNotExist(err) {
		ignoreFile, err = parseIgnoreFile(DefaultIgnore)
		if err != nil {
			return nil, fmt.Errorf("default ignore configuration is broken: %v", err)
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

	config := &Config{
		RootDirectory: rootPath,
		ConfigFile:    *configFile,
		IgnoreFile:    *ignoreFile,
	}
	if lintFile != nil {
		config.LintFile = *lintFile
	}
	return config, nil
}

func parseConfigFile(content string) (*ConfigFile, error) {
	r := strings.NewReader(content)
	d := toml.NewDecoder(r)
	d.DisallowUnknownFields()
	var result ConfigFile
	err := d.Decode(&result)

	// Apply default values
	for _, deck := range result.Deck {
		if deck.Algorithm == "" {
			deck.Algorithm = DefaultSRSAlgorithm
		}
		// Only a single one currently supported
		if deck.Algorithm != DefaultSRSAlgorithm {
			return nil, fmt.Errorf("unsupported SRS algorithm %q", deck.Algorithm)
		}
		if deck.BoostFactor == 0 {
			deck.BoostFactor = DefaultSRSBoostFactor
		}
		if deck.AlgorithmSettings == nil {
			deck.AlgorithmSettings = make(map[string]any)
		}
		if _, ok := deck.AlgorithmSettings["easeFactor"]; !ok {
			deck.AlgorithmSettings["easeFactor"] = DefaultSRSEaseFactor
		}
		// And...
		// - Search for all flashcards if query is empty
		// - Don't add new cards by default
		// - Don't limit the number of reviews by default
	}

	return &result, err
}

func parseLintFile(content string) (*LintFile, error) {
	r := strings.NewReader(content)
	d := yaml.NewDecoder(r)
	var result LintFile
	err := d.Decode(&result)

	// Apply default values
	for _, schema := range result.Schemas {
		for _, attribute := range schema.Attributes {
			if attribute.Type == "" {
				attribute.Type = "string"
			}
			if attribute.Inherit == nil {
				attribute.Inherit = BoolPointer(true)
			}
			if attribute.Required == nil {
				attribute.Required = BoolPointer(false)
			}
		}
	}

	return &result, err
}

func parseIgnoreFile(content string) (*IgnoreFile, error) {
	var result IgnoreFile
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if text.IsBlank(line) {
			// ignore blank line
			continue
		}
		if strings.HasPrefix(line, "#") {
			// ignore comment
			continue
		}
		var entry = PathSpec(line)
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
		return nil, fmt.Errorf("current configuration detected: %s", currentConfig.RootDirectory)
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

	// Init .nt/.gitignore file
	gitIgnorePath := filepath.Join(ntPath, ".gitignore")
	_, err = os.Stat(gitIgnorePath)
	if os.IsNotExist(err) { // Do not override existing file!
		err = os.WriteFile(gitIgnorePath, []byte(DefaultGitIgnore), 0644)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
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

func (c *Config) Check() error {

	// Check for invalid reference templates
	for key, referenceConfig := range c.ConfigFile.Reference {
		// Only path and template supports Go Templating
		_, err := reference.ParseTemplate(referenceConfig.Path)
		if err != nil {
			return fmt.Errorf("invalid path for reference %q: %w", key, err)
		}
		_, err = reference.ParseTemplate(referenceConfig.Template)
		if err != nil {
			return fmt.Errorf("invalid template for reference %q: %w", key, err)
		}
	}

	// Check all rules are valid
	for _, rule := range c.LintFile.Rules {
		ruleName := rule.Name
		_, ok := LintRules[ruleName]
		if !ok {
			return fmt.Errorf("unknown lint rule %q", rule.Name)
		}
		if rule.Severity != "" && !slices.Contains([]string{"error", "warning"}, rule.Severity) {
			return fmt.Errorf("unknown severity %q for lint rule %q", rule.Severity, rule.Name)
		}
	}

	// Check for conflicting types in schemas
	attributesTypes := make(map[string]string)
	for _, schema := range c.LintFile.Schemas {
		for _, attribute := range schema.Attributes {
			attributeKnownType, found := attributesTypes[attribute.Name]
			if found && attributeKnownType != attribute.Type {
				return fmt.Errorf("conflicting type for attribute %q: found %s and %s", attribute.Name, attribute.Type, attributeKnownType)
			}
			attributesTypes[attribute.Name] = attribute.Type
		}
	}

	// Check for invalid patterns
	for _, schema := range c.LintFile.Schemas {
		for _, attribute := range schema.Attributes {
			if attribute.Pattern != "string" {
				if _, err := regexp.Compile(attribute.Pattern); err != nil {
					return fmt.Errorf("invalid pattern %q for attribute %q: %v", attribute.Pattern, attribute.Name, err)
				}
			}
		}
	}

	return nil
}

/* Helpers */

func BoolPointer(b bool) *bool {
	return &b
}
