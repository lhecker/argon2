# argon2

[![](https://godoc.org/github.com/lhecker/argon2?status.svg)](https://godoc.org/github.com/lhecker/argon2)

~~The **fastest** _and_ **easiest** to use [Argon2](https://github.com/P-H-C/phc-winner-argon2) bindings for Go!~~

## ⚠️ Notice ⚠️

**In general I recommend using [`github.com/matthewhartstonge/argon2`](https://github.com/matthewhartstonge/argon2) for now**, for the reasons explained below.<br>
It has the the _exact same_ API as this project and can be used as a drop in replacement.

If you do want to use this project please first download it on one of the actual machines you plan to deploy this project on and then run:
```sh
CGO_CFLAGS="-O3 -march=native" go test -run="^$" -bench=BenchmarkHash
```

You can adjust the `Config` used for benchmarking [here](https://github.com/lhecker/argon2/blob/master/argon2_test.go#L17-L25).<br>
If `BenchmarkHash` is slower or not significantly enough faster than `BenchmarkHashXCryptoArgon2` I recommend checking out the alternative project above.

While you should actually still find that this project indeed is "up to twice as fast" as other projects (including those based on `golang.org/x/crypto/argon2`) on Linux and macOS on modern bare metal hardware, the primary issue is that this performance advantage cannot be reliably replicated when being used in any VMs, including those used by popular Cloud Providers.<br>
I've failed to find a good enough explaination for this performance discrepancy between bare metal and virtualized hardware within a reasonable time frame and thus recommend the library above for now.

## Features

- Zero dependencies
- Easy to use API, including generation of raw and encoded hashes
- Up to date & used in production environments
- _Up to twice_ as fast as `golang.org/x/crypto/argon2`, allowing you to apply more secure settings while keeping the same latency

## Usage

See [`examples/example.go`](https://github.com/lhecker/argon2/blob/master/examples/example.go) for a simple introduction and try it out with:

```bash
go run examples/example.go
```

## Performance

This library makes use of AVX/SSE, depending on whether they are enabled during compilation.
This can be done by adding appropriate `gcc` optimization flags to the `CGO_CFLAGS` environment variable.

Here's an example which you could set before running `go build` etc.:
```bash
export CGO_CFLAGS="-O3 -march=native"
```

In this example `-march=native` will optimize the program for the _current_ platform you're compiling on.
If you're planning to deploy this library in a different environment you should replace it with a matching value listed [here](https://gcc.gnu.org/onlinedocs/gcc/x86-Options.html).

This way you can achieve a significant performance improvement.
You can use this performance improvement to apply stronger hash settings and thus improve security at the same cost.

## Current downsides

This package uses `cgo` like all Go bindings and thus comes with all it's downsides. Among others:

- `cgo` makes cross-compilation hard
- Excessive thread spawning¹

¹
Almost every time this library hashes something the scheduler will notice that a Goroutine is blocked in a cgo call and will spawn a new, costly, native thread.
To prevent this you may use my [workerpool](https://github.com/lhecker/workerpool) project to set up a worker pool like [this](https://github.com/lhecker/workerpool/blob/026271cb185e1421ed2a032d5bfad85589585703/workerpool_test.go#L68-L71).

## Modifications to Argon2

Based on [fba7b9a](https://github.com/P-H-C/phc-winner-argon2/tree/fba7b9a73a1bb913f49fadf6126f6e6b352d2fda).

- Moved blake2 code into the root source directory and adjusted include paths to match this change.
- Merged `ref.c` and `opt.c` into one file (`ref_opt.c`). This allows us to use the `__SSE__` precompiler flag for SSE detection instead of relying on a Makefile.
