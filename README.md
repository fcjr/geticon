# geticon

A tiny cross-plaform (macOS + windows) slibrary to get app icons of running applications.

## Installation

```sh
go get github.com/fcjr/geticon
```

## Usage

```go
import (
    "github.com/fcjr/geticon"
)

icon, err := geticon.FromPid(pid) // returns an image.Image
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
