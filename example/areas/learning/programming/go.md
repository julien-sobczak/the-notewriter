# Go Programming

## Syntax

### Flashcard: Variable Declaration

What are the **three ways** to **declare a variable** in Go?

---

1. `var name string = "value"` - explicit type
2. `var name = "value"` - type inference
3. `name := "value"` - short declaration (only inside functions)

### Flashcard: Slice vs Array

What's the **key difference** between a **slice** and an **array** in Go?

---

Arrays have fixed size defined at compile time `[3]int`, while slices have dynamic size `[]int`. Slices are more commonly used and are references to arrays.

### Flashcard: Defer Statement

What does the **`defer` keyword** do in Go?

---

Defer postpones the execution of a function until the surrounding function returns. Deferred functions are executed in LIFO (Last In First Out) order.

### Flashcard: Error Handling

What's the **idiomatic way** to **handle errors** in Go?

---

Check errors explicitly: `if err != nil { return err }`. Go doesn't use exceptions; errors are values returned by functions.

### Flashcard: Goroutines

How do you **create a goroutine**?

---

Add the `go` keyword before a function call: `go myFunction()`. This runs the function concurrently in a lightweight thread.

### Flashcard: Channels

What's a **buffered channel** and how is it **different from unbuffered**?

---

Buffered channels have capacity: `ch := make(chan int, 3)`. They block only when full (send) or empty (receive). Unbuffered channels block until both sender and receiver are ready.

### Flashcard: Interface Implementation

How do you **implement an interface** in Go?

---

Implicitly. If a type implements all methods of an interface, it satisfies that interface. No explicit declaration needed.

### Flashcard: Pointer Receivers

When should you use **pointer receivers** vs **value receivers** for methods?

---

Use pointer receivers when:
1. Method needs to modify the receiver
2. Receiver is a large struct (avoid copying)
3. Consistency (if some methods use pointers, all should)

### Flashcard: Select Statement

What does the **`select` statement** do in Go?

---

`select` lets a goroutine wait on multiple channel operations. It blocks until one case can proceed, similar to switch but for channels.

### Flashcard: Struct Embedding

How does **struct embedding** work in Go?

---

A struct can embed another struct to "inherit" its fields and methods:
```go
type Person struct { Name string }
type Employee struct {
    Person  // embedded
    Title string
}
```

### Flashcard: Context Package

What's the **purpose** of the **context package**?

---

Provides a way to carry deadlines, cancellation signals, and request-scoped values across API boundaries and goroutines.

### Flashcard: Empty Interface

What does **`interface{}`** (or **`any`** in Go 1.18+) represent?

---

The empty interface can hold values of any type since every type implements zero methods. Used for generic programming before Go 1.18.

## Tools

### List: Conferences

* GopherCon - The largest Go conference, held annually in multiple locations worldwide `@date: 2006` `@continent: North America`
* dotGo - European Go conference held in Paris, France `@date: 2014` `@continent: Europe`
* GoLab - Italian Go conference focused on practical applications `@date: 2015` `@continent: Europe`
* GopherCon Europe - European edition with rotating locations `@date: 2015` `@continent: Europe`
* Go devroom at FOSDEM - Open source conference with dedicated Go track `@date: 2009` `@continent: Europe`
* Capital Go - Conference in Washington DC focusing on Go in production `@date: 2016` `@continent: North America`
* GoWayFest - Conference in Minsk focusing on Go ecosystem `@date: 2017` `@continent: Europe`

### List: Frameworks

* [Gin](https://github.com/gin-gonic/gin) - High-performance HTTP web framework `#web` `#framework`
* [Echo](https://echo.labstack.com/) - High performance, minimalist Go web framework `#web` `#framework`
* [Fiber](https://gofiber.io/) - Express-inspired web framework built on Fasthttp `#web` `#framework`
* [Beego](https://beego.vip/) - Full-featured MVC framework for rapid development `#web` `#framework`
* [Buffalo](https://gobuffalo.io/) - Rapid web development framework with scaffolding `#web` `#framework`
* [Revel](https://revel.github.io/) - High-productivity framework inspired by Rails `#web` `#framework`
* [Iris](https://www.iris-go.com/) - Fast HTTP/2 web framework with MVC support `#web` `#framework`
* [Chi](https://github.com/go-chi/chi) - Lightweight router for building Go HTTP services `#web` `#library`
* [Gorilla](https://github.com/gorilla) - Collection of packages for web applications (mux, sessions, etc.) `#web` `#library`
* [Martini](https://github.com/go-martini/martini) - Classy web framework (now deprecated but influential) `#web` `#framework`

### List: OSS

* [Kubernetes](https://kubernetes.io/) - Container orchestration platform `#devops` `#infrastructure`
* [Docker](https://www.docker.com/) - Containerization platform (core written in Go) `#devops` `#infrastructure`
* [Prometheus](https://prometheus.io/) - Monitoring and alerting toolkit `#devops` `#monitoring`
* [Terraform](https://www.terraform.io/) - Infrastructure as code tool `#devops` `#infrastructure`
* [etcd](https://etcd.io/) - Distributed key-value store `#database` `#infrastructure`
* [Hugo](https://gohugo.io/) - Fast static site generator `#web` `#tooling`
* [CockroachDB](https://www.cockroachlabs.com/) - Distributed SQL database `#database`
* [InfluxDB](https://www.influxdata.com/) - Time series database `#database` `#monitoring`
* [Consul](https://www.consul.io/) - Service mesh and service discovery `#devops` `#infrastructure`
* [Vault](https://www.vaultproject.io/) - Secrets management tool `#devops` `#security`
* [Traefik](https://traefik.io/) - Modern HTTP reverse proxy and load balancer `#web` `#devops`
* [Caddy](https://caddyserver.com/) - Web server with automatic HTTPS `#web` `#server`
* [Minio](https://min.io/) - High-performance object storage `#storage` `#infrastructure`
* [Gitea](https://gitea.io/) - Git service written in Go `#devops` `#tooling`
* [Drone](https://www.drone.io/) - Container-native CI/CD platform `#devops` `#ci-cd`
