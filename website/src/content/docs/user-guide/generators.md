---
title: Generators
---

A `Generator` note is a special note type that allows you to dynamically generate notes using scripts. This is useful to automate the creation of similar notes (ex: flashcards).

Scripts must output valid Markdown document, just as if you have entered the notes manually. The Markdown will be parsed by _The NoteWriter_. If the script doesn't exit with status code `0`, the parsing fails abruptly.

## Examples

### Inline Python Script

The following note will generate a list of flashcards based on a Python list:

````md
## Generator: Inline

`@interpreter: python3`

```python
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
```
````

When parsing this note, _The NoteWriter_ creates a temporary file containing the first codeblock present in the note. The optional attribute `interpreter` is used to override the binary used to run this script. By default, the codeblock language is used (ex: `python`). In this example, _The NoteWriter_ will run `python3 /tmp/path/to/temp/file`. The standard output will then be copied and parsed again as if the Markdown has been inserted manually.


### External Python Script

For more complex scripts or when using compiled language, the script may be defined in a different file.

````md title
## Generator: External

`@file: expressions.py`

This note could contain additional information about the script. The content will not be searchable and be replaced with the script output.
````

Where `expressions.py` is present in the same directory (the path is resolved using the note absolute path):

```python
#!/usr/bin/env python
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
```

When using external scripts, the script file must be executable (`chmod +x expressions.py`) and must declare the shebang for interpreted languages.

_The NoteWriter_ will therefore trigger the command `./expressions.py` to copy the standard output:

```md
## Flashcard: be caught between a rock and a hard place

`@slug: caught-between-rock-and-hard-place`

(Expression) **Translate**

_<span class="foreign">be caught between a rock and a hard place</span>_

---

_<span class="native">être pris entre le marteau et l'enclume</span>_
```

To declare new flashcards for new expressions to learn, simply edit the Python variable to append a new items in the list.


:::tip

Use generators to automate the creation of your flashcard and thus limit duplication in your notes, and make it easy to refactor them later.

:::
