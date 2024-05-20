---
title: Guide
---

## Output Management

_The NoteWriter_ commands write to stdout using `fmt.PrintX` methods.

```go title=internal/core/database.go
fmt.Printf(" %d objects changes", changesTotal)
```

_The NoteWriter_ commands also log to stderr using Go `log` package (see `internal/core/logger.go`):

```go title=internal/core/database.go
CurrentLogger().Debugf("Uploading blob %s...", blobRef.OID)
```

By default, no log messages are displayed. Use global flags to enable them:

* `--v`: show all messages with a verbosity level >= `info`
* `--vv`: show all messages with a verbosity level >= `debug`
* `--vvv`: show all messages with a verbosity level >= `trace`

Ex:

```shell
$ nt add --vvv example/
2024/01/01 12:21:41 Reading example/...
2024/01/01 12:21:41 Processing example/journal/today.md...
```

Commands can show progress using `\r`:

```go
import (
	"fmt"
	"strings"
	"time"
)

func main() {
	for i := range 10 {
		fmt.Print(strings.Repeat("#", i))
		fmt.Print(strings.Repeat(" ", 10-i))
		fmt.Printf(" (%d%%)\r", i*10)
		time.Sleep(1 * time.Second)
	}
}
```

When using verbose flags, log messages can break progress statuses:

```
##         (20%)
2024/01/01 12:21:41 Reading file note.md
#####      (50%)
2024/01/01 12:21:41 Saving note in database...
#########  (90%)
```

A solution is to redirect stderr to another terminal (or another file):

```shell
# Terminal B
$ tty
/dev/ttys004

# Terminal A
$ nt add . 2>/dev/ttys004
```

The command output will continue to be displayed in the current terminal and debugging logs will flow to the second terminal.
