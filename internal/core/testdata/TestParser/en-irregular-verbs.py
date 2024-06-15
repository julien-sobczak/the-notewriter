#!/usr/bin/env python3
import re
import unicodedata

def slugify(value):
    value = unicodedata.normalize('NFKD', value).encode('ascii', 'ignore').decode('ascii').lower()
    return re.sub(r'[\W_]+', '-', value)

IRREGULAR_VERBS = [
    {
        "base": "be",
        "past_tense": "was/were",
        "past_participle": "been",
    },
    {
        "base": "have",
        "past_tense": "had",
        "past_participle": "had",
    },
    # ...
]

for verb in IRREGULAR_VERBS:
    print(f"""
## Flashcard: {verb["base"]}

`@slug: {slugify("em-irregular-verb-" + verb["base"])}`

(Irregular Verb) {verb["base"]}

---

{verb["past_tense"]} / {verb["past_participle"]}
""")
