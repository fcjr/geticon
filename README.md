# geticon

A tiny cGo library to get app icons of running applications.

## Installation

    go get github.com/fcjr/geticon

## Usage

```go
import (
    "github.com/fcjr/geticon"
)

icon, err := geticon.FromPid(pid) // returns an image.Image
```

## Todos

* [x] macOS support
* [ ] windows support
* [ ] test
* [ ] benchmarks
* [ ] linux support?
