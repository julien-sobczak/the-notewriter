---
title: "nt-vault"
---

## Name

`nt-vault` — Encrypt and decrypt markdown files.

## Synopsis

```
Usage:
  nt-vault [command]

Available Commands:
  create      Create a new encrypted file
  decrypt     Decrypt an encrypted file
  edit        Edit an encrypted file
  encrypt     Encrypt a plaintext file
  view        View an encrypted file

Flags:
  -h, --help   help for nt-vault
      --v      enable verbose info output
      --vv     enable verbose debug output
      --vvv    enable verbose trace output
```

## Description

`nt-vault` is a tool to encrypt and decrypt markdown files using AES-256-GCM encryption, inspired by [ansible-vault](https://docs.ansible.com/ansible/latest/vault_guide/index.html). It allows you to store sensitive notes securely inside your repository.

The encryption key is provided via the environment variable `NT_VAULT_AES_KEY_FILE`, which must point to a file containing a 32-byte key.

## Subcommands

### `nt-vault create <path>`

Opens an editor to create a new file that will be encrypted when the editor is closed.

### `nt-vault encrypt <path>`

Encrypts an existing plaintext markdown file in-place using the vault secret.

### `nt-vault decrypt <path>`

Decrypts an encrypted file in-place.

### `nt-vault edit <path>`

Opens and decrypts an existing vaulted file in an editor. The file is re-encrypted when the editor is closed.

### `nt-vault view <path>`

Opens, decrypts, and displays an existing vaulted file using a pager.

## Options

* `--v`
  * Enable verbose info output.
* `--vv`
  * Enable verbose debug output.
* `--vvv`
  * Enable verbose trace output.

## Environment Variables

* `NT_VAULT_AES_KEY_FILE`
  * Path to a file containing a 32-byte AES encryption key (required).
* `EDITOR`
  * Editor command used by `create` and `edit` subcommands (default: `vi`).
* `PAGER`
  * Pager command used by the `view` subcommand (default: `less`).

## Examples

* Encrypt an existing file:

        $ nt-vault encrypt notes/private.md

* Create a new encrypted file:

        $ nt-vault create notes/private.md

* View the decrypted content of a vaulted file:

        $ nt-vault view notes/private.md

* Edit a vaulted file:

        $ nt-vault edit notes/private.md

* Decrypt a file in-place:

        $ nt-vault decrypt notes/private.md
