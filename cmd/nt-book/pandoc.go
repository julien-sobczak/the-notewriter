package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

func generateBookFormat(config *core.Config, book *core.ConfigBook, markdownContent string, format string) error {
	// Handle markdown format separately (no pandoc needed)
	if format == "markdown" {
		outputPath := book.OutputPath(config, format)
		// Ensure output directory exists
		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		if err := os.WriteFile(outputPath, []byte(markdownContent), 0644); err != nil {
			return fmt.Errorf("failed to write markdown file: %v", err)
		}
		return nil
	}

	// Create temporary markdown file for pandoc
	tempDir := config.TempDir()
	tempMarkdownFile := filepath.Join(tempDir, "book_"+text.Slugify(book.Title)+".md")
	if err := os.WriteFile(tempMarkdownFile, []byte(markdownContent), 0644); err != nil {
		return fmt.Errorf("failed to write temporary markdown file: %v", err)
	}
	defer os.Remove(tempMarkdownFile)

	// Create temporary CSS file for pandoc
	tempCSSFile := filepath.Join(tempDir, "book_"+text.Slugify(book.Title)+".css")
	if err := os.WriteFile(tempCSSFile, []byte(defaultCSS), 0644); err != nil {
		return fmt.Errorf("failed to write temporary CSS file: %v", err)
	}
	defer os.Remove(tempCSSFile)

	// Generate the book file for the specific format
	outputPath := book.OutputPath(config, format)
	if err := generateBookFile(tempMarkdownFile, tempCSSFile, outputPath, format, book); err != nil {
		return fmt.Errorf("error generating %s format: %v", format, err)
	}

	return nil
}

func generateBookFile(markdownFile, cssFile, outputPath, format string, book *core.ConfigBook) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Build pandoc command
	args := []string{}

	// Input file
	args = append(args, markdownFile)

	// Output file
	args = append(args, "-o", outputPath)

	// Add CSS styling
	args = append(args, "--css", cssFile)

	// Format-specific options
	switch strings.ToLower(format) {
	case "epub":
		// Add metadata for EPUB
		args = append(args, "--metadata", fmt.Sprintf("title=%s", book.Title))
		if len(book.Author) > 0 {
			args = append(args, "--metadata", fmt.Sprintf("author=%s", strings.Join(book.Author, ", ")))
		}
		args = append(args, "--metadata", fmt.Sprintf("lang=%s", book.Language))

		// Add cover if specified
		if book.Cover != "" {
			coverPath, err := resolveCoverPath(book.Cover)
			if err != nil {
				fmt.Printf("Warning: Failed to resolve cover image: %v\n", err)
			} else {
				args = append(args, "--epub-cover-image", coverPath)
			}
		}

	case "pdf":
		// PDF-specific options
		args = append(args, "--pdf-engine=weasyprint")

		// Add metadata for PDF
		args = append(args, "--metadata", fmt.Sprintf("title=%s", book.Title))
		if len(book.Author) > 0 {
			args = append(args, "--metadata", fmt.Sprintf("author=%s", strings.Join(book.Author, ", ")))
		}

	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Add table of contents if requested
	if book.TOC {
		args = append(args, "--toc")
	}

	// Execute pandoc
	cmd := exec.Command("pandoc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pandoc command failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func resolveCoverPath(coverPath string) (string, error) {
	// If it's an HTTP URL, return as-is (pandoc can handle URLs)
	if strings.HasPrefix(coverPath, "http://") || strings.HasPrefix(coverPath, "https://") {
		return coverPath, nil
	}

	// If it's a relative path, make it absolute
	if !filepath.IsAbs(coverPath) {
		config := core.CurrentConfig()
		coverPath = filepath.Join(config.RootDirectory, coverPath)
	}

	// Check if file exists
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		return "", fmt.Errorf("cover image file does not exist: %s", coverPath)
	}

	return coverPath, nil
}
