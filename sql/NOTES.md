# Notes

See [Flow source code](https://github.com/julien-sobczak/flow/blob/main/flow.js)

## Go Links

```javascript
getLinks() {
        const links = [];

        const linkRegexp = /(?<!!)\[(.*?)\]\("?(http[^\s"]*)"?(?:\s+["'](.*?)["'])?\)/g;
        // Note: Markdown images uses the same syntax as links but precedes the link by !
        // We use negative lookbehind (supported in major browsers except Safari) to discard them.
        // Ex: "(?<!Y)X" =  matches X, but only if there’s no Y before it.
        // See https://javascript.info/regexp-lookahead-lookbehind for explanations
        // See https://caniuse.com/js-regexp-lookbehind for support status

        for (const [filename, fileContent] of this.files.entries()) {
            const matches = [...fileContent.matchAll(linkRegexp)];
            for (const m of matches) {
                const match = m[0];
                const lineNumber = locate(fileContent, match);
                const text = m[1];
                const url = m[2];
                let title = m[3];
                let goName = undefined;
                if (title) {
                    const subm = title.match(/(?:(.*)\s+)?#go\/(\S+).*/);
                    if (subm) {
                        title = subm[1];
                        goName = subm[2];
                    }
                }
                links.push({
                    kind: "link",
                    source: this.name,
                    tags: this.tags,
                    path: filename,
                    relativePath: filename.replace(this.path, path.basename(this.path)),
                    line: lineNumber,
                    text: text,
                    url: url,
                    title: title,
                    goName: goName,
                });
            }
        }
        return links;
    }
```

## Notes

```javascript
getDocumentsFromFile(filepath, content) {
        const docs = [];
        let headline = undefined;
        const lines = content.split(/\r?\n/);
        let i = 0;
        while (i < lines.length) {
            const line = lines[i];
            if (line.startsWith("# ")) {
                headline = line.substring(2); // trim #
            }
            if (line.startsWith("## ")) {
                // New document
                let contentLines = [];

                // Add section title (prefixed by the document title)
                const title = line.substring("## ".length);
                contentLines.push(`## ${headline} / ${title}`);

                // Add all lines untils the next section/eof
                let lineNumber = i + 1;
                i++;
                while (i < lines.length && !lines[i].startsWith("## ")) {
                    contentLines.push(lines[i]);
                    i++;
                }

                // Remove possible blank ending lines
                while (contentLines[contentLines.length - 1].trim() === '') {
                    contentLines.pop();
                }

                docs.push({
                    kind: "md",
                    source: this.name,
                    tags: this.tags,
                    path: filepath,
                    relativePath: filepath.replace(this.path, path.basename(this.path)),
                    line: lineNumber,
                    fullTitle: markdownToHTML(headline + ' / ' + title),
                    title: markdownToHTML(title),
                    content: contentLines.join('\n'),
                });
            } else {
                i++;
            }
        }

        return docs;
    }
```

## Markdown to HTML

```javascript
function markdownToHTML(text) {
    let html = text;
    html = html.replace(/(?<!\w)\*\*(.*?)\*\*/g, "<b>$1</b>");
    html = html.replace(/(?<!\w)\*(.*?)\*/g, "<b>$1</b>");
    html = html.replace(/(?<!\w)__(.*?)__/g, "<i>$1</i>");
    html = html.replace(/(?<!\w)_(.*?)_/g, "<i>$1</i>");
    html = html.replace(/(?<!\w)``(.*?)``/g, "<code>$1</code>");
    html = html.replace(/(?<!\w)`(.*?)`/g, "<code>$1</code>");
    return html;
}
```


## Flashcards

See [Anki SRS algorithm explained](https://github.com/julien-sobczak/anki-srs-under-the-hood/blob/main/anki/schedv2_annotated.py)
