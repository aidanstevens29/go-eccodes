# go-eccodes

Go bindings for decoding [GRIB](https://en.wikipedia.org/wiki/GRIB) files with
[ECMWF ecCodes](https://github.com/ecmwf/eccodes).

The package currently provides a read-only, streaming API for GRIB1 and GRIB2
messages. It is modeled on ecCodes 2.48.0 and uses cgo to call the native
library.

## Development with Flox

Install [Flox](https://flox.dev/docs/install-flox/install), then activate the
environment committed to this repository:

```sh
flox activate
```

The environment provides the pinned Go toolchain, ecCodes library, C compiler,
`pkg-config`, and `golangci-lint`. Once activated, run the usual project
commands directly:

```sh
go test -race -count=1 ./...
golangci-lint run ./...
```

For a single command without entering an interactive shell, use:

```sh
flox activate -- go test ./...
```

## Reading a GRIB file

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"log"

	eccodes "github.com/aidanstevens29/go-eccodes"
)

func main() {
	reader, err := eccodes.Open("forecast.grib2")
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	for {
		message, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		shortName, err := message.String("shortName")
		if err != nil {
			message.Close()
			log.Fatal(err)
		}
		values, err := message.Doubles("values")
		if err != nil {
			message.Close()
			log.Fatal(err)
		}
		fmt.Printf("%s: %d values\n", shortName, len(values))

		if err := message.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
```

Each call to `Next` creates a native ecCodes handle. Close each `Message`
promptly; avoid deferring message closes inside a long loop.

## Working with messages

Metadata and decoded data are available through typed key accessors:

- `Long` and `Longs`
- `Double` and `Doubles`
- `String`
- `Size`

`Doubles("values")` returns the decoded key array in the message's native scan
order. Do not assume that independently fetched `latitudes`, `longitudes`, and
`values` arrays are index-aligned for every valid GRIB scanning mode.

Use `GeographicData` when coordinates and values must describe the same
points. It delegates to ecCodes' geographic iterator and returns three aligned
slices while honoring scanning modes such as alternating rows and column-major
scans. Once geometry has been obtained from one message, `GeographicValues`
returns only values in that same geographic ordering without allocating unused
coordinate slices:

```go
grid, err := message.GeographicData()
if err != nil {
	log.Fatal(err)
}
fmt.Printf("first point: %.4f, %.4f = %g\n",
	grid.Latitudes[0], grid.Longitudes[0], grid.Values[0])
```

`Bytes` returns an independent copy of the encoded GRIB message, and
`NewMessage` parses a complete encoded message already held in memory.

`RuntimeVersion` reports the linked native ecCodes version. `TargetVersion`
reports the upstream version targeted by these bindings.
