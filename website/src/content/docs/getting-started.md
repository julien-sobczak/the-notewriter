---
title: Getting Started
---

:::note[Prerequisites]
This guide requires that you have already installed the CLI following the [instructions in the `README.md`](https://github.com/julien-sobczak/the-notewriter#installing).
:::

1. Create a new directory

```shell
$ mkdir ~/notes
$ cd ~/notes
```

2. Init the repository

```shell
$ nt init
```

A new subdirectory `.nt` has been created in the current repository containing the default configuration files.

```shell
$ tree -a
.
├── .nt
│   ├── .gitignore
│   ├── config.jsonnet
│   └── nt.libsonnet
└── .ntignore
```

:::tip[Git?]
The CLI `nt` is largely inspired by `git`. The directory containing your notes is also called a repository. _The NoteWriter_ is not a versioning tool and is designed to work with Git to version your notes.
:::

3. Create a new file

````txt
$ cat << EOF > go.md
---
tags: go
---

# Go

## Common Mistakes

### Note: Using goroutines on loop iterator variables

`@source: https://go.dev/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables`

```go
// ❌ DON'T
for _, val := range values {
    go func() {
        fmt.Println(val)
    }()
}

// ✅ DO
for _, val := range values {
    go func(val interface{}) {
        fmt.Println(val)
    }(val)
}
```

By adding `val` as a parameter to the closure, `val` is evaluated at each iteration and placed on the stack for the goroutine.

## Note: Golang Conferences `#watching`

* GopherCon, USA, Late August `#remind-every-${year}-08`
* GopherCon Europe, Mid June

### Flashcard: Largest Conference

(Go) **Largest conference**?

---

**GopherCon** is the United States.
EOF
````

In practice, you will use your favorite editor or IDE to edit your Markdown notes.

4. Add the file

```shell
$ nt add .
```

Things becomes more interesting. The command `add` parses all Markdown files to extract objects (like notes and flashcards) and add them to the index. The objects are also saved in a SQLite database for performance reasons and for full-text search support.

```txt
$ tree -a
.
├── .nt
│   ├── .gitignore
│   ├── config.jsonnet
│   ├── database.db
│   ├── index
│   ├── nt.libsonnet
│   └── objects
│       └── 98
│           ├── 984f0d5226f6cb15ebae9ec89579ae82a7b2f487.blob
│           └── 984f0d5226f6cb15ebae9ec89579ae82a7b2f487.pack
├── .ntignore
└── go.md
```

5. Manipulate the objects

The CLI `nt` is mainly used when editing your notes. To use your notes creatively, you will use _The NoteWriter Desktop_. Follow the [installation instructions](https://github.com/julien-sobczak/the-notewriter-desktop?tab=readme-ov-file#installing) and simply run inside your repository:

```shell
$ nt-desktop
```

This application is where your notes will come alive. You will be able to study your flashcards, complete your reminders, render notes like you would manipulate them on your desk, or even relax and review random notes to find inspiration.
