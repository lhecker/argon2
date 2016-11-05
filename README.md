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
- Can cause excessive spawning of native threads. ¹

Due to the infinitely superior performance compared to a pure Go implementation I still personally believe that the benefits outweigh the drawbacks though.

¹
Let's say there is a situation where one or more Goroutines are stuck in a `cgo` call and a new regular Goroutine is spawned.
If the scheduler can't find a suitable processor to run that Goroutine on, it will spawn a new processor.
Furthermore one of the Goroutines stuck in `cgo` is then marked as a "native" thread, which is destroyed as soon as the call is finished.
This downside has been mostly offset though by optimizing Argon2 to not spawn native threads as long as `Config.Parallelism` is 1 (which is the default).

## Modifications to Argon2

Based on [bc345e3](https://github.com/P-H-C/phc-winner-argon2/tree/bc345e3afb8ed1a26f3e41b2e778357bafea4a16).

- Moved blake2 code into the root source directory & adjusted include paths to match this.
- Merged `ref.{h,c}` and `opt.{h,c}` into one file (`ref_opt.{h,c}`) & adjusted include paths to match this. This allows us to use the `__SSE__` precompiler flag for SSE detection instead of relying on a Makefile.
- Optimized `core.c`'s `fill_memory_blocks` (see `core.diff`). This will now prevent spawning additional threads whenever `Config.Parallelism` is 1.
