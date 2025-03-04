---
title: Hooks
---


Hooks allow you to run custom script before a note is committed.

## Syntax

Hooks are declared using the attribute `hook`.

```md title=todo.md
## TODO: Reading List

`@hook: gist`

* [*] _Limitless_, by Jim Kwik
* [*] _Tribe of Mentors_, by Tim Ferris
```

Just before a new commit is created, _The NoteWriter_ will try to execute the hook by looking for an **executable** filename (ignoring the extension) present in `.nt/hooks` (ex: `gist.py`).

You can use any language to write your hooks. The JSON representation of the note is available on stdin:

```json
{
    "oid": "16252daf",
    "relativePath": "todo.md",
    "wikilink": "#TODO: Reading List",
    "attributes": {
     "title": "Reading List",
     "hook": "gist",
    },
    "shortTitleRaw": "Reading List",
    "shortTitleMarkdown": "Reading List",
    "shortTitleHTML": "Reading List",
    "shortTitleText": "Reading List",
    "contentRaw": "* [*] _Limitless_, by Jim Kwik\n* [*] _Tribe of Mentors_, by Tim Ferris",
    "contentMarkdown": "* [*] _Limitless_, by Jim Kwik\n* [*] _Tribe of Mentors_, by Tim Ferris",
    "contentHTML": "<ul>\n<li>[*] <em>Limitless</em>, by Jim Kwik</li>\n<li>[ ] <em>Tribe of Mentors</em>, by Tim Ferris</li>\n</ul>",
    "contentText": "* [*] _Limitless_, by Jim Kwik\n* [*] _Tribe of Mentors_, by Tim Ferris",
}
```

## Run

Hooks are automatically triggered when commiting changes using the comand `nt commit`.

Sometimes, you may want to run a hook manually (useful when developing new hooks). The command `nt run-hook` allows to execute a hook on a single note (you still need to use `nt add` to place the note in the index).

```shell
$ nt run-hook --vvv "todo.md#Reference: Reading List"
```

## Example

Let's write the hook `gist` that synchronize the note content (usually stored in a private GitHub repository) with a Gist that can be shared publicly.

```md title=todo.md
## TODO: Reading List

`@hook: gist`

* [*] _Limitless_, by Jim Kwik
* [*] _Tribe of Mentors_, by Tim Ferris
```

Here is an example of hook written in Python:

```py title=.nt/hooks/gist.py
#!/usr/bin/env python
"""
This script uses the GitHub API to create gists from a note.

Note: The script expect an environment variable $GITHUB_GIST_TOKEN to be defined (ex: ~/.bashrc).
"""

import sys
import os
import fileinput
import json
import requests

class GistClient:

    def __init__(self, token):
        self.api_url = "https://api.github.com/gists"
        self.token = token

    def create_gist(self, description, filename, content):
        request = json.dumps({
            'description': description,
            'public': False, # Not really secret, share the link
            'files': {
                filename: {
                    'content': content,
                },
            },
        })
        response  = requests.post(self.api_url, headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {self.token}",
            "X-GitHub-Api-Version": "2022-11-28",
        }, data=request)
        if not response.ok:
            print(f"Unable to create Gist due to code {response.status_code}",
              file=sys.stderr)
            os._exit(1)

if __name__ == "__main__":

    # Step 0: Check secrets
    api_token = os.environ.get('GITHUB_GIST_TOKEN')
    if not api_token:
        print("Missing env variable $GITHUB_GIST_TOKEN", file=sys.stderr)
        os._exit(1)

    github = GistClient(api_token)

    # Step 1: Read input note
    note_str = ""
    for line in fileinput.input():
        note_str += line
    note = json.loads(note_str)

    # Step 2: Create the gist
    print(f'Creating gist from note {note["shortTitleText"]}...', file=sys.stderr)
    github.create_gist(note["shortTitleText"], "note.md", note["contentMarkdown"])
```

This hook is incomplete. Hooks must be idempotent as you don't want to create a new Gist every time the note is edited. But you now have a good idea of how hooks work.

