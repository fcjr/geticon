# geticon-go

A tiny cGo library to get app icons of running applications.

## Usage

```go
icon, err := geticon.FromPid(pid) // returns an image.Image
```

## Todos

[x] macOS support
[ ] windows support
[ ] memory improvement for macOS (don't allocate an extra large buffer)
[ ] test
[ ] benchmarks
[ ] linux support?
