# Linter Specification

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

## References

### ESLint

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


### Checkstyle

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

### Buf

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


## Format

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
