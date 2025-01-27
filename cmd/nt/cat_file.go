package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/spf13/cobra"
)

var outputFormat string

func init() {
	catFileCmd.Flags().StringVarP(&outputFormat, "format", "o", "yaml", "format of output. Allowed: json, md, yaml")
	rootCmd.AddCommand(catFileCmd)
}

var catFileCmd = &cobra.Command{
	Use:   "cat-file",
	Short: "Display a repository file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Too many arguments. You can only have one which must be a wikilink or an OID")
			os.Exit(1)
		}

		arg := args[0]

		if oid := oid.ParseOrNil(arg); !oid.IsNil() {
			dumpOID(oid)
		}

		// If the argument is not an oid.OID, it must be a path
		dumpPath(arg)
	},
}

// dumpOID checks if the given OID exists and dumps it
func dumpOID(oid oid.OID) {
	// OIDs can represent a pack file, an object inside a pack file, or a blob.
	packFile, err := core.CurrentDB().Index().ReadPackFile(oid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read pack file: %v", err)
		os.Exit(1)
	}
	if packFile != nil {
		dumpObject(packFile)
		return
	}
	object, err := core.CurrentDB().Index().ReadObject(oid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read object: %v", err)
		os.Exit(1)
	}
	if object != nil {
		dumpObject(object)
		return
	}
	blob, err := core.CurrentDB().Index().ReadBlob(oid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read blob: %v", err)
		os.Exit(1)
	}
	if blob != nil {
		dumpObject(blob)
		return
	}

	fmt.Fprintf(os.Stderr, "Unknown OID %s", oid)
	os.Exit(1)
}

// dumpPath checks if the given path exists and returns the filename if it does
func dumpPath(path string) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Path does not exist: %s", path)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read file: %v", err)
		os.Exit(1)
	}
	if info.IsDir() {
		fmt.Fprintf(os.Stderr, "Path is a directory: %s", path)
		os.Exit(1)
	}
	filename := filepath.Base(path)
	if !isOID(filename) {
		fmt.Fprintf(os.Stderr, "Path to an invalid file: %s", path)
		os.Exit(1)
	}

	dumpOID(oid.MustParse(filename))
}

func dumpObject(object core.Dumpable) {
	switch outputFormat {
	case "yaml":
		fmt.Println(object.ToYAML())
	case "json":
		fmt.Println(object.ToJSON())
	case "md":
		fallthrough
	case "markdown":
		fmt.Println(object.ToMarkdown())
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output format %q", outputFormat)
		os.Exit(1)
	}
}

// isOID checks if the value looks like an OID.
func isOID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
