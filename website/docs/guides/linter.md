---
sidebar_position: 3
---

# Linter

**TODO**

## Rules


## Schemas

Schemas are only required when using the rule `check-attribute`.

```yaml
rules:
# Enforce strict rules (consistency helps when having a large collection of notes)
- name: no-duplicate-note-title
- name: min-lines-between-notes
  args: [2]
- name: max-lines-between-notes
  args: [2]
- name: no-free-note
- name: no-dangling-media
- name: no-dead-wikilink
- name: no-extension-wikilink
- name: no-ambiguous-wikilink
- name: check-attribute
- name: require-quote-tag
  args: ["^(learning|doing|being|perseverance|money|life|passion|purpose|belief|work|parenting|success|history|thinking|creativity|courage|productivity|programming|reading|deciding|sleeping|leadership|understanding|management|mindfulness|meditation|intelligence|health|planning|happiness|complexity|widsom|art|drawing|running|writing|stress|note-taking|innovation|relationship|humor|imagination|persuasion|excellence|changing|listening|death|philosophy|friendship|time|aging|curiosity|habit|memory|self-help)$"]
  includes:
  - references/ # Ignore quotes under projects/ for example

schemas:
- name: Tags
  attributes:
    - name: tags
      type: array
      inherit: true
- name: Hooks
  attributes:
    - name: hook
      type: array
      inherit: false
- name: Links
  attributes:
    - name: references
      type: array
      inherit: true
    - name: source
      inherit: true
- name: Quotes
  kind: quote
  path: references
  attributes:
    # Force quotes to have an author (use "Anonymous" if needed)
    - name: name
      aliases: [author, illustrator]
      type: string
      required: true
    # Force authors of quotes to have their main occupation filled to append in rendered quotes
    - name: occupation
      type: string
      required: true
    # Force authors of quotes to have their nationality
    - name: nationality
      type: string
      required: true
- name: Reviews
  kind: note
  path: references/reviews
  attributes:
    # Force to tell if a review can be edited (prevent premature publication my mistake)
    - name: draft
      type: boolean
      required: true
    - name: isbn
      required: true
    - name: subject
      required: true
```
