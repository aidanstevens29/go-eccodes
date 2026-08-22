# NBM GRIB test fixture

`nbm.t00z.core.f001.co.aptmp.grib2` is the first, unmodified GRIB message
extracted from `blend.t00z.core.f001.co.grib2`, an operational National Blend
of Models forecast file retrieved from NOAA's NOMADS archive:

<https://nomads.ncep.noaa.gov/pub/data/nccf/com/blend/prod/blend.20260822/00/core/blend.t00z.core.f001.co.grib2>

The source is the 2026-08-22 00Z NBM core forecast at a one-hour lead time. The
fixture is its real 2 m apparent-temperature field on the 2,345×1,597 NBM
Lambert grid, using complex spatial-differencing packing.

The 153 MB source file has SHA-256
`3d4c8e1131b4c16149a62690a41c6cdc3d8a84ff673f71bc27f79c0a67859ff7`.
The single-message fixture was extracted without repacking using ecCodes 2.26.0:

```sh
grib_copy -w count=1 blend.t00z.core.f001.co.grib2 \
  testdata/nbm.t00z.core.f001.co.aptmp.grib2
```

`nbm.t00z.core.f001.co.aptmp.expected.json` records the native ecCodes results,
including metadata, full-field statistics, selected decoded values, and the
fixture checksum. They were checked with:

```sh
grib_count testdata/nbm.t00z.core.f001.co.aptmp.grib2
grib_get -F %.15g \
  -p edition,shortName,typeOfLevel,level,dataDate,dataTime,stepRange,gridType,Nx,Ny,numberOfPoints,bitsPerValue,packingType,min,max,average,totalLength \
  testdata/nbm.t00z.core.f001.co.aptmp.grib2
```

The probe values were read directly through the native ecCodes API at indexes
listed in the expectation file. Updating this binary or its expectations is an
intentional golden-data change and should be reviewed against the full source.
