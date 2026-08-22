# go-eccodes

Go bindings for decoding [GRIB](https://en.wikipedia.org/wiki/GRIB) files with
[ECMWF ecCodes](https://github.com/ecmwf/eccodes).

The package currently provides a read-only, streaming API for GRIB1 and GRIB2
messages. It is modeled on ecCodes 2.48.0 and uses cgo to call the native
library.

## Requirements

- Go 1.26 or newer with cgo enabled
- ecCodes 2.48.0 and `pkg-config`
- A C compiler

On macOS with Homebrew:

```sh
brew install eccodes pkg-config
```

On Debian or Ubuntu:

```sh
sudo apt-get install libeccodes-dev pkg-config
```

For Conda installations, activate the environment and expose its pkg-config
metadata if necessary:

```sh
export PKG_CONFIG_PATH="$CONDA_PREFIX/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
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

`Doubles("values")` returns the decoded grid-point values. `Bytes` returns an
independent copy of the encoded GRIB message, and `NewMessage` parses a complete
encoded message already held in memory.

`RuntimeVersion` reports the linked native ecCodes version. `TargetVersion`
reports the upstream version targeted by these bindings.
