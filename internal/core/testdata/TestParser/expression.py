import re
import unicodedata

def slugify(value):
    value = unicodedata.normalize('NFKD', value).encode('ascii', 'ignore').decode('ascii').lower()
    return re.sub(r'[\W_]+', '-', value)

EXPRESSIONS=[
    {
        "en-en": "be caught between a rock and a hard place",
        "fr-fr": "être pris entre le marteau et l'enclume",
    }
]
for expr in EXPRESSIONS:
    print(f"""
## Flashcard: {expr["en-en"]}

`@slug: {slugify(expr["en-en"])}`

(Expression) **Translate**

_<span class="foreign">{expr["en-en"]}</span>_

---

_<span class="native">{expr["fr-fr"]}</span>_

""")
