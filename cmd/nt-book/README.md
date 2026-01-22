# NoteWriter Book Generation

The `nt-book` command allows you to generate ePub and PDF books from your notes using pandoc.

## Installation

The `nt-book` binary is built along with other NoteWriter tools:

```bash
make build
# Binary is created as build/ntbook
```

## Dependencies

- **pandoc**: Required for book generation. Install with `apt install pandoc` or similar.
- **weasyprint**: Required for PDF generation. Install with `pip install weasyprint` or `apt install python3-weasyprint`.

## Configuration

Books are configured in your `.nt/config.jsonnet` file under the `books` array:

```jsonnet
{
    // ... other configuration ...

    books: [
        {
            title: "My Book Title",
            author: ["Author Name", "Co-Author"],
            language: "en-US",
            toc: true,                    // Generate table of contents
            format: ["epub", "pdf"],      // Output formats
            cover: "path/to/cover.png",   // Optional cover image
            build: "output/my-book.${extension}", // Optional output path
            chapters: [
                {
                    title: "Chapter 1",
                    subtitle: "Optional Subtitle",
                    text: "Direct markdown content here.",
                },
                {
                    title: "Chapter 2",
                    sections: [
                        {
                            title: "Section 1",
                            query: "type:Note #important",
                            pageBreaks: true,
                            includeComments: false,
                        },
                        {
                            title: "Section 2",
                            notes: [
                                {
                                    wikilink: "path/to/note#Note: Title"
                                },
                                {
                                    slug: "note-slug"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    ]
}
```

### Configuration Options

#### Book-level Options

- `title` (required): Book title
- `author` (required): Array of author names
- `language` (required): Language code (e.g., "en-US")
- `toc` (required): Whether to generate table of contents
- `format` (required): Array of output formats ("epub", "pdf")
- `cover` (optional): Path to cover image (local path or URL)
- `build` (optional): Output path template with `${extension}` placeholder

#### Chapter/Section Structure

Each chapter or section can have:

- `title` (required): Section heading
- `subtitle` (optional): Section subtitle
- `text` (optional): Direct markdown content
- `query` (optional): Query to select notes
- `notes` (optional): Specific notes to include
- `sections` (optional): Nested sections
- `pageBreaks` (optional): Insert page breaks between notes
- `includeComments` (optional): Include note comments

### Query Syntax

Queries follow the NoteWriter query syntax:

- `type:Note` - Notes of specific type
- `#tag` - Notes with specific tag
- `path:folder/file.md` - Notes from specific path
- `term1 term2` - Text search terms
- `slug:note-slug` - Specific note by slug

### Note References

You can reference specific notes using:

- `wikilink`: Format like `"path/to/file#Note: Title"`
- `slug`: Unique note slug

## Usage

### Generate All Books

```bash
nt-book generate
```

### Generate Specific Book

```bash
nt-book generate "My Book Title"
```

### Dry Run (Preview Content)

```bash
nt-book generate --dry-run
```

This outputs the generated markdown without creating book files.

### With Verbose Output

```bash
nt-book generate --vv "My Book Title"
```

## Output

Books are generated in the format(s) specified in the configuration:

- **EPUB**: Electronic book format for e-readers
- **PDF**: Portable document format

Default output location is the repository root with slugified book title, or the path specified in the `build` configuration option.

## Styling

The tool includes default CSS styling that works for both EPUB and PDF formats, providing:

- Professional typography using serif fonts
- Proper heading hierarchy and spacing
- Readable line height and margins
- Print-optimized styles for PDF
- E-reader optimized styles for EPUB

## Examples

See the `examples/` directory for a working configuration that generates a sample book from the example notes.

## Troubleshooting

### Pandoc Not Found
```
Error: pandoc is not installed or not in PATH
```
Solution: Install pandoc using your system's package manager.

### Weasyprint Not Found (PDF only)
```
weasyprint not found. Please select a different --pdf-engine
```
Solution: Install weasyprint using `pip install weasyprint`.

### Empty Book Content
If your book generates with empty sections, check that:
1. Your queries match existing notes
2. Notes are committed to the repository
3. Query syntax is correct

### Configuration Errors
The tool validates book configuration and will report specific errors for:
- Missing required fields (title, author, language, format)
- Invalid formats (only "epub" and "pdf" supported)
- Empty format arrays
