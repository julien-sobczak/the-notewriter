import { useEffect, useState, useRef } from "react";

// Utility to parse text with option groups in parentheses
function parseOptionGroups(text) {
    // Example: "(A|B) foo (C|D)" => [{options: ["A","B"]}, " foo ", {options: ["C","D"]}]
    const regex = /\(([^)]+)\)/g;
    let result = [];
    let lastIndex = 0;
    let match;
    let groupIndex = 0;
    while ((match = regex.exec(text)) !== null) {
        if (match.index > lastIndex) {
            result.push(text.slice(lastIndex, match.index));
        }
        result.push({
            options: match[1].split("|"),
            group: groupIndex++,
        });
        lastIndex = regex.lastIndex;
    }
    if (lastIndex < text.length) {
        result.push(text.slice(lastIndex));
    }
    return result;
}

export default function Tagline({ text }) {
    const groups = parseOptionGroups(text);
    // Track which option is selected for each group
    const [indices, setIndices] = useState(groups.map(g => (g.options ? getRandomInt(g.options.length) : null)));
    const [activeGroup, setActiveGroup] = useState(0);

    useInterval(() => {
        setActiveGroup(prevActiveGroup => {
            // Find the group indices with options
            const optionGroups = groups
                .map((g, idx) => (g.options ? idx : null))
                .filter(idx => idx !== null);
            if (optionGroups.length === 0) return prevActiveGroup;

            const groupIdx = optionGroups[prevActiveGroup];
            setIndices(prev => {
                const next = [...prev];
                next[groupIdx] = ((next[groupIdx] || 0) + 1) % groups[groupIdx].options.length;
                return next;
            });
            // Move to next group
            return (prevActiveGroup + 1) % optionGroups.length;
        });
    }, 2000);

    return (
        <span className="Tagline">
            {groups.map((g, i) => {
                if (g.options) { // Option group
                    return (
                        <span key={i} className="TaglineOption">
                            {g.options[indices[i] || 0]}
                        </span>
                    );
                } else { // raw text
                    // Replace newlines with <br />
                    const parts = g.split('\\n');
                    console.log(g, `found ${parts.length} parts in raw text:`, parts);
                    return parts.map((part, idx) =>
                        idx === 0
                            ? part
                            : [<br key={`br-${i}-${idx}`} />, part]
                    );
                }
            })}
        </span>
    );
}

/* Utility hook to set up an interval */
function useInterval(callback, delay) {
  const savedCallback = useRef();

  // Remember the latest callback.
  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  // Set up the interval.
  useEffect(() => {
    function tick() {
      savedCallback.current();
    }
    if (delay !== null) {
      let id = setInterval(tick, delay);
      return () => clearInterval(id);
    }
  }, [delay]);
}

/* Utility to get a random integer from 0 to max-1 */
function getRandomInt(max) {
  return Math.floor(Math.random() * max);
}
