# geticon

A tiny library to get app icons of other applications.

## Installation

```sh
go get github.com/fcjr/geticon
```

## Usage

```go
import (
    "github.com/fcjr/geticon"
)

// get icon of a running application by PID
icon, err := geticon.FromPid(pid) // returns an image.Image

// get icon of an application by path
icon, err := geticon.FromPath(path) // returns an image.Image
```

## Todos

* [x] macOS support
* [ ] windows support
* [ ] test
* [ ] benchmarks
* [ ] linux support?
