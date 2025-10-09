# nt-vault

A tool to encrypt and decrypt markdown files using AES-256 encryption, inspired by ansible-vault.

## Overview

`nt-vault` allows you to securely encrypt sensitive markdown notes so that they can be stored in version control (like GitHub) without exposing private information. The encrypted files remain valid markdown files with a special frontmatter indicating they are encrypted.

## Installation

Build the binary:

```bash
make build
```

The binary will be available at `build/ntvault`.

## Setup

### Creating an Encryption Key

Generate a 32-byte random key for AES-256 encryption:

```bash
mkdir -p ~/.nt
head -c 32 /dev/urandom > ~/.nt/vault.key
chmod 0600 ~/.nt/vault.key
```

**Important:** The key file must have restricted permissions (0600 or 0400) for security. The tool will refuse to work with keys that have overly permissive file permissions.

### Setting the Environment Variable

Set the `NT_VAULT_AES_KEY_FILE` environment variable to point to your key file:

```bash
export NT_VAULT_AES_KEY_FILE=~/.nt/vault.key
```

You may want to add this to your shell profile (e.g., `~/.bashrc` or `~/.zshrc`).

## Usage

### Create a New Encrypted File

Opens your editor (from `$EDITOR`) to create a new file that will be encrypted when saved:

```bash
nt-vault create path/to/secret-note.md
```

### Encrypt an Existing File

Encrypts a plaintext markdown file in-place:

```bash
nt-vault encrypt path/to/note.md
```

### Decrypt a File

Decrypt to stdout:

```bash
nt-vault decrypt path/to/encrypted-note.md
```

Decrypt to a file:

```bash
nt-vault decrypt --output decrypted.md path/to/encrypted-note.md
```

### Edit an Encrypted File

Opens the encrypted file in your editor, decrypted. When you close the editor, the file is re-encrypted:

```bash
nt-vault edit path/to/encrypted-note.md
```

### View an Encrypted File

Opens the decrypted content in a pager (from `$PAGER`, defaults to `less`):

```bash
nt-vault view path/to/encrypted-note.md
```

## Encrypted File Format

Encrypted files are valid markdown files with the following structure:

```markdown
---
encrypted: true
version: "1.0"
algorithm: AES256
hmac: <base64-encoded-hmac>
---

```
<base64-encoded-encrypted-content>
```
```

The file contains:
- **YAML frontmatter** with encryption metadata
- **encrypted** flag set to `true`
- **version** of the encryption format
- **algorithm** used (AES256 with GCM mode)
- **hmac** for integrity verification (HMAC-SHA256)
- **Encrypted content** in a code block, base64-encoded

## Security Features

1. **AES-256-GCM Encryption**: Uses the Go standard library's AES implementation with Galois/Counter Mode (GCM) for authenticated encryption
2. **HMAC Verification**: Each file includes an HMAC-SHA256 to detect tampering or use of the wrong key
3. **Key File Permissions**: Enforces strict file permissions (0600 or 0400) on the key file
4. **Secure Random Nonces**: Each encryption uses a cryptographically secure random nonce

## Environment Variables

- `NT_VAULT_AES_KEY_FILE`: Path to the 32-byte encryption key file (required)
- `EDITOR`: Editor to use for create/edit commands (defaults to `vi`)
- `PAGER`: Pager to use for view command (defaults to `less`)

## Examples

### Complete Workflow

```bash
# 1. Generate a key
mkdir -p ~/.nt
head -c 32 /dev/urandom > ~/.nt/vault.key
chmod 0600 ~/.nt/vault.key
export NT_VAULT_AES_KEY_FILE=~/.nt/vault.key

# 2. Create a new encrypted note
nt-vault create personal/diary.md

# 3. Edit it later
nt-vault edit personal/diary.md

# 4. View it without editing
nt-vault view personal/diary.md

# 5. Encrypt an existing note
nt-vault encrypt old-note.md

# 6. Decrypt when needed
nt-vault decrypt old-note.md > plaintext.md
```

### Using with Git

Encrypted files can be safely committed to Git:

```bash
# Encrypt sensitive files
nt-vault encrypt secrets/passwords.md

# Add to git
git add secrets/passwords.md
git commit -m "Add encrypted password notes"

# The encrypted content is safe to push to GitHub
git push
```

## Best Practices

1. **Never commit the key file**: Add your key file to `.gitignore`
2. **Backup your key securely**: If you lose the key, encrypted files cannot be recovered
3. **Use different keys for different repositories**: Don't reuse keys across projects
4. **Set restrictive permissions**: Always use `chmod 0600` on key files
