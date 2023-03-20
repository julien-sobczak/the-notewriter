# Linter Specification

## Rules

Here is a list of rules:

* `no-duplicate-note-title` => Enfore no title have the same long name
* `min-lines-between-notes` => Enforce a minimum number of lines between notes
* `note-title-match` => Enforce a consistent naming for notes
* `no-free-note` => Forbid untyped notes (must be able to exclude paths)
* `no-dangling-media` => Path to media files must exist
* `no-dead-wikilink` => Links between notes must exist
* `no-extension-wikilink` => No extension in wikilink
* `no-ambiguous-wikilink` => Force wikilinks to be qualified enough

We need to configure if rules are `warn` (commit/push OK), `error` (push KO) or `off` (do not run the rule).
We need to restrict the rules on a subjet of note files.

### References

#### ESLint

Check [documentation](https://eslint.org/)

```json
{
    "root": true,
    "extends": [
        "eslint:recommended",
        "plugin:@typescript-eslint/recommended"
    ],
    "parser": "@typescript-eslint/parser",
    "parserOptions": { "project": ["./tsconfig.json"] },
    "plugins": [
        "@typescript-eslint"
    ],
    "rules": {
        "@typescript-eslint/strict-boolean-expressions": [
            2,
            {
                "allowString" : false,
                "allowNumber" : false
            }
        ]
    },
    "ignorePatterns": ["src/**/*.test.ts", "src/frontend/generated/*"]
}
```

Support also JavaScript and YAML syntaxes.


#### Checkstyle

Check [documentation](https://checkstyle.sourceforge.io/config.html)

```xml
<module name="MethodLength">
  <property name="max" value="60"/>
</module>
<module name="Checker">
  <module name="JavadocPackage"/>
  <module name="TreeWalker">
    <property name="tabWidth" value="4"/>
    <module name="AvoidStarImport"/>
    <module name="ConstantName"/>
    ...
  </module>
</module>
```

#### Buf

Check [Documentation](https://docs.buf.build/lint/usage)

```yaml
lint:
  use:
    - DEFAULT
  except:
    - ENUM_VALUE_UPPER_SNAKE_CASE # We use lower case
    - ENUM_VALUE_PREFIX           # We do not enforce enum value prefix
    - ENUM_ZERO_VALUE_SUFFIX      # We do not enforce enum zero-value suffix
    - RPC_RESPONSE_STANDARD_NAME  # CRUD and SAD pattern return same type message
    - RPC_REQUEST_RESPONSE_UNIQUE # ^ same
  ignore:
    - dirA/
  ignore_only:
    MESSAGE_PASCAL_CASE:
      - dirA/fileA.proto
    FIELD_LOWER_SNAKE_CASE:
      - dirA/fileB.proto
  service_suffix: API
  rpc_allow_google_protobuf_empty_requests: true
```


### Format

We will use YAML unlink `.nt/config` which uses TOML like Git. Rules must be configured and TOML is not as expressive as YAML.

Proposal:

```yaml
# .nt/lint
rules:

# Forbid duplicate note titles
- name: no-duplicate-note-title
  includes:
  - "!archives"

# Enforce a minimum number of lines between notes
- name: min-lines-between-notes
  severity: warning # Default to error
  args: [2]

# Forbid untyped notes (must be able to exclude paths)
- name: no-free-note
  includes: # default to root
  - projects/
  - references/
  - todo/
  - "!todo/misc"

# Path to media files must exist
- name: no-dangling-media

# Links between notes must exist
- name: no-dead-wikilink

# No extension in wikilink
- name: no-extension-wikilink
  severity: warning

# No ambiguity in wikilinks
- name: no-ambiguous-wikilink
```

## Attributes

### References

#### XSD Schema

Check [example](https://www.w3schools.com/xml/schema_example.asp):

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">

<xs:simpleType name="stringtype">
  <xs:restriction base="xs:string"/>
</xs:simpleType>

<xs:simpleType name="inttype">
  <xs:restriction base="xs:positiveInteger"/>
</xs:simpleType>

<xs:simpleType name="dectype">
  <xs:restriction base="xs:decimal"/>
</xs:simpleType>

<xs:simpleType name="orderidtype">
  <xs:restriction base="xs:string">
    <xs:pattern value="[0-9]{6}"/>
  </xs:restriction>
</xs:simpleType>

<xs:complexType name="shiptotype">
  <xs:sequence>
    <xs:element name="name" type="stringtype"/>
    <xs:element name="address" type="stringtype"/>
    <xs:element name="city" type="stringtype"/>
    <xs:element name="country" type="stringtype"/>
  </xs:sequence>
</xs:complexType>

<xs:complexType name="itemtype">
  <xs:sequence>
    <xs:element name="title" type="stringtype"/>
    <xs:element name="note" type="stringtype" minOccurs="0"/>
    <xs:element name="quantity" type="inttype"/>
    <xs:element name="price" type="dectype"/>
  </xs:sequence>
</xs:complexType>

<xs:complexType name="shipordertype">
  <xs:sequence>
    <xs:element name="orderperson" type="stringtype"/>
    <xs:element name="shipto" type="shiptotype"/>
    <xs:element name="item" maxOccurs="unbounded" type="itemtype"/>
  </xs:sequence>
  <xs:attribute name="orderid" type="orderidtype" use="required"/>
</xs:complexType>

<xs:element name="shiporder" type="shipordertype"/>

</xs:schema>
```


#### YAML Schema

See [example](https://blog.picnic.nl/how-to-use-yaml-schema-to-validate-your-yaml-files-c82c049c2097):

A document:

```yaml
---
draft: true
title: This guy validated his YAML
body: This post will explain how a random guy started validating his YAML.
tags:
  - "tech"
  - "YAML"
```

A schema to validate the document:

```yaml
---
type: "//rec"
required:
  title: "//str"
  body: "//str"
  draft: "//bool"
optional:
  subtitle: "//str"
  tags:
    type: "//arr"
    contents: "//str"
```

#### JSON Schema

See [example](https://json-schema.org/learn/getting-started-step-by-step):

A document:

```json
{
  "productId": 1,
  "productName": "A green door",
  "price": 12.50,
  "tags": [ "home", "green" ]
}
```

A schema to validate the document:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/product.schema.json",
  "title": "Product",
  "description": "A product from Acme's catalog",
  "type": "object",
  "properties": {
    "productId": {
      "description": "The unique identifier for a product",
      "type": "integer"
    },
    "productName": {
      "description": "Name of the product",
      "type": "string"
    },
    "price": {
      "description": "The price of the product",
      "type": "number",
      "exclusiveMinimum": 0
    },
    "tags": {
      "description": "Tags for the product",
      "type": "array",
      "items": {
        "type": "string"
      },
      "minItems": 1,
      "uniqueItems": true
    }
  },
  "required": [ "productId", "productName", "price" ]
}
```

Notes:

- Famous schemas validate a document with a well-defined structure. We are more interested in making sure some attributes are present on notes that are spreaded in many files. The motivation is different.


### Format


The goal is the validate required attributes are present on notes (ex: `Quote` notes must have the same `name` or `author`). The idea is to support schemas for notes and let the lint enforces them.

Examples of validations:

* Quotes must contains a `name` or `author` attributes.
* TODOs can contains an attribute `priority` with values `high,medium,low`
* The attribute `tags` is an array and is inheritable (sub notes must automatically inherit this attribute).
* The tag `favorite` is not inheritable (if put on a note, it must not be ideally included on a flashcard created from this note) => Hardcode the rule for special tags instead of making the format even more complex.

Let's try to make a format supporting these use cases:

```yaml
rules:
   ...

schemas:
  - name: Quotes must be attributed # Do not use a comment to be able to include this message when reporting violations
    query: kind:quote
    attributes:
      - name: name
        type: string|number|object|array|bool # https://www.w3schools.com/js/js_json_datatypes.asp
        aliases: [artist, writer, author]
        pattern: .* # Only for string attributes
        required: true
      - name: year
        pattern: \d{4}
      - name: source
        required: true

  - name: Tags
    attributes:
      - name: tags
        type: array
        inherit: true

  - name: Links
    attributes:
      - name: references
        type: array
        inherit: true
      - name: source
        inherit: true
      - name: inspirations
        type: array
        inherit: true
```

💡 Inline schema! Let file includes a schema in their Front Matter to validate all notes inside the file against it.

```markdown
---
schemas:
- name: Children Books
  query: "kind:Artwork" # Can be used to restrict on which notes INSIDE this document the schema apply
  attributes:
    - required:
      - name: year
    - required:
      - name: author
    - required:
      - name: illustrator
---

# Caldecott Award

## Dorothy P. Lathrop - Animals of the Bible (1938) - Winner

<!-- title: Animals of the Bible -->
<!-- year: 1965 -->
<!-- author: Helen Dean Fish -->
<!-- illustrator: Dorothy P. Lathrop -->

...
```
