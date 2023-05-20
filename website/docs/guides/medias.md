---
sidebar_position: 3
---

# Medias

Notes can include medias (images, videos, audios) using the usual Markdown syntax.

```md
## Reference: Me

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

Original files are not used directly (= not stored in `.nt/objects`). The applications _The NoteWriter Desktop_ and _The NoteWriter Nomad_ rely on optimized versions to reduce the storage and network bandwidth requirements.

:::tip

Place your medias in a `medias/` directory present along your note file to navigate easily in your editor.

:::
