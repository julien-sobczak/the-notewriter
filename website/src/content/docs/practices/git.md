---
title: Editing Notes with Git
---


_The NoteWriter_ is designed to work with Git. _The NoteWriter_ doesn't version your notes but let you use your favorite version control system.

## Usage

_The NoteWriter_ commands match the usual Git commands. You will probably run `nt add` before running `git add` and will often run `nt commit` before running `nt commit`.


## Secured Files

You may want to push to a remote like GitHub. You may also want some notes to be encrypted before being pushed. You can use the [command `nt-vault`](../reference/commands/nt-vault) for that.

`nt-vault` allows you to securely encrypt sensitive Markdown notes. The encrypted files remain valid Markdown files with a special Front Matter indicating they are encrypted.

```shell
# 1. Generate a key
mkdir -p ~/.nt
head -c 32 /dev/urandom > ~/.nt/vault.key
chmod 0600 ~/.nt/vault.key
export NT_VAULT_AES_KEY_FILE=~/.nt/vault.key

# 2. Create a new encrypted note
nt-vault create personal/diary.md

# 3. Edit it in $EDITOR
nt-vault edit personal/diary.md

# 4. View it without editing
nt-vault view personal/diary.md

# 5. Encrypt an existing note
nt-vault encrypt old-note.md

# 6. Decrypt when needed
nt-vault decrypt old-note.md > old-note.md
```

:::tip

_How to ensure files are never committed unencrypted?_

Add a tag `secure` at the top of your Markdown file. When running `nt add`, if a unencrypted file is parsed with the tag `secure`, the command will fail.

:::


## Secured Objects

When using `nt-vault`, only Markdown files are encrypted. When running `nt add` on these files, the code will use the AES key to decrypt the content and extracts objects into a packfile. The packfile object present under `.nt/objects` will not be encrypted. When using `nt push`, you need to choose carefully your remote if you want to preserve confidentiality (ex: using an object storage solution like [Filebase](https://filebase.com/) or [Storj](https://www.storj.io/) using a decentralized approach).


