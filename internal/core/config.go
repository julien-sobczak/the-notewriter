package core

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"text/template"

	"github.com/google/go-jsonnet"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/medias"
	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

//go:embed config.jsonnet.tmpl
var DefaultConfigTemplateFile string

//go:embed nt.libsonnet
var DefaultConfigLibFile string

/*
 * Ignore Config
 */

// Default .nt/.gitignore content
const DefaultGitIgnore = `
/database.db
/database.db-journal
/objects/
/index
/refs/
/.config.json
`

// Default .ntignore content
const DefaultIgnore = `
build/
README.md
`

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

/*
 * Jsonnet Config
 */

var (
	// Lazy-load configuration and ensure a single read
	configOnce      resync.Once
	configSingleton *Config

	converterOnce      resync.Once
	converterSingleton medias.Converter
)

type ConfigAttributes map[string]*ConfigAttribute
type ConfigNoteTypes map[string]*ConfigNoteType
type ConfigFileTypes map[string]*ConfigFileType

// Find returns the attribute with the given name or nil if not found.
func (a ConfigAttributes) Find(name string) (*ConfigAttribute, bool) {
	for _, attribute := range a {
		if attribute.Name == name {
			return attribute, true
		}
		for _, alias := range attribute.Aliases {
			if alias == name {
				return attribute, true
			}
		}
	}
	return nil, false
}

// Note: Fields must be JSON parser to marshal them
type ConfigFile struct {
	Core ConfigCore `json:"core"`

	Attributes ConfigAttributes `json:"attributes"`
	NoteTypes  ConfigNoteTypes  `json:"noteTypes"`
	FileTypes  ConfigFileTypes  `json:"fileTypes"`

	// Remotes
	Remotes []ConfigRemote `json:"remote"` // IMPROVEMENT Support multiple remotes

	// Predefined searches
	Searches map[string]*ConfigSearch `json:"searches"`

	// Linter
	Linter ConfigLinter `json:"linter"`

	// Extensions

	// Reference options when using the "nt-reference" command
	References []*ConfigReference `json:"references"`

	// Decks definition when declaring notes of type "Flashcard"
	Decks []*ConfigDeck `json:"decks"`

	// Books definition for generating ePub/PDF books from notes
	Books []*ConfigBook `json:"books"`
}

// GetNoteType returns the definition of a note type
func (c *ConfigFile) GetNoteType(noteType string) (*ConfigNoteType, bool) {
	if obj, ok := c.NoteTypes[noteType]; ok {
		return obj, true
	}
	return nil, false
}

// MustGetNoteType returns the definition of a note type or panics if not found
func (c *ConfigFile) MustGetNoteType(noteType string) *ConfigNoteType {
	if obj, ok := c.GetNoteType(noteType); ok {
		return obj
	}
	panic(fmt.Sprintf("Unknown note type %q", noteType))
}

// MatchNoteType checks if the given text matches any of the supported note types.
func (f *ConfigFile) MatchNoteType(heading string) (*ConfigNoteType, markdown.Document, bool) {
	for _, noteType := range f.NoteTypes {
		if title, ok := noteType.MatchHeading(heading); ok {
			return noteType, markdown.Document(title), true
		}
	}
	return nil, markdown.Document(heading), false
}

// GetFileType returns the definition of a file type
func (c *ConfigFile) GetFileType(fileType string) (*ConfigFileType, bool) {
	if obj, ok := c.FileTypes[fileType]; ok {
		return obj, true
	}
	return nil, false
}

// MustGetFileType returns the definition of a file type or panics if not found
func (c *ConfigFile) MustGetFileType(fileType string) *ConfigFileType {
	if obj, ok := c.GetFileType(fileType); ok {
		return obj
	}
	panic(fmt.Sprintf("Unknown file type %q", fileType))
}

// MatchFileType checks if the given file heading matches any of the supported file types.
func (c *ConfigFile) MatchFileType(heading string) (*ConfigFileType, markdown.Document, bool) {
	for _, fileType := range c.FileTypes {
		if title, ok := fileType.MatchHeading(heading); ok {
			return fileType, markdown.Document(title), true
		}
	}
	return nil, markdown.Document(heading), false
}

// GetAttribute returns the definition of an attribute
func (c *ConfigFile) GetAttribute(name string) (*ConfigAttribute, bool) {
	if obj, ok := c.Attributes[name]; ok {
		return obj, true
	}
	return nil, false // Attributes used without a definition is allowed, unlike types
}

// GetAttributeDefaults returns the default values for required attributes of the given type of note.
func (c ConfigFile) GetAttributeDefaults(noteType string) AttributeSet {
	result := make(AttributeSet)
	typeDefinition := c.MustGetNoteType(noteType)
	for _, attribute := range typeDefinition.RequiredAttributes() {
		if attributeDefinition, ok := c.GetAttribute(attribute.Name); ok {
			if attributeDefinition.DefaultValue != nil {
				attributeValue := MustCastAttribute(attributeDefinition.DefaultValue, *attributeDefinition)
				result[attribute.Name] = attributeValue
			}
		}
	}
	return result
}

// DefaultConfigFile is the default configuration file
func (c *ConfigLinter) Severity(name string) string {
	for _, rule := range c.Rules {
		if rule.Name == name {
			return rule.Severity
		}
	}
	// Default severity is "error"
	return "error"
}

type ConfigCore struct {
	Extensions []string     `json:"extensions"` // List of supported file extensions (ex: "md", "markdown")
	Medias     ConfigMedias `json:"medias"`     // Media converter configuration
}
type ConfigAttribute struct {
	Name              string         `json:"name"`
	Aliases           []string       `json:"aliases,omitempty"`
	Type              string         `json:"type"`                    // string, integer, bool, string[], integer[], bool[]
	Format            string         `json:"format"`                  // Useful for value types (ex: "markdown", "date", etc.)
	Min               int            `json:"min,omitempty"`           // Default: 0 (for "number"-type only)
	Max               int            `json:"max,omitempty"`           // Default: -1 (for "number"-type only)
	Pattern           string         `json:"pattern,omitempty"`       // Regex (for "string"-type only)
	Memory            *bool          `json:"memory,omitempty"`        // (for "date"-type only)
	Inherit           *bool          `json:"inherit"`                 // Default: true
	AllowedValues     []string       `json:"allowedValues,omitempty"` // For "string"-type only
	Shorthands        map[string]any `json:"shorthands,omitempty"`
	ShorthandPattern  string         `json:"shorthandPattern,omitempty"`
	PreserveShorthand *bool          `json:"preserveShorthand"`      // Default: true
	DefaultValue      any            `json:"defaultValue,omitempty"` // For any type
	DailyMetrics      *bool          `json:"dailyMetrics,omitempty"` // Whether to include this attribute in daily metrics stats
}

type ConfigTypeAttribute struct {
	Name          string `json:"name"`
	Required      *bool  `json:"required,omitempty"`      // Default: false
	PromoteInline *bool  `json:"promoteInline,omitempty"` // Default: false
}

type ConfigNoteType struct {
	Name       string                `json:"name"`
	Pattern    string                `json:"pattern,omitempty"`    // Regex to detect Markdown headings matching the type
	Processors []string              `json:"processors"`           // Additional logic to run after parsing a note
	Attributes []ConfigTypeAttribute `json:"attributes,omitempty"` // Structure for attributes
	Hooks      []string              `json:"hooks"`                // List of hooks to run on this type of note
}

// ConfigFileSchema defines the structure of a file's Markdown schema
type ConfigFileSchema struct {
	Body *ConfigHeadingSchema `json:"body,omitempty"` // Body structure validation rules
}

// ConfigHeadingSchema defines the structure and validation for a heading in the document
type ConfigHeadingSchema struct {
	MatchType     string                 `json:"matchType,omitempty"` // Regex to validate the note type of heading (e.g., "Note", "Quote")
	Match         string                 `json:"match,omitempty"`     // Regex to validate the raw heading text
	Required      bool                   `json:"required"`            // Whether this heading is required
	AllowMultiple bool                   `json:"allowMultiple"`       // Whether multiple instances are allowed
	EnforceOrder  bool                   `json:"enforceOrder"`        // Whether child headings must appear in order
	Children      []*ConfigHeadingSchema `json:"children,omitempty"`  // Nested heading schemas
}

type ConfigFileType struct {
	Name       string                `json:"name"`
	Pattern    string                `json:"pattern,omitempty"`    // Regex to match file titles
	Attributes []ConfigTypeAttribute `json:"attributes,omitempty"` // Required/optional attributes
	Schema     *ConfigFileSchema     `json:"schema,omitempty"`     // Markdown schema definition for validation
	Processors []string              `json:"processors"`           // Additional logic to run after parsing a file
}
type ConfigLinter struct {
	Rules []*ConfigLinterRule `json:"rules"` // List of rules to apply to notes
}
type ConfigLinterRule struct {
	Name     string `json:"name"`
	Args     []any  `json:"args"`
	Severity string `json:"severity"`        // error, warning (default: error)
	Query    string `json:"query,omitempty"` // Optional query to restrict which notes are concerned
}

type ConfigMedias struct {
	Command  string `json:"command"`
	Parallel int    `json:"parallel"`
	Preset   string `json:"preset"`
}
type ConfigRemote struct {
	Name string `json:"name"` // Name as specified when running commands
	Type string `json:"type"` // fs or s3
	// fs-specific attributes
	Dir string `json:"dir,omitempty"`
	// s3-specific attributes
	Endpoint   string `json:"endpoint,omitempty"`
	AccessKey  string `json:"-"` // Use $NT_S3_ACCESS_KEY to set
	SecretKey  string `json:"-"` // Use $NT_S3_SECRET_KEY to set
	BucketName string `json:"bucketName,omitempty"`
	Secure     bool   `json:"secure,omitempty"`
}
type ConfigDeck struct {
	Name  string `json:"name"`
	Query string `json:"query"`
	// General attributes
	BoostFactor         int `json:"boostFactor"`         // How passionate I am on this topic (100 = neutral, 80 = challenging, 120 = smooth)
	NewFlashcardsPerDay int `json:"newFlashcardsPerDay"` // How many new flashcards to add every day (= 0 no more cards for now)
	MaxFlashcardsPerDay int `json:"maxFlashcardsPerDay"` // How many flashcards (including new) to review every day (= 0 no limit, review what is due)
	// Specific attributes
	Algorithm         string         `json:"algorithm"`         // Anki2
	AlgorithmSettings map[string]any `json:"algorithmSettings"` // SRS-specific attributes
}
type ConfigSearch struct {
	Title string `json:"title"`
	Q     string `json:"q"`
}
type ConfigReference struct {
	Title    string `json:"title"`    // Ex: "A book"
	Manager  string `json:"manager"`  // Ex: "zotero"
	Path     string `json:"path"`     // Ex: "references/books"
	Template string `json:"template"` // Ex: "# {{.Title}}\n"
}

type ConfigBook struct {
	Title    string               `json:"title"`           // Ex: "Life in Progress"
	Author   []string             `json:"author"`          // Ex: ["Julien Sobczak"]
	Cover    string               `json:"cover,omitempty"` // Optional URL or path to cover image
	Language string               `json:"language"`        // Ex: "en-US"
	TOC      bool                 `json:"toc"`             // Generate table of contents
	Format   []string             `json:"format"`          // Ex: ["epub", "pdf"]
	Build    string               `json:"build,omitempty"` // Ex: "build/life-in-progress.${extension}"
	Chapters []*ConfigBookSection `json:"chapters"`        // Book chapters
}

type ConfigBookSection struct {
	Title           string               `json:"title"`                     // Chapter/section title
	Subtitle        string               `json:"subtitle,omitempty"`        // Optional subtitle
	Illustration    string               `json:"illustration,omitempty"`    // Optional image path (relative from root)
	Text            string               `json:"text,omitempty"`            // Direct text content
	Query           string               `json:"query,omitempty"`           // Query to select notes
	Notes           []*ConfigBookNote    `json:"notes,omitempty"`           // Specific notes to include
	Sections        []*ConfigBookSection `json:"sections,omitempty"`        // Nested sections
	PageBreaks      bool                 `json:"pageBreaks,omitempty"`      // Force page breaks between notes
	IncludeComments bool                 `json:"includeComments,omitempty"` // Include comment fields
}

type ConfigBookNote struct {
	Wikilink string `json:"wikilink,omitempty"` // Ex: "projects/life-in-progress#Note: Afterword"
	Slug     string `json:"slug,omitempty"`     // Alternative to wikilink
}

// OutputPath returns the output path for a book in the specified format
func (b *ConfigBook) OutputPath(config *Config, format string) string {
	if b.Build != "" {
		// Use configured build path with extension substitution
		path := strings.ReplaceAll(b.Build, "${extension}", format)
		if !filepath.IsAbs(path) {
			path = filepath.Join(config.RootDirectory, path)
		}
		return path
	}

	// Default: use book title as filename in repository root
	filename := text.Slugify(b.Title) + "." + format
	return filepath.Join(config.RootDirectory, filename)
}

// SetParallel overrides the value in config file.
func (c *Config) SetParallel(value int) {
	c.ConfigFile.Core.Medias.Parallel = value
}

// RequiredAttribute checks if the given attribute is required for this note type.
func (c *ConfigNoteType) RequiredAttribute(name string) bool {
	for _, attr := range c.Attributes {
		if attr.Name == name && attr.Required != nil && *attr.Required {
			return true
		}
	}
	return false
}

// PromoteInlineAttribute checks if the given attribute could be promoted as a note attribute.
func (c *ConfigNoteType) PromoteInlineAttribute(name string) bool {
	for _, attr := range c.Attributes {
		if attr.Name == name && attr.PromoteInline != nil && *attr.PromoteInline {
			return true
		}
	}
	return false
}

// RequiredAttributes returns the list of required attributes for this note type.
func (c *ConfigNoteType) RequiredAttributes() []*ConfigTypeAttribute {
	var requiredAttrs []*ConfigTypeAttribute
	for _, attr := range c.Attributes {
		if attr.Required != nil && *attr.Required {
			requiredAttrs = append(requiredAttrs, &attr)
		}
	}
	return requiredAttrs
}

// MatchHeading checks if the given heading matches the note type.
func (c *ConfigNoteType) MatchHeading(heading string) (string, bool) {
	if c.Pattern == "" {
		return "", false
	}
	rePattern := regexp.MustCompile(c.Pattern)
	if m := rePattern.FindStringSubmatch(heading); m != nil {
		return m[1], true
	}
	return "", false
}

// MatchHeading checks if the given file heading matches the file type pattern.
func (c *ConfigFileType) MatchHeading(heading string) (string, bool) {
	if c.Pattern == "" {
		return "", false
	}
	rePattern := regexp.MustCompile(c.Pattern)
	if m := rePattern.FindStringSubmatch(heading); m != nil {
		return m[1], true
	}
	return "", false
}

// RequiredAttribute checks if the given attribute is required for this note type.
func (c *ConfigFileType) RequiredAttribute(name string) bool {
	for _, attr := range c.Attributes {
		if attr.Name == name && attr.Required != nil && *attr.Required {
			return true
		}
	}
	return false
}

// RequiredAttributes returns the list of required attributes for this note type.
func (c *ConfigFileType) RequiredAttributes() []*ConfigTypeAttribute {
	var requiredAttrs []*ConfigTypeAttribute
	for _, attr := range c.Attributes {
		if attr.Required != nil && *attr.Required {
			requiredAttrs = append(requiredAttrs, &attr)
		}
	}
	return requiredAttrs
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

// Valid validates a string value against this attribute's constraints.
// Returns an empty string if valid, or an error message if invalid.
func (c *ConfigAttribute) Valid(value any) (bool, error) {
	// First check if the value can be cast to the correct type
	typedValue, ok := CastAttribute(value, *c)
	if !ok {
		return false, fmt.Errorf("value %q is not a valid %s or cannot be converted", value, c.Type)
	}

	// Check pattern constraint (only for string type)
	if c.Pattern != "" && c.Type == "string" {
		if matched, err := regexp.MatchString(c.Pattern, typedValue.(string)); err != nil {
			return false, fmt.Errorf("pattern %q is invalid: %v", c.Pattern, err)
		} else if !matched {
			return false, fmt.Errorf("value %q does not match pattern %q", value, c.Pattern)
		}
	}

	// Check allowedValues constraint (only for string type)
	if len(c.AllowedValues) > 0 && c.Type == "string" {
		found := false
		for _, allowedValue := range c.AllowedValues {
			if allowedValue == value {
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("value %q is not in allowedValues %v", value, c.AllowedValues)
		}
	}

	return true, nil
}

func (c ConfigAttribute) String() string {
	typeStr := c.Type
	if c.Pattern != "" {
		typeStr = fmt.Sprintf("%s/%s", c.Type, c.Pattern)
	}
	return fmt.Sprintf("%s (%s)", c.Name, typeStr)
}

func (c ConfigHeadingSchema) String() string {
	var sb strings.Builder
	sb.WriteString("heading(")

	attrs := []string{}
	if c.Match != "" {
		attrs = append(attrs, "@title=~"+c.Match)
	}
	if c.MatchType != "" {
		attrs = append(attrs, "@type=~"+c.MatchType)
	}

	sb.WriteString(strings.Join(attrs, ", "))
	sb.WriteString(")")

	return sb.String()
}

/*
 * PathSpecs
 */

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

/*
 * Main Config
 */

// How many parent directories to traverse before considering a directory as not a nt repository
const maxDepth = 10

// SRS
const (
	DefaultSRSBoostFactor = 100
	DefaultSRSAlgorithm   = "Anki2"
	DefaultSRSEaseFactor  = 2.5
)

type Config struct {
	// Absolute top directory containing the .nt sub-directory
	RootDirectory string

	// .nt/config content
	ConfigFile *ConfigFile

	// .ntignore content
	IgnoreFile IgnoreFile

	// Dir used to store temporary files like blob generations.
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

// Reload rereads the configuration file and returns the new configuration.
func (c *Config) Reload() *Config {
	configOnce.Reset()
	return CurrentConfig()
}

// CurrentConfigFile returns the current configuration file.
func CurrentConfigFile() *ConfigFile {
	config := CurrentConfig()
	if config == nil {
		log.Fatalf("Not a NoteWriter repository (or any of the parent directories)")
	}
	return config.ConfigFile
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
	// IMPROVEMENT Call defer os.RemoveAll(CurrentConfig().TempDir()) from tests?
}

// Converter returns the convertor to use when creating blobs from media files.
func (c *Config) Converter() medias.Converter {
	converterOnce.Do(func() {
		var err error
		mediaConfig := c.ConfigFile.Core.Medias
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
			log.Fatalf("Unsupported converter %q", c.ConfigFile.Core.Medias.Command)
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
	if path, ok := os.LookupEnv("NT_HOME"); ok && path != "" {
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

	// Check for .nt/config.jsonnet
	ntConfigPath := filepath.Join(rootPath, ".nt", "config.jsonnet")
	_, err = os.Stat(ntConfigPath)
	var configFile *ConfigFile
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to locate .nt/config.jsonnet file: %v", err)
	}
	configFile, err = ParseConfigFile(ntConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .nt/config.jsonnet file: %v", err)
	}

	// Check for .ntignore
	ntignorePath := filepath.Join(rootPath, ".ntignore")
	_, err = os.Stat(ntignorePath)
	var ignoreFile *IgnoreFile
	if os.IsNotExist(err) {
		// Unlike config.jsonnet, .ntignore is optional
		CurrentLogger().Debugf("No .ntignore file found in %s", rootPath)
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
		ConfigFile:    configFile,
		DryRun:        false,
	}
	if ignoreFile != nil {
		config.IgnoreFile = *ignoreFile
	}
	return config, nil
}

// ReservedAttributes are the attributes that are used internally by the application and must not be overriden or redeclared.
var ReservedAttributes = map[string]ConfigAttribute{
	"title": {
		Name:    "title",
		Type:    "string",
		Inherit: BoolPointer(false), // Each note must have a unique title
	},
	"slug": {
		Name:    "slug",
		Type:    "string",
		Inherit: BoolPointer(false), // Same reason as "title"
	},

	"date": {
		Name:    "date",
		Type:    "string",
		Format:  "date", // TODO map[format]{pattern} ???
		Inherit: BoolPointer(true),
	},

	"hook": {
		Name:    "hook",
		Type:    "string[]",
		Inherit: BoolPointer(false), // Subnotes must not trigger the hook
	},

	"tags": {
		Name:    "tags",
		Type:    "string[]",
		Inherit: BoolPointer(true),
	},

	"source": {
		Name:    "source",
		Type:    "string",
		Format:  "markdown",
		Inherit: BoolPointer(true),
	},

	"references": {
		Name:    "references",
		Type:    "string[]",
		Format:  "markdown",
		Inherit: BoolPointer(false),
	},

	"inspirations": {
		Name:    "inspirations",
		Type:    "string[]",
		Format:  "markdown",
		Inherit: BoolPointer(false),
	},
}

func ParseConfigFile(jsonnetPath string) (*ConfigFile, error) {
	// Evaluate the Jsonnet template
	vm := jsonnet.MakeVM()
	jsonContent, err := vm.EvaluateFile(jsonnetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate config.jsonnet: %v", err)
	}

	// Parse the JSON content
	result := ConfigFile{
		// Default values
		Core: ConfigCore{
			Extensions: []string{"md", "markdown"},
			Medias: ConfigMedias{
				Command:  "ffmpeg",
				Parallel: 1,
				Preset:   "ultrafast",
			},
		},
	}
	err = json.Unmarshal([]byte(jsonContent), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Apply default values...
	// ... to attributes
	for _, attribute := range result.Attributes {
		if attribute.Inherit == nil {
			attribute.Inherit = BoolPointer(true)
		}
		if attribute.PreserveShorthand == nil {
			attribute.PreserveShorthand = BoolPointer(true)
		}
	}
	// Add reserved attributes
	if result.Attributes == nil {
		result.Attributes = make(ConfigAttributes)
	}
	for _, attribute := range ReservedAttributes {
		result.Attributes[attribute.Name] = &attribute
	}

	// ... to note types
	for _, noteType := range result.NoteTypes {
		if noteType.Pattern == "" {
			noteType.Pattern = fmt.Sprintf("(?i)^%s:\\s*(.*)$", noteType.Name) // Ex: "Note: A basic note" or "note: A basic note"
		}
		// Set defaults for new attribute structure
		for i := range noteType.Attributes {
			if noteType.Attributes[i].Required == nil {
				noteType.Attributes[i].Required = BoolPointer(false)
			}
			if noteType.Attributes[i].PromoteInline == nil {
				noteType.Attributes[i].PromoteInline = BoolPointer(false)
			}
		}
	}

	// ... to file types
	for _, fileType := range result.FileTypes {
		if fileType.Pattern == "" {
			fileType.Pattern = fmt.Sprintf("(?i)^%s:\\s*(.*)$", fileType.Name) // Ex: "Reading: My Book Title" or "reading: My book title"
		}
	}
	// ... to decks
	for _, deck := range result.Decks {
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
func InitConfigFromDirectory(path string, options ConfigOptions) (*Config, error) {
	currentConfig, _ := ReadConfigFromDirectory(path)
	if currentConfig != nil {
		// Do not override current configuration
		return nil, fmt.Errorf("current configuration detected: %s", currentConfig.RootDirectory)
	}

	// Create .nt directory
	ntPath := filepath.Join(path, ".nt")
	if err := os.Mkdir(ntPath, 0755); err != nil {
		return nil, err
	}

	err := InitConfigFileFromDirectory(path, options)
	if err != nil {
		return nil, fmt.Errorf("failed to init config file: %v", err)
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

// ConfigOptions are the options used to initialize the configuration file.
// Useful to override default values, especially in unit tests
// (ex: avoid the external dependency on ffmpeg).
type ConfigOptions struct {
	// IMPROVEMENT add more options and ask questions during 'nt init -i'
	MediaConverter string
}

// DefaultConfigOptions are the default options used to initialize the configuration file.
var DefaultConfigOptions = ConfigOptions{
	MediaConverter: "ffmpeg",
}

// InitConfigFileFromDirectory initializes the default configuration files under a .nt subdirectory.
func InitConfigFileFromDirectory(path string, options ConfigOptions) error {
	CurrentLogger().Debugf("✨ Set up configuration files under %s/.nt", path)

	// Init .nt/ directory
	ntPath := filepath.Join(path, ".nt")
	if err := os.MkdirAll(ntPath, 0755); err != nil {
		return fmt.Errorf("failed to create .nt directory: %v", err)
	}
	CurrentLogger().Infof("✅ Created directory %s", ntPath)

	// Init .nt/nt.libsonnet file
	ntConfigLibPath := filepath.Join(ntPath, "nt.libsonnet")
	if err := os.WriteFile(ntConfigLibPath, []byte(DefaultConfigLibFile), 0644); err != nil {
		return err
	}
	CurrentLogger().Infof("✅ Created file %s/nt.libsonnet", ntPath)

	// Generate config.jsonnet file content.
	// We use a template to allow for dynamic values (convenient in tests).
	tmpl, err := template.New("config.jsonnet").Parse(DefaultConfigTemplateFile)
	if err != nil {
		// Syntax error?
		return err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, options)
	if err != nil {
		return err
	}

	// Init .nt/config.jsonnet file
	ntConfigPath := filepath.Join(ntPath, "config.jsonnet")
	if err := os.WriteFile(ntConfigPath, buf.Bytes(), 0644); err != nil {
		return err
	}
	CurrentLogger().Infof("✅ Created file %s/config.jsonnet", ntPath)

	return nil
}

func (c *Config) Check() error {

	// Check for invalid note types
	for _, noteType := range c.ConfigFile.NoteTypes {
		// Check for invalid regex pattern
		if noteType.Pattern != "" {
			if _, err := regexp.Compile(noteType.Pattern); err != nil {
				return fmt.Errorf("invalid pattern %q for note type %q: %v", noteType.Pattern, noteType.Name, err)
			}
		}
	}

	// Check for invalid file types
	for _, fileType := range c.ConfigFile.FileTypes {
		// Check for invalid regex pattern
		if fileType.Pattern != "" {
			if _, err := regexp.Compile(fileType.Pattern); err != nil {
				return fmt.Errorf("invalid pattern %q for file type %q: %v", fileType.Pattern, fileType.Name, err)
			}
		}
		// Check for invalid regexes in schema
		if fileType.Schema != nil {
			err := c.checkHeadingSchema(fileType.Schema.Body)
			if err != nil {
				return err
			}
		}
	}

	// Check for default attribute values
	for _, attribute := range c.ConfigFile.Attributes {
		if attribute.DefaultValue != nil {
			_, valid := CastAttribute(attribute.DefaultValue, *attribute)
			if !valid {
				return fmt.Errorf("default value %q for attribute %q is not valid", attribute.DefaultValue, attribute.Name)
			}
		}
	}

	// Check for invalid attributes
	for _, attribute := range c.ConfigFile.Attributes {
		// Check for invalid regex pattern
		if attribute.Pattern != "" && attribute.Pattern != "string" {
			if _, err := regexp.Compile(attribute.Pattern); err != nil {
				return fmt.Errorf("invalid pattern %q for attribute %q: %v", attribute.Pattern, attribute.Name, err)
			}
		}

		// Check for invalid shorthand regex pattern
		if attribute.ShorthandPattern != "" {
			if _, err := regexp.Compile(attribute.ShorthandPattern); err != nil {
				return fmt.Errorf("invalid shorthand pattern %q for attribute %q: %v", attribute.ShorthandPattern, attribute.Name, err)
			}
		}

		// Check that Memory is only set on date format attributes
		if attribute.Memory != nil && *attribute.Memory {
			if attribute.Type != "string" && attribute.Type != "date" {
				return fmt.Errorf("memory can only be enabled on string or date type attributes, but attribute %q has type %q", attribute.Name, attribute.Type)
			}
			if attribute.Format == "" {
				return fmt.Errorf("memory requires a date format to be specified, but attribute %q has no format", attribute.Name)
			}

		}

		// Check for invalid shorthand values
		for shorthandKey, shorthandValue := range attribute.Shorthands {
			if valid, err := attribute.Valid(shorthandValue); !valid {
				return fmt.Errorf("shorthand value %q for key %q in attribute %q is not valid: %s", shorthandValue, shorthandKey, attribute.Name, err)
			}
		}
	}

	// Check all rules are valid
	for _, rule := range c.ConfigFile.Linter.Rules {
		ruleName := rule.Name
		_, ok := LintRulesFn[ruleName]
		if !ok {
			return fmt.Errorf("unknown lint rule %q", rule.Name)
		}
		if rule.Severity != "" && !slices.Contains([]string{"error", "warning"}, rule.Severity) {
			return fmt.Errorf("unknown severity %q for lint rule %q", rule.Severity, rule.Name)
		}
		if rule.Query != "" {
			_, err := ParseQuery(rule.Query)
			if err != nil {
				return fmt.Errorf("invalid query %q for lint rule %q: %v", rule.Query, rule.Name, err)
			}
		}
	}

	// Check for invalid reference templates
	for key, referenceConfig := range c.ConfigFile.References {
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

	// Check for invalid book configurations
	for _, book := range c.ConfigFile.Books {
		if book.Title == "" {
			return fmt.Errorf("book title cannot be empty")
		}
		if len(book.Format) == 0 {
			return fmt.Errorf("book '%s' must specify at least one format", book.Title)
		}
		for _, format := range book.Format {
			if !slices.Contains([]string{"epub", "pdf", "markdown"}, strings.ToLower(format)) {
				return fmt.Errorf("unsupported format '%s' for book '%s'. Only 'epub', 'pdf', and 'markdown' are supported", format, book.Title)
			}
		}
		if len(book.Author) == 0 {
			return fmt.Errorf("book '%s' must specify at least one author", book.Title)
		}
		if book.Language == "" {
			return fmt.Errorf("book '%s' must specify a language", book.Title)
		}
	}

	return nil
}

func (c *Config) checkHeadingSchema(schema *ConfigHeadingSchema) error {
	if schema.Match != "" {
		if _, err := regexp.Compile(schema.Match); err != nil {
			return fmt.Errorf("invalid match pattern %q in heading schema: %v", schema.Match, err)
		}
	}
	if schema.MatchType != "" {
		if _, err := regexp.Compile(schema.MatchType); err != nil {
			return fmt.Errorf("invalid match type pattern %q in heading schema: %v", schema.MatchType, err)
		}
	}
	for _, child := range schema.Children {
		if err := c.checkHeadingSchema(child); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) GetGeneratedPath() string {
	return filepath.Join(c.RootDirectory, ".nt", ".config.json")
}

// Save saves the configuration to the .nt/config.json file.
func (c *Config) Save() error {
	// We want to save ConfigFile to .nt/config.json
	// so that the result could be pushed to remotes
	// and read by the desktop application without having a dependency on Jsonnet.

	// Serialize the ConfigFile to JSON
	jsonData, err := json.MarshalIndent(c.ConfigFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize configuration to JSON: %v", err)
	}

	// Write the JSON data to the file, overriding if it already exists
	err = os.WriteFile(c.GetGeneratedPath(), jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration to %s: %v", c.GetGeneratedPath(), err)
	}

	return nil
}

/* Helpers */

func BoolPointer(b bool) *bool {
	return &b
}
