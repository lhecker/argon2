# argon2

[![](https://godoc.org/github.com/lhecker/argon2?status.svg)](https://godoc.org/github.com/lhecker/argon2)

The **fastest** _and_ **easiest** to use [Argon2](https://github.com/P-H-C/phc-winner-argon2) bindings for Go!

## Features

- Zero dependencies
- Easy to use API, including generation of raw and encoded hashes.
- Up to date & used in production environments.
- Contains Go-specific optimizations for a consistent **15%** performance boost.
- Allows you to enable all possible optimizations in Argon2, improving performance by up to **40%** in total!

## Usage

See [`examples/example.go`](https://github.com/lhecker/argon2/blob/master/examples/example.go) for a simple introduction and try it out with:

```bash
go run examples/example.go
```

## Performance

This library makes use of SSE, SSE2, SSSE3 and XOP, depending on whether they are enabled during compilation.
This can be done by adding appropriate `gcc` optimization flags to the `CGO_CFLAGS` environment variable.

Here's an example which you could set before running `go build` etc.:
```bash
export CGO_CFLAGS="-Ofast -funroll-loops -march=native"
```

In this example `-march=native` will optimize the program for the _current_ platform you're compiling on.
If you are planning to deploy this library in a different environment you should replace it with a matching value listed [here](https://gcc.gnu.org/onlinedocs/gcc/x86-Options.html).

This way you can achieve an performance improvement of up to 25%.
You can use this performance improvement as a free ticket for stronger hash settings and thus improved security at the same cost.

## Current downsides

This package uses `cgo` like all Go bindings and thus comes with all it's downsides:

- `cgo` makes cross-compilation hard.
- Can cause excessive spawning of native threads. ¹²

Due to the infinitely superior performance compared to a pure Go implementation I still personally believe that the benefits outweigh the drawbacks though.

¹
Even if `GOMAXPROCS` has been reached, the Goroutine scheduler will spawn another thread if all threads are already busy processing Goroutines and atleast one of those threads is stuck inside a `cgo` call.
That thread and its goroutine will then be taken out of the scheduler's thread pool and be replaced by a new thread.
The old thread on the other hand will be discarded as soon as the goroutine finishes.
As long as `Config.Parallelism` is 1 (which is the default) Argon2 will not spawn any additional threads internally though, keeping this overhead relatively modest.

²
If excessive thread spawning still turns out as a performance problem I recommend creating an old-fashioned worker pool by spawning some Goroutines (e.g. as many as CPU cores in the system) in each of which `runtime.LockOSThread()` is called at the beginning.
In an forever-loop you could then process requests for password hashing using channels from the outside.
Since those goroutines never return their threads will never be scavanged, allowing them to idle around in cgo as long as they want to.

## Modifications to Argon2

Based on [54ff100](https://github.com/P-H-C/phc-winner-argon2/tree/54ff100b0717505493439ec9d4ca85cb9cbdef00).

- Moved blake2 code into the root source directory and adjusted include paths to match this change.
- Merged `ref.c` and `opt.c` into one file (`ref_opt.c`). This allows us to use the `__SSE__` precompiler flag for SSE detection instead of relying on a Makefile.
