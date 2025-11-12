---
title: Medias
---

Notes can include medias (images, videos, audios) using the usual Markdown image syntax.

```md
## Note: Me

![Profile](medias/me.png)
```

## Conversion

All medias are converted using the external dependency `ffmpeg`:

* Images (`jpeg`, `png`, `gif`, `tiff`, ...) ➡️ `avif`
  * A thumbnail image is generated (useful when displaying a list of notes)
  * A medium image is generated (useful when displaying a single note)
* Audios (`wav`, `aac`, `flac`, ...) ➡️ `mp3`
  * A single audio is generated from the original file.
* Videos (`mp4`, `avi`, ...) ➡️ `webm`
  * A `avif` image is generated using the first frame.

Original files are not used directly (= not stored in `.nt/objects`). The desktop application relies on optimized versions to reduce the storage and network bandwidth requirements.

:::tip

Place your medias in a `medias/` directory present along your note files to navigate easily in your editor.

:::

:::tip[Under the hood]
Let's create a note embedding a picture:

```shell
$ wget -O da_vinci_notebook.jpg https://upload.wikimedia.org/wikipedia/commons/6/67/Leonardo_da_vinci%2C_Study_for_the_Last_Supper.jpg
$ cat << EOF > notes.md
# Note-Taking

## Note: Leonardo da Vinci's Notebooks

![Study for the Last Supper](da_vinci_notebook.jpg)
EOF

$ tree .
.
├── da_vinci_notebook.jpg
└── notes.md

$ nt add notes
```

When running `nt add`, the media is extracted as an object and the original file is converted into different formats.

```shell
$ tree .nt
.nt
├── config.jsonnet
├── database.db
├── index
├── nt.libsonnet
└── objects
    ├── 12
    │   └── 12a57a8f22f8d91c38015888d63be8651ddabed6.pack
    ├── 5e
    │   └── 5e4cfa56be4335106ee90942901fc5d07f2b7eae.blob
    ├── 70
    │   └── 70845923c6534721041e9b56b5a97be59f3b9251.blob
    └── a3
        ├── a385ef03b63569418055a730493a96ffae842ea8.blob
        └── a385ef03b63569418055a730493a96ffae842ea8.pack
```

The CLI created two files `.pack`, one for every file in the repository (`notes.md` and `da_vinci_notebook.jpg`):

```shell
$ nt cat-file 12a57a8f22f8d91c38015888d63be8651ddabed6 | grep relative_path
file_relative_path: da_vinci_notebook.jpg
$ nt cat-file a385ef03b63569418055a730493a96ffae842ea8 | grep relative_path
file_relative_path: notes.md
```

We have already presented the content for a [file](./files). The object for a media is similar:

```yaml
$ nt cat-file 12a57a8f22f8d91c38015888d63be8651ddabed6
oid: 12a57a8f22f8d91c38015888d63be8651ddabed6
file_relative_path: da_vinci_notebook.jpg
file_mtime: 2013-10-05T11:05:51+02:00
file_size: 102277
ctime: 2026-01-01T21:13:07.694421+01:00
kind: objects
objects:
  - oid: 8d99fd4ffbed4e6cac77d3c68d3023541276ceda
    kind: media
    desc: media da_vinci_notebook.jpg [8d99fd4ffbed4e6cac77d3c68d3023541276ceda]
    data: eJys...
blobs:
  - oid: 5e4cfa56be4335106ee90942901fc5d07f2b7eae
    mime: image/avif
    tags:
      - preview
      - lossy
  - oid: 70845923c6534721041e9b56b5a97be59f3b9251
    mime: image/avif
    tags:
      - original
      - lossy
```

The pack file contains one object of kind `media`:

```shell
$ nt cat-file 8d99fd4ffbed4e6cac77d3c68d3023541276ceda
oid: 8d99fd4ffbed4e6cac77d3c68d3023541276ceda
relative_path: da_vinci_notebook.jpg
kind: picture
dangling: false
extension: .jpg
mtime: 2013-10-05T11:05:51+02:00
hash: 12a57a8f22f8d91c38015888d63be8651ddabed6
size: 102277
```

The pack file contains also two blobs. Blobs are raw files (often binary files). For example, the file `5e4cfa56be4335106ee90942901fc5d07f2b7eae.blob` is an AVIF picture representing a preview of the original file. You can rename the file to `.avif` or on run `open -a "Preview" .nt/objects/5e/5e4cfa56be4335106ee90942901fc5d07f2b7eae.blob` on MacOS.

:::


