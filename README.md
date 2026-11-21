# sol-lib

`sol-lib` computes the sun's position and the day's sun events —
sunrise, sunset, and solar noon — in pure Go, from the equations
published by NOAA's solar calculator. It depends on the Go standard
library only.

## Install

```
go get github.com/zaephor-bot/sol-lib
```

The code lives in the module's `solar` package:

```
import "github.com/zaephor-bot/sol-lib/solar"
```

## Example

Sunrise, solar noon, and sunset for a mid-latitude spot, in local wall
clock. The snippet is the `Example` function from
[`example_test.go`](example_test.go) at the repo root, copied verbatim;
`go test ./...` runs it and checks the output lines below, so the code
you read here cannot rot — if the API or the times drift, the test
fails.

```go
package sol_lib_test

import (
	"fmt"
	"log"
	"time"

	"github.com/zaephor-bot/sol-lib/solar"
)

// Example is the exact code the README shows. go test runs it and checks
// the output below, so the README snippet cannot rot: if the API or the
// pinned times drift, this example fails. Keep the README's copy in sync
// with this function on every call, argument, and output line.
func Example() {
	local, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal(err)
	}
	day := time.Date(2026, time.March, 20, 0, 0, 0, 0, local) // spring equinox
	const lat, lon = 33.7455, -117.8677                       // Santa Ana, Orange County, CA

	sunrise, err := solar.Sunrise(day, lat, lon)
	if err != nil {
		log.Fatal(err)
	}
	noon, err := solar.SolarNoon(day, lat, lon)
	if err != nil {
		log.Fatal(err)
	}
	sunset, err := solar.Sunset(day, lat, lon)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("sunrise:   ", sunrise.Format("2006-01-02 15:04:05 MST"))
	fmt.Println("solar noon:", noon.Format("2006-01-02 15:04:05 MST"))
	fmt.Println("sunset:    ", sunset.Format("2006-01-02 15:04:05 MST"))
	// Output:
	// sunrise:    2026-03-20 06:54:54 PDT
	// solar noon: 2026-03-20 12:58:48 PDT
	// sunset:     2026-03-20 19:03:14 PDT
}
```

Latitudes are degrees positive north; longitudes are degrees positive
east. When the sun never rises or never sets on the requested day,
`Sunrise` and `Sunset` report `solar.ErrSunNeverRises` or
`solar.ErrSunNeverSets` rather than a bogus time.

## Accuracy

For mid-latitude locations — Orange County, Riverside, and the like —
sunrise, sunset, and solar noon land within **about two minutes** of
NOAA's own calculator. That is the whole claim: about two minutes on sun
events at mid-latitudes, and nothing tighter.

The library does **not** guarantee:

- **Polar latitudes.** Near and inside the polar circles the sun may
  never rise or never set on a given day; the event functions report
  `solar.ErrSunNeverRises` or `solar.ErrSunNeverSets` instead of a time,
  and no accuracy claim covers days close to that boundary.
- **Horizon and elevation effects.** Results assume a flat, unobstructed
  sea-level horizon at the standard −0.833° rise/set elevation. A high
  observing site or mountains on the horizon can shift the observed
  events by more than two minutes.
- **Refraction beyond NOAA's standard model.** Times fold in NOAA's
  standard atmospheric refraction correction; unusual conditions that
  refract differently (temperature inversions, extreme pressure) are not
  accounted for.

The calculations follow [NOAA's solar calculation details](https://gml.noaa.gov/grad/solcalc/calcdetails.html),
the page that documents the equations and conventions this library
implements.

## License

MIT — see [LICENSE](LICENSE).
