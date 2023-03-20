# Golang Goroutines

## Flashcard: Start a goroutine

`@example: https://go.dev/tour/concurrency/1`

(Go) **How** to **start a goroutine**?

```go
// execute in a goroutine
slowFunction()
```

---

Use the `go` keyword:

```go
go slowFunction()
```
