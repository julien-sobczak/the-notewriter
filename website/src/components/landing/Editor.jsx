import React from "react";

export default function Editor({ filename, lines = [], width = 340, height = 180, top = 0, left = 0, id = "" }) {
    // Determine the number of lines to render (max line number or at least 1)
    const maxLine =
        lines.length > 0 ? Math.max(...lines.map(l => l.number)) : 1;
    const lineHeight = 22;
    const headerHeight = 32;
    const gutterWidth = 36;
    const editorHeight = headerHeight + maxLine * lineHeight + 16;

    // Map line number to line object for quick lookup
    const lineMap = {};
    lines.forEach(line => {
        lineMap[line.number] = line;
    });

    return (
        <svg
            width={width}
            height={editorHeight}
            style={{
                borderRadius: 10,
                background: "#f7f7f7", // light background
                boxShadow: "0 2px 8px #0002",
                position: "relative",
                top,
                left
            }}
            xmlns="http://www.w3.org/2000/svg"
            aria-labelledby={id}
        >
            {/* Tab/Header */}
            <rect x="0" y="0" width={width} height={headerHeight} rx="10" fill="#eaeaea" />
            <text
                x={gutterWidth + 12}
                y={headerHeight / 2 + 6}
                fontFamily="monospace"
                fontSize="15"
                fill="#222"
                fontWeight="bold"
            >
                {filename}
            </text>
            {/* Editor background */}
            <rect
                x="0"
                y={headerHeight}
                width={width}
                height={editorHeight - headerHeight}
                fill="#fff"
                rx="0"
            />
            {/* Lines */}
            {[...Array(maxLine)].map((_, idx) => {
                const lineNum = idx + 1;
                const line = lineMap[lineNum];
                let xCursor = gutterWidth + 8;
                return (
                    <g key={lineNum}>
                        {/* Line number */}
                        <text
                            x={gutterWidth - 8}
                            y={headerHeight + lineHeight * (lineNum - 0.25)}
                            fontFamily="monospace"
                            fontSize="13"
                            fill="#b0b0b0"
                            textAnchor="end"
                        >
                            {lineNum}
                        </text>
                        {/* Tokens */}
                        {line && Array.isArray(line.tokens)
                            ? line.tokens.map((token, tIdx) => {
                                const { text, fill, ...rest } = token;
                                // Add a small gap (e.g. 2px) between tokens
                                const tokenX = xCursor;
                                // Estimate width for monospace font (approx 8px per char) + 2px gap
                                xCursor += ((text?.length || 0) * 8) + 8;
                                return (
                                    <text
                                        key={tIdx}
                                        x={tokenX}
                                        y={headerHeight + lineHeight * (lineNum - 0.25)}
                                        fontFamily="monospace"
                                        fontSize="14"
                                        fill={fill || "#222"}
                                        {...rest}
                                    >
                                        {text}
                                    </text>
                                );
                            })
                            : null}
                    </g>
                );
            })}
        </svg>
    );
}
