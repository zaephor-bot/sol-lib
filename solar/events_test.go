package solar

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"
)

// Event reference values were read off NOAA's solar calculator
// (https://gml.noaa.gov/grad/solcalc/) for the pinned inputs and retrieved
// on 2026-11-19. Tolerance is the two-minute bar from the project brief,
// which also absorbs NOAA's published-minute rounding.

const eventTolerance = 2 * time.Minute

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return loc
}

func withinTolerance(t *testing.T, label string, got, want time.Time) {
	t.Helper()
	if diff := got.Sub(want); math.Abs(float64(diff)) > float64(eventTolerance) {
		t.Errorf("%s = %s, want %s (±%s)", label, got.Format(time.RFC3339), want.Format(time.RFC3339), eventTolerance)
	}
}

// TestEventsPinnedNOAA pins sunrise, sunset, and solar noon for Orange
// County and Riverside across the 2024 equinoxes and solstices, so the
// seasonal extremes are exercised.
func TestEventsPinnedNOAA(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	cases := []struct {
		name                 string
		lat, lon             float64
		date                 time.Time
		sunrise, noon, unset string // wall-clock times on the pinned date
	}{
		// Orange County (Santa Ana), 33.7455 N, 117.8677 W.
		{"Orange County 2024-03-20 (March equinox)", 33.7455, -117.8677, time.Date(2024, time.March, 20, 0, 0, 0, 0, la), "06:54:15", "12:58:40", "19:03:36"},
		{"Orange County 2024-06-20 (June solstice)", 33.7455, -117.8677, time.Date(2024, time.June, 20, 0, 0, 0, 0, la), "05:41:16", "12:53:15", "20:05:14"},
		{"Orange County 2024-09-22 (September equinox)", 33.7455, -117.8677, time.Date(2024, time.September, 22, 0, 0, 0, 0, la), "06:40:03", "12:43:55", "18:47:16"},
		{"Orange County 2024-12-21 (December solstice)", 33.7455, -117.8677, time.Date(2024, time.December, 21, 0, 0, 0, 0, la), "06:52:39", "11:49:57", "16:47:16"},
		// Riverside, 33.9533 N, 117.3962 W.
		{"Riverside 2024-03-20 (March equinox)", 33.9533, -117.3962, time.Date(2024, time.March, 20, 0, 0, 0, 0, la), "06:52:21", "12:56:47", "19:01:44"},
		{"Riverside 2024-06-20 (June solstice)", 33.9533, -117.3962, time.Date(2024, time.June, 20, 0, 0, 0, 0, la), "05:38:49", "12:51:22", "20:03:55"},
		{"Riverside 2024-09-22 (September equinox)", 33.9533, -117.3962, time.Date(2024, time.September, 22, 0, 0, 0, 0, la), "06:38:09", "12:42:02", "18:45:24"},
		{"Riverside 2024-12-21 (December solstice)", 33.9533, -117.3962, time.Date(2024, time.December, 21, 0, 0, 0, 0, la), "06:51:17", "11:48:04", "16:44:51"},
	}

	for _, c := range cases {
		rise, err := Sunrise(c.date, c.lat, c.lon)
		if err != nil {
			t.Errorf("%s: Sunrise: %v", c.name, err)
			continue
		}
		noon, err := SolarNoon(c.date, c.lat, c.lon)
		if err != nil {
			t.Errorf("%s: SolarNoon: %v", c.name, err)
			continue
		}
		set, err := Sunset(c.date, c.lat, c.lon)
		if err != nil {
			t.Errorf("%s: Sunset: %v", c.name, err)
			continue
		}

		wantRise := parseWall(t, c.date, c.sunrise)
		wantNoon := parseWall(t, c.date, c.noon)
		wantSet := parseWall(t, c.date, c.unset)
		withinTolerance(t, c.name+" sunrise", rise, wantRise)
		withinTolerance(t, c.name+" noon", noon, wantNoon)
		withinTolerance(t, c.name+" sunset", set, wantSet)

		if rise.Location() != c.date.Location() || noon.Location() != c.date.Location() || set.Location() != c.date.Location() {
			t.Errorf("%s: results not in the requested location", c.name)
		}
	}
}

// TestEventsDSTTransitionDays pins the two 2024 DST transition days in
// Los Angeles. The wall clock is built from wall components through
// time.Date, so these days must resolve to the correct offset at the
// event instant (PDT on 2024-03-10, PST on 2024-11-03), not to a duration
// added across the one-hour discontinuity.
func TestEventsDSTTransitionDays(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	cases := []struct {
		name                 string
		date                 time.Time
		sunrise, noon, unset string
		zone                 string
	}{
		{"Los Angeles 2024-03-10 (spring forward)", time.Date(2024, time.March, 10, 0, 0, 0, 0, la), "07:09:14", "13:03:00", "18:57:18", "PDT"},
		{"Los Angeles 2024-11-03 (fall back)", time.Date(2024, time.November, 3, 0, 0, 0, 0, la), "06:14:58", "11:36:29", "16:57:37", "PST"},
	}
	for _, c := range cases {
		rise, err := Sunrise(c.date, 34.0522, -118.2437)
		if err != nil {
			t.Fatalf("%s: Sunrise: %v", c.name, err)
		}
		noon, err := SolarNoon(c.date, 34.0522, -118.2437)
		if err != nil {
			t.Fatalf("%s: SolarNoon: %v", c.name, err)
		}
		set, err := Sunset(c.date, 34.0522, -118.2437)
		if err != nil {
			t.Fatalf("%s: Sunset: %v", c.name, err)
		}

		for _, ev := range []struct {
			label string
			at    time.Time
			want  string
		}{
			{"sunrise", rise, c.sunrise},
			{"noon", noon, c.noon},
			{"sunset", set, c.unset},
		} {
			if zone, _ := ev.at.Zone(); zone != c.zone {
				t.Errorf("%s %s: zone = %s, want %s", c.name, ev.label, zone, c.zone)
			}
			withinTolerance(t, c.name+" "+ev.label, ev.at, parseWall(t, c.date, ev.want))
		}
	}
}

// TestPolarJuneNeverSets: Utqiaġvik (Barrow), 71.2906 N, in June the sun
// never sets; the library must say so instead of returning a time.
func TestPolarJuneNeverSets(t *testing.T) {
	ak := mustLocation(t, "America/Anchorage")
	date := time.Date(2024, time.June, 21, 0, 0, 0, 0, ak)
	const lat, lon = 71.2906, -156.7886

	if _, err := Sunset(date, lat, lon); !errors.Is(err, ErrSunNeverSets) {
		t.Errorf("June sunset error = %v, want ErrSunNeverSets", err)
	}
	if _, err := Sunrise(date, lat, lon); !errors.Is(err, ErrSunNeverSets) {
		t.Errorf("June sunrise error = %v, want ErrSunNeverSets (the sun is already up)", err)
	}

	// The transit still happens and is pinned (NOAA calculator, retrieved
	// 2026-11-19).
	noon, err := SolarNoon(date, lat, lon)
	if err != nil {
		t.Fatalf("June solar noon: %v", err)
	}
	withinTolerance(t, "June solar noon", noon, parseWall(t, date, "14:29:10"))
}

// TestPolarDecemberNeverRises: Utqiaġvik (Barrow) in December the sun
// never rises; the library must say so instead of returning a time.
func TestPolarDecemberNeverRises(t *testing.T) {
	ak := mustLocation(t, "America/Anchorage")
	date := time.Date(2024, time.December, 21, 0, 0, 0, 0, ak)
	const lat, lon = 71.2906, -156.7886

	if _, err := Sunrise(date, lat, lon); !errors.Is(err, ErrSunNeverRises) {
		t.Errorf("December sunrise error = %v, want ErrSunNeverRises", err)
	}
	if _, err := Sunset(date, lat, lon); !errors.Is(err, ErrSunNeverRises) {
		t.Errorf("December sunset error = %v, want ErrSunNeverRises (the sun never came up)", err)
	}

	noon, err := SolarNoon(date, lat, lon)
	if err != nil {
		t.Fatalf("December solar noon: %v", err)
	}
	withinTolerance(t, "December solar noon", noon, parseWall(t, date, "13:25:42"))
}

// TestArcticCircleEdge exercises days just inside and outside the
// midnight-sun boundary at the Arctic Circle latitude.
func TestArcticCircleEdge(t *testing.T) {
	date := time.Date(2024, time.June, 21, 0, 0, 0, 0, time.UTC)
	// 65.0 N on the June solstice: a short night remains. (The midnight
	// sun reaches down to about 65.7 N with the -0.833° horizon.)
	if _, err := Sunset(date, 65.0, 0.0); err != nil {
		t.Errorf("65.0 N June sunset: unexpected error %v", err)
	}
	// 67.5 N on the June solstice: the sun never sets.
	if _, err := Sunset(date, 67.5, 0.0); !errors.Is(err, ErrSunNeverSets) {
		t.Errorf("67.5 N June sunset error = %v, want ErrSunNeverSets", err)
	}
	// 67.5 N on the December solstice: the sun never rises.
	dec := time.Date(2024, time.December, 21, 0, 0, 0, 0, time.UTC)
	if _, err := Sunrise(dec, 67.5, 0.0); !errors.Is(err, ErrSunNeverRises) {
		t.Errorf("67.5 N December sunrise error = %v, want ErrSunNeverRises", err)
	}
}

// TestEventsReachHorizon: at the returned sunrise and sunset instants the
// sun's geometric elevation must sit at the −0.833° horizon, tying the
// event formulas to the position engine.
func TestEventsReachHorizon(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	cases := []struct {
		lat, lon float64
		date     time.Time
	}{
		{33.7455, -117.8677, time.Date(2024, time.March, 20, 0, 0, 0, 0, la)},
		{33.7455, -117.8677, time.Date(2024, time.June, 20, 0, 0, 0, 0, la)},
		{33.7455, -117.8677, time.Date(2024, time.December, 21, 0, 0, 0, 0, la)},
		{33.9533, -117.3962, time.Date(2024, time.June, 20, 0, 0, 0, 0, la)},
		{34.0522, -118.2437, time.Date(2024, time.March, 10, 0, 0, 0, 0, la)},
		{34.0522, -118.2437, time.Date(2024, time.November, 3, 0, 0, 0, 0, la)},
		{51.4779, -0.0015, time.Date(2024, time.September, 22, 0, 0, 0, 0, time.UTC)},
		{-33.8688, 151.2093, time.Date(2024, time.December, 21, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		for _, rise := range []bool{true, false} {
			at, err := riseSet(c.date, c.lat, c.lon, rise)
			if err != nil {
				t.Fatalf("%.4f,%.4f %s: %v", c.lat, c.lon, c.date, err)
			}
			elev := sunAnglesAt(at, c.lat, c.lon).elevation
			if math.Abs(elev-(-0.833)) > 0.15 {
				t.Errorf("%.4f,%.4f %s %s: geometric elevation %.3f, want -0.833 (±0.15)",
					c.lat, c.lon, c.date.Format("2006-01-02"), map[bool]string{true: "sunrise", false: "sunset"}[rise], elev)
			}
		}
	}
}

// TestSolarNoonIsTransit: the sun's elevation must peak within a minute
// of the returned solar noon.
func TestSolarNoonIsTransit(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	cases := []struct {
		lat, lon float64
		date     time.Time
	}{
		{33.7455, -117.8677, time.Date(2024, time.March, 20, 0, 0, 0, 0, la)},
		{33.7455, -117.8677, time.Date(2024, time.June, 20, 0, 0, 0, 0, la)},
		{33.9533, -117.3962, time.Date(2024, time.December, 21, 0, 0, 0, 0, la)},
		{71.2906, -156.7886, time.Date(2024, time.June, 21, 0, 0, 0, 0, mustLocation(t, "America/Anchorage"))},
	}
	for _, c := range cases {
		noon, err := SolarNoon(c.date, c.lat, c.lon)
		if err != nil {
			t.Fatalf("solar noon: %v", err)
		}

		best, bestAt := -91.0, noon
		for m := -15 * time.Minute; m <= 15*time.Minute; m += 10 * time.Second {
			at := noon.Add(m)
			if e := sunAnglesAt(at, c.lat, c.lon).elevation; e > best {
				best, bestAt = e, at
			}
		}
		if d := bestAt.Sub(noon); math.Abs(float64(d)) > float64(time.Minute) {
			t.Errorf("%.4f,%.4f %s: elevation peaks %s from solar noon (want ≤1m)",
				c.lat, c.lon, c.date.Format("2006-01-02"), d.Truncate(time.Second))
		}
	}
}

// TestEquationOfTimeAnchors2024 pins Greenwich transits on the days of the
// 2024 equation-of-time extremes (−14m15s around February 11, +16m25s
// around November 3, zero around April 15), external anchors from
// astronomical tables: noon = 12:00 − EoT.
func TestEquationOfTimeAnchors2024(t *testing.T) {
	cases := []struct {
		name string
		date time.Time
		want string
	}{
		{"Greenwich 2024-02-11 (EoT ≈ −14¼ min)", time.Date(2024, time.February, 11, 0, 0, 0, 0, time.UTC), "12:14:14"},
		{"Greenwich 2024-04-15 (EoT ≈ 0)", time.Date(2024, time.April, 15, 0, 0, 0, 0, time.UTC), "11:59:54"},
		{"Greenwich 2024-11-03 (EoT ≈ +16½ min)", time.Date(2024, time.November, 3, 0, 0, 0, 0, time.UTC), "11:43:31"},
	}
	for _, c := range cases {
		noon, err := SolarNoon(c.date, 51.4779, -0.0015)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		withinTolerance(t, c.name+" noon", noon, parseWall(t, c.date, c.want))
	}
}

// TestAdjacentDayConvention pins the documented convention for an extreme
// longitude paired with a mismatched offset: at 179.9° E with a UTC clock,
// the 2024-03-20 sunrise happens on the previous civil day, and NOAA's
// calculator reports it there (2024-03-19 18:04; retrieved 2026-11-19).
func TestAdjacentDayConvention(t *testing.T) {
	date := time.Date(2024, time.March, 20, 0, 0, 0, 0, time.UTC)
	rise, err := Sunrise(date, 40.0, 179.9)
	if err != nil {
		t.Fatalf("sunrise: %v", err)
	}
	withinTolerance(t, "179.9°E sunrise", rise, time.Date(2024, time.March, 19, 18, 4, 4, 0, time.UTC))

	set, err := Sunset(date, 40.0, 179.9)
	if err != nil {
		t.Fatalf("sunset: %v", err)
	}
	withinTolerance(t, "179.9°E sunset", set, time.Date(2024, time.March, 20, 6, 12, 17, 0, time.UTC))
}

// TestLocalCalendarDaySemantics: the same instant read in two locations
// selects two different local days: 2024-03-20 18:00 UT is March 21 in
// Tokyo but March 20 in UTC. The extreme east longitude with a UTC clock
// also demonstrates the adjacent-day convention. (Values from NOAA's
// calculator, retrieved 2026-11-19.)
func TestLocalCalendarDaySemantics(t *testing.T) {
	tokyo := mustLocation(t, "Asia/Tokyo")
	instant := time.Date(2024, time.March, 20, 18, 0, 0, 0, time.UTC)

	tokyoDay := instant.In(tokyo) // 2024-03-21 03:00 JST
	rise, err := Sunrise(tokyoDay, 35.6762, 139.6503)
	if err != nil {
		t.Fatalf("Tokyo sunrise: %v", err)
	}
	if rise.Location() != tokyo {
		t.Errorf("Tokyo sunrise location = %v, want Asia/Tokyo", rise.Location())
	}
	withinTolerance(t, "Tokyo 2024-03-21 sunrise", rise, time.Date(2024, time.March, 21, 5, 43, 39, 0, tokyo))

	utcDay := instant.In(time.UTC) // 2024-03-20 18:00 UTC
	riseUTC, err := Sunrise(utcDay, 35.6762, 139.6503)
	if err != nil {
		t.Fatalf("UTC-day sunrise: %v", err)
	}
	// 139.65° E pairs badly with a UTC clock: the March 20 sunrise is
	// reported on March 19 in UTC wall time, as NOAA's calculator does.
	withinTolerance(t, "UTC-day sunrise", riseUTC, time.Date(2024, time.March, 19, 20, 45, 5, 0, time.UTC))
}

// parseWall builds a wall-clock time from "HH:MM:SS" on date's calendar
// day. It assembles wall components through time.Date (not midnight plus
// a duration) so DST transition days resolve like the library does.
func parseWall(t *testing.T, date time.Time, hhmmss string) time.Time {
	t.Helper()
	var hh, mm, ss int
	if _, err := fmt.Sscanf(hhmmss, "%2d:%2d:%2d", &hh, &mm, &ss); err != nil {
		t.Fatalf("parse %q: %v", hhmmss, err)
	}
	y, m, d := date.Date()
	return time.Date(y, m, d, hh, mm, ss, 0, date.Location())
}
