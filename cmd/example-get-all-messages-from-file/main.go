package main

import (
	"flag"
	codes "github.com/aidanstevens29/go-eccodes"
	"io"
	"log"
	"runtime/debug"
	"time"

	"github.com/aidanstevens29/go-eccodes/native"
	"github.com/pkg/errors"

	cio "github.com/aidanstevens29/go-eccodes/io"
)

func main() {
	filename := flag.String("file", "", "io path, e.g. /tmp/ARPEGE_0.1_SP1_00H12H_201709290000.grib2")

	flag.Parse()

	f, err := cio.OpenFile(*filename, "r")
	if err != nil {
		log.Fatalf("failed to open file on file system: %s", err.Error())
	}
	defer f.Close()

	file, err := codes.OpenFile(f, native.ProductAny)
	if err != nil {
		log.Fatalf("failed to open file: %s", err.Error())
	}
	defer file.Close()

	n := 0
	for {
		err = process(file, n)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("failed to get message (#%d) from index: %s", n, err.Error())
		}
		n++
	}
}

func process(file codes.File, n int) error {
	start := time.Now()

	msg, err := file.Next()
	if err != nil {
		return err
	}
	defer msg.Close()

	log.Printf("============= BEGIN MESSAGE N%d ==========\n", n)

	shortName, err := msg.GetString("shortName")
	if err != nil {
		return errors.Wrap(err, "failed to get 'shortName' value")
	}
	name, err := msg.GetString("name")
	if err != nil {
		return errors.Wrap(err, "failed to get 'name' value")
	}
	forecastTime, err := msg.GetString("forecastTime")
	if err != nil {
		log.Printf("failed to get 'forecastTime' value: %v\n", err)
	}

	log.Printf("Variable = [%s](%s), forecastTime=%s\n", shortName, name, forecastTime)

	// just to measure timing
	lat, lon, values, err := msg.Data()
	if err != nil {
		return errors.Wrap(err, "failed to get data (latitudes, longitudes, values)")
	}

	log.Printf("Lengths of slices: %d %d %d\n", len(lat), len(lon), len(values))

	log.Printf("elapsed=%.0f ms", time.Since(start).Seconds()*1000)
	log.Printf("============= END MESSAGE N%d ============\n\n", n)

	debug.FreeOSMemory()

	return nil
}
