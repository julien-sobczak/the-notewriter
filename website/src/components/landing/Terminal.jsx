export default function Terminal({
    id = "",
    top = 0,
    left = 0,
    width = 500,
    content = [],
    lineHeight = 18,
    padding = 20,
    fontSize = 14,
    fontFamily = "monospace",
}) {
    const paddingBottom = 20;
    const height = content.length * lineHeight + padding * 2 + paddingBottom;
    return (
        <svg
            id={id}
            width={width}
            height={height}
            style={{ position: "absolute", top, left }}
            viewBox={`0 0 ${width} ${height}`}
            xmlns="http://www.w3.org/2000/svg"
        >
            {/* Background */}
            <rect width={width} height={height} rx="12" fill="#111" />

            {/* Traffic lights */}
            <circle cx={padding} cy={padding} r="6" fill="#f55" />
            <circle cx={padding + 20} cy={padding} r="6" fill="#fa5" />
            <circle cx={padding + 40} cy={padding} r="6" fill="#5f5" />

            {/* Code contents */}
            {content.map((line, i) => {
                const isCmd = line.type === "cmd";

                return (
                    <text
                        key={i}
                        x={padding}
                        y={padding + 30 + i * lineHeight}
                        fontSize={fontSize}
                        fontFamily={fontFamily}
                        fontWeight={isCmd ? 'bold' : 'normal'}
                        fill="#e5e7eb"
                    >
                        {isCmd ? `$ ${line.text}` : line.text}
                    </text>
                );
            })}
        </svg>
    );
}


/*
How to use it?
<div style={{ position: "relative", height: 800 }}>
  <Terminal width={300} content={[
    { type: "cmd", text: "nt add" },
    { type: "output", text: "1 note added" },
    { type: "output", text: "Command successful" }
  ]} />
</div>
*/
