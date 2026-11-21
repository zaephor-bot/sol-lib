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
