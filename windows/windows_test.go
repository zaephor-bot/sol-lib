package windows

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/zaephor-bot/sol-lib/solar"
)

// Window reference values were worked out in advance from the
// Astronomical Almanac's low-precision solar tables (an implementation
// independent of the engine's NOAA equations), then cross-checked
// against NOAA's calculator's published sunrise and sunset for the same
// days, which those tables reproduce within two seconds. The allowed
// error below covers the independent tables' own accuracy plus the
// engine's reported two-minute mid-latitude budget.

const windowTolerance = 2 * time.Minute

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
	if diff := got.Sub(want); math.Abs(float64(diff)) > float64(windowTolerance) {
		t.Errorf("%s = %s, want %s (±%s)", label, got.Format(time.RFC3339), want.Format(time.RFC3339), windowTolerance)
	}
}

// parseWall builds a wall-clock time from "HH:MM:SS" on date's calendar
// day, assembled from wall components through time.Date so DST
// transition days resolve like the library does.
func parseWall(t *testing.T, date time.Time, hhmmss string) time.Time {
	t.Helper()
	var hh, mm, ss int
	if _, err := fmt.Sscanf(hhmmss, "%2d:%2d:%2d", &hh, &mm, &ss); err != nil {
		t.Fatalf("parse %q: %v", hhmmss, err)
	}
	y, m, d := date.Date()
	return time.Date(y, m, d, hh, mm, ss, 0, date.Location())
}

// TestWindowsPinnedReferences pins all six windows for two date and
// location pairs — Orange County on the 2024 March equinox and
// Riverside on the 2024 June solstice — against reference times worked
// out in advance (see the comment above windowTolerance).
func TestWindowsPinnedReferences(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	cases := []struct {
		name     string
		lat, lon float64
		date     time.Time
		// start/end wall-clock times of each window on the pinned
		// date, in All's fixed order.
		times [6][2]string
	}{
		{"Orange County 2024-03-20 (March equinox)", 33.7455, -117.8677,
			time.Date(2024, time.March, 20, 0, 0, 0, 0, la),
			[6][2]string{
				{"06:29:09", "06:52:52"}, // dawn civil twilight
				{"06:29:09", "06:38:39"}, // dawn blue hour
				{"06:52:51", "07:26:27"}, // dawn golden hour
				{"18:31:24", "19:05:03"}, // dusk golden hour
				{"19:19:16", "19:28:48"}, // dusk blue hour
				{"19:05:03", "19:28:48"}, // dusk civil twilight
			}},
		{"Riverside 2024-06-20 (June solstice)", 33.9533, -117.3962,
			time.Date(2024, time.June, 20, 0, 0, 0, 0, la),
			[6][2]string{
				{"05:09:18", "05:37:12"}, // dawn civil twilight
				{"05:09:18", "05:20:37"}, // dawn blue hour
				{"05:37:12", "06:15:00"}, // dawn golden hour
				{"19:27:43", "20:05:31"}, // dusk golden hour
				{"20:22:06", "20:33:25"}, // dusk blue hour
				{"20:05:31", "20:33:25"}, // dusk civil twilight
			}},
	}

	for _, c := range cases {
		ws := All(c.date, c.lat, c.lon)
		if len(ws) != 6 {
			t.Fatalf("%s: All returned %d windows, want 6", c.name, len(ws))
		}
		for i, w := range ws {
			if w.Absent {
				t.Errorf("%s: %s reported absent, want present", c.name, w.Event)
				continue
			}
			if w.Start.Location() != c.date.Location() || w.End.Location() != c.date.Location() {
				t.Errorf("%s: %s not in the requested location", c.name, w.Event)
			}
			withinTolerance(t, c.name+" "+w.Event.String()+" start", w.Start, parseWall(t, c.date, c.times[i][0]))
			withinTolerance(t, c.name+" "+w.Event.String()+" end", w.End, parseWall(t, c.date, c.times[i][1]))
		}
	}
}

// TestGoldenHourTwiceOnOrdinaryDay: on an ordinary mid-latitude day the
// library reports golden hour twice, once while the sun rises and once
// while it sets, and All's six windows share crossings where the
// definitions share a threshold edge.
func TestGoldenHourTwiceOnOrdinaryDay(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	date := time.Date(2024, time.September, 22, 0, 0, 0, 0, la)
	const lat, lon = 33.7455, -117.8677

	dawn, dusk := GoldenHour(date, lat, lon)
	for _, w := range []Window{dawn, dusk} {
		if w.Absent {
			t.Fatalf("%s reported absent on an ordinary day", w.Event)
		}
		if d := w.End.Sub(w.Start); d < 15*time.Minute || d > 3*time.Hour {
			t.Errorf("%s duration %s, want between 15m and 3h", w.Event, d)
		}
	}
	if dawn.Event != DawnGoldenHour || dusk.Event != DuskGoldenHour {
		t.Errorf("golden hour events = %s, %s, want dawn golden hour, dusk golden hour", dawn.Event, dusk.Event)
	}
	if !(dawn.Start.Before(dawn.End) && dawn.End.Before(dusk.Start) && dusk.Start.Before(dusk.End)) {
		t.Errorf("golden hours out of order: dawn %s..%s, dusk %s..%s",
			dawn.Start, dawn.End, dusk.Start, dusk.End)
	}

	// All reports the same two golden hours plus the four other
	// windows, in the documented fixed order, with shared threshold
	// edges landing on identical crossings.
	wantOrder := []Event{DawnCivilTwilight, DawnBlueHour, DawnGoldenHour, DuskGoldenHour, DuskBlueHour, DuskCivilTwilight}
	ws := All(date, lat, lon)
	for i, w := range ws {
		if w.Event != wantOrder[i] {
			t.Fatalf("All order: window %d is %s, want %s", i, w.Event, wantOrder[i])
		}
		if w.Absent {
			t.Fatalf("%s reported absent on an ordinary day", w.Event)
		}
	}
	if !ws[2].Start.Equal(dawn.Start) || !ws[3].Start.Equal(dusk.Start) {
		t.Errorf("All's golden hours disagree with GoldenHour's")
	}
	// Shared edges: civil twilight and blue hour open together at the
	// −6° crossing; civil twilight hands off to golden hour at the
	// horizon crossing.
	if !ws[0].Start.Equal(ws[1].Start) {
		t.Errorf("dawn civil twilight and blue hour start apart: %s vs %s", ws[0].Start, ws[1].Start)
	}
	if !ws[0].End.Equal(ws[2].Start) {
		t.Errorf("dawn civil twilight does not hand off to golden hour: %s vs %s", ws[0].End, ws[2].Start)
	}
	if !ws[3].End.Equal(ws[5].Start) {
		t.Errorf("dusk golden hour does not hand off to civil twilight: %s vs %s", ws[3].End, ws[5].Start)
	}
	if !ws[4].End.Equal(ws[5].End) {
		t.Errorf("dusk blue hour and civil twilight end apart: %s vs %s", ws[4].End, ws[5].End)
	}
}

// TestPolarDayAndNightAllWindowsAbsent: Utqiaġvik (Barrow), 71.2906° N,
// in June the sun never drops to any window's lower edge, and in
// December it never climbs to any window's upper edge. Every window
// must say it is absent, and no window — absent or not — may have equal
// start and end.
func TestPolarDayAndNightAllWindowsAbsent(t *testing.T) {
	ak := mustLocation(t, "America/Anchorage")
	cases := []struct {
		name string
		date time.Time
	}{
		{"Utqiaġvik 2024-06-21 (midnight sun)", time.Date(2024, time.June, 21, 0, 0, 0, 0, ak)},
		{"Utqiaġvik 2024-12-21 (polar night)", time.Date(2024, time.December, 21, 0, 0, 0, 0, ak)},
	}
	const lat, lon = 71.2906, -156.7886

	for _, c := range cases {
		for _, w := range All(c.date, lat, lon) {
			if !w.Absent {
				t.Errorf("%s: %s = %s..%s, want absent", c.name, w.Event, w.Start, w.End)
			}
			if w.Start.Equal(w.End) {
				t.Errorf("%s: %s is zero-length (%s..%s)", c.name, w.Event, w.Start, w.End)
			}
			if !w.Start.Before(w.End) {
				t.Errorf("%s: %s bounds out of order: %s..%s", c.name, w.Event, w.Start, w.End)
			}
			if w.Start.Location() != c.date.Location() {
				t.Errorf("%s: %s not in the requested location", c.name, w.Event)
			}
		}
	}
}

// TestShortWindowsNearBoundaries: minutes-scale windows must survive
// the crossing scan. Just inside the Arctic Circle at the June
// solstice the sun spends only minutes below the horizon, squeezed
// between the dusk and dawn golden hours; at the equator the blue hour
// band is crossed in minutes by the steep morning climb.
func TestShortWindowsNearBoundaries(t *testing.T) {
	// 65.35° N, 0° E on the June solstice: golden hour happens twice,
	// and the night between the dusk golden hour's end and the next
	// dawn golden hour's start collapses to well under an hour. Blue
	// hour and civil twilight never happen: the sun never reaches −4°.
	date := time.Date(2024, time.June, 21, 0, 0, 0, 0, time.UTC)
	const lat, lon = 65.35, 0.0
	dayStart := time.Date(2024, time.June, 21, 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	dawn, dusk := GoldenHour(date, lat, lon)
	if dawn.Absent || dusk.Absent {
		t.Fatalf("65.35°N June solstice golden hours: dawn absent=%v, dusk absent=%v", dawn.Absent, dusk.Absent)
	}
	night := dawn.Start.Sub(dayStart) + dayEnd.Sub(dusk.End)
	if night <= 0 || night >= time.Hour {
		t.Errorf("65.35°N June solstice night = %s, want positive and under an hour", night)
	}

	dawnBlue, duskBlue := BlueHour(date, lat, lon)
	if !dawnBlue.Absent || !duskBlue.Absent {
		t.Errorf("65.35°N June solstice blue hours: dawn absent=%v, dusk absent=%v, want both absent", dawnBlue.Absent, duskBlue.Absent)
	}
	dawnCivil, duskCivil := CivilTwilight(date, lat, lon)
	if !dawnCivil.Absent || !duskCivil.Absent {
		t.Errorf("65.35°N June solstice civil twilights: dawn absent=%v, dusk absent=%v, want both absent", dawnCivil.Absent, duskCivil.Absent)
	}

	// Singapore on the March equinox: the sun climbs the −6…−4° band
	// in about eight minutes.
	sg := time.Date(2024, time.March, 20, 0, 0, 0, 0, time.UTC)
	sgDawn, sgDusk := BlueHour(sg, 1.3521, 103.8198)
	for _, w := range []Window{sgDawn, sgDusk} {
		if w.Absent {
			t.Fatalf("Singapore blue hour %s reported absent", w.Event)
		}
		if d := w.End.Sub(w.Start); d < 5*time.Minute || d > 12*time.Minute {
			t.Errorf("Singapore %s duration = %s, want 5–12 minutes", w.Event, d)
		}
	}
}

// TestWindowEdgesAtThreshold ties every present window's edges to the
// engine's elevation: at a window's start and end the reported apparent
// elevation must sit at the threshold the definition names. Absent
// windows must carry unequal bounds everywhere.
func TestWindowEdgesAtThreshold(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	// start/end threshold elevations keyed by event.
	thresholds := map[Event][2]float64{
		DawnGoldenHour:    {horizonElevation, goldenElevation},
		DuskGoldenHour:    {goldenElevation, horizonElevation},
		DawnBlueHour:      {twilightElevation, blueShallowElevation},
		DuskBlueHour:      {blueShallowElevation, twilightElevation},
		DawnCivilTwilight: {twilightElevation, horizonElevation},
		DuskCivilTwilight: {horizonElevation, twilightElevation},
	}
	cases := []struct {
		lat, lon float64
		date     time.Time
	}{
		{33.7455, -117.8677, time.Date(2024, time.March, 20, 0, 0, 0, 0, la)},
		{33.7455, -117.8677, time.Date(2024, time.June, 20, 0, 0, 0, 0, la)},
		{33.7455, -117.8677, time.Date(2024, time.December, 21, 0, 0, 0, 0, la)},
		{1.3521, 103.8198, time.Date(2024, time.March, 20, 0, 0, 0, 0, time.UTC)},
		{-33.8688, 151.2093, time.Date(2024, time.December, 21, 0, 0, 0, 0, time.UTC)},
		{35.6762, 139.6503, time.Date(2024, time.March, 21, 0, 0, 0, 0, mustLocation(t, "Asia/Tokyo"))},
		{64.1466, -21.9426, time.Date(2024, time.June, 20, 0, 0, 0, 0, time.UTC)}, // Reykjavík
		{65.35, 0.0, time.Date(2024, time.June, 21, 0, 0, 0, 0, time.UTC)},
		{71.2906, -156.7886, time.Date(2024, time.June, 21, 0, 0, 0, 0, mustLocation(t, "America/Anchorage"))},
	}
	for _, c := range cases {
		for _, w := range All(c.date, c.lat, c.lon) {
			if w.Absent {
				if w.Start.Equal(w.End) {
					t.Errorf("%.4f,%.4f %s: absent window is zero-length", c.lat, c.lon, w.Event)
				}
				continue
			}
			if !w.Start.Before(w.End) {
				t.Errorf("%.4f,%.4f %s: bounds out of order %s..%s", c.lat, c.lon, w.Event, w.Start, w.End)
			}
			want := thresholds[w.Event]
			for _, edge := range []struct {
				at      time.Time
				degrees float64
				label   string
			}{
				{w.Start, want[0], "start"},
				{w.End, want[1], "end"},
			} {
				elev := solar.Position(edge.at, c.lat, c.lon).Elevation
				if math.Abs(elev-edge.degrees) > 0.05 {
					t.Errorf("%.4f,%.4f %s %s: elevation %.3f°, want %.3f° (±0.05)",
						c.lat, c.lon, w.Event, edge.label, elev, edge.degrees)
				}
			}
		}
	}
}

// TestTimesInCallersLocation: the date's location selects the local day
// and labels the answers. The same instant read in Tokyo (March 21) and
// in UTC (March 20) yields different days; at Tokyo's longitude with a
// UTC clock the dawn windows land on the previous civil date, matching
// the solar package's adjacent-day convention.
func TestTimesInCallersLocation(t *testing.T) {
	tokyo := mustLocation(t, "Asia/Tokyo")
	instant := time.Date(2024, time.March, 20, 18, 0, 0, 0, time.UTC)

	tokyoDay := instant.In(tokyo) // 2024-03-21 03:00 JST
	ws := All(tokyoDay, 35.6762, 139.6503)
	for _, w := range ws {
		if w.Absent {
			t.Fatalf("Tokyo 2024-03-21: %s reported absent", w.Event)
		}
		if w.Start.Location() != tokyo || w.End.Location() != tokyo {
			t.Errorf("Tokyo 2024-03-21 %s: times not in Asia/Tokyo", w.Event)
		}
	}
	dawn, _ := GoldenHour(tokyoDay, 35.6762, 139.6503)
	withinTolerance(t, "Tokyo 2024-03-21 dawn golden hour start", dawn.Start, parseWall(t, tokyoDay, "05:42:13"))

	utcDay := instant.In(time.UTC) // 2024-03-20 18:00 UTC
	utcDawn, _ := GoldenHour(utcDay, 35.6762, 139.6503)
	if utcDawn.Absent {
		t.Fatal("UTC 2024-03-20 dawn golden hour reported absent")
	}
	if y, m, d := utcDawn.Start.UTC().Date(); y != 2024 || m != time.March || d != 19 {
		t.Errorf("UTC-day dawn golden hour starts %s, want the previous civil date 2024-03-19", utcDawn.Start.UTC())
	}
	if utcDawn.Start.Location() != time.UTC {
		t.Errorf("UTC-day dawn golden hour not in UTC")
	}
}

// TestDSTTransitionDays pins the zone of every window on the two 2024
// DST transition days in Los Angeles: the times must resolve to the
// offset in force at the window instant (PDT on 2024-03-10, PST on
// 2024-11-03).
func TestDSTTransitionDays(t *testing.T) {
	la := mustLocation(t, "America/Los_Angeles")
	cases := []struct {
		date time.Time
		zone string
	}{
		{time.Date(2024, time.March, 10, 0, 0, 0, 0, la), "PDT"},
		{time.Date(2024, time.November, 3, 0, 0, 0, 0, la), "PST"},
	}
	for _, c := range cases {
		for _, w := range All(c.date, 34.0522, -118.2437) {
			if w.Absent {
				t.Fatalf("%s: %s reported absent", c.date.Format("2006-01-02"), w.Event)
			}
			for _, at := range []time.Time{w.Start, w.End} {
				if zone, _ := at.Zone(); zone != c.zone {
					t.Errorf("%s %s: zone = %s, want %s", c.date.Format("2006-01-02"), w.Event, zone, c.zone)
				}
			}
		}
	}
}
