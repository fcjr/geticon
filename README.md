# geticon

A tiny cross-plaform (macOS + windows) library to get app icons of other applications.

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

## Technical Details

On mac the image.Image returned should alwasy be tiff encoded.

On windows the image.Image returned will be the largest available image from the apps
ico set.  This can either be a PNG or a BMP.

## Todos

* [x] macOS support
* [x] windows support
* [ ] test
* [ ] benchmarks
* [ ] linux support?
