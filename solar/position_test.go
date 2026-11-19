package solar

import (
	"math"
	"testing"
	"time"
)

// Reference values were read off NOAA's solar calculator
// (https://gml.noaa.gov/grad/solcalc/) for the pinned inputs and retrieved
// on 2026-11-19. The seasonal instants used as declination anchors are the
// 2024 equinox/solstice times from the US Naval Observatory
// (https://aa.usno.navy.mil/data/Seasons).
//
// Tolerances absorb NOAA's published-minute rounding plus the library's
// own error: a tenth of a degree for positions.

func TestPositionPinnedNOAA(t *testing.T) {
	cases := []struct {
		name      string
		at        time.Time
		lat, lon  float64
		elevation float64 // apparent elevation, degrees
		azimuth   float64 // degrees clockwise from north
	}{
		// Orange County (Santa Ana), 2024 March equinox, pre-dawn.
		{"Orange County 2024-03-20 12:00 UT", time.Date(2024, time.March, 20, 12, 0, 0, 0, time.UTC), 33.7455, -117.8677, -24.219, 72.298},
		// Riverside, 2024 June solstice instant, early afternoon local.
		{"Riverside 2024-06-20 20:51 UT", time.Date(2024, time.June, 20, 20, 51, 0, 0, time.UTC), 33.9533, -117.3962, 73.260, 235.004},
		// Orange County, 2024 September equinox instant, before dawn.
		{"Orange County 2024-09-22 12:44 UT", time.Date(2024, time.September, 22, 12, 44, 0, 0, time.UTC), 33.7455, -117.8677, -12.408, 81.531},
		// Orange County, 2024 December solstice instant, deep night.
		{"Orange County 2024-12-21 09:20 UT", time.Date(2024, time.December, 21, 9, 20, 0, 0, time.UTC), 33.7455, -117.8677, -67.735, 68.329},
		// Sydney (southern hemisphere), 2024 December solstice instant,
		// just after sunset.
		{"Sydney 2024-12-21 09:20 UT", time.Date(2024, time.December, 21, 9, 20, 0, 0, time.UTC), -33.8688, 151.2093, -3.302, 238.684},
		// Singapore (tropics, sun near the zenith past transit).
		{"Singapore 2024-03-20 05:30 UT", time.Date(2024, time.March, 20, 5, 30, 0, 0, time.UTC), 1.3521, 103.8198, 85.335, 253.700},
	}

	for _, c := range cases {
		got := Position(c.at, c.lat, c.lon)
		if math.Abs(got.Elevation-c.elevation) > 0.1 {
			t.Errorf("%s: elevation = %.3f, want %.3f (±0.1)", c.name, got.Elevation, c.elevation)
		}
		if math.Abs(got.Azimuth-c.azimuth) > 0.1 {
			t.Errorf("%s: azimuth = %.3f, want %.3f (±0.1)", c.name, got.Azimuth, c.azimuth)
		}
	}
}

// TestDeclinationAnchors checks the declination machinery against the
// 2024 season instants: the declination crosses 0° at the equinoxes and
// reaches ±23.44° at the solstices. The instants are external anchors;
// the library must land within a tenth of a degree.
func TestDeclinationAnchors(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want float64
	}{
		{"March equinox 2024-03-20 03:06 UT", time.Date(2024, time.March, 20, 3, 6, 0, 0, time.UTC), 0},
		{"June solstice 2024-06-20 20:51 UT", time.Date(2024, time.June, 20, 20, 51, 0, 0, time.UTC), 23.4386},
		{"September equinox 2024-09-22 12:44 UT", time.Date(2024, time.September, 22, 12, 44, 0, 0, time.UTC), 0},
		{"December solstice 2024-12-21 09:20 UT", time.Date(2024, time.December, 21, 9, 20, 0, 0, time.UTC), -23.4386},
	}
	for _, c := range cases {
		jd := julianDay(c.at.Year(), c.at.Month(), c.at.Day()) + utMinutes(c.at.UTC())/1440.0
		if got := declination(julianCentury(jd)); math.Abs(got-c.want) > 0.1 {
			t.Errorf("%s: declination = %.4f, want %.4f (±0.1)", c.name, got, c.want)
		}
	}
}

// TestRefractionPinned checks NOAA's refraction correction at points from
// each branch of the published piecewise formula (values in arc minutes,
// as NOAA's calculator reports them; retrieved 2026-11-19).
func TestRefractionPinned(t *testing.T) {
	cases := []struct {
		elevationDeg float64
		wantArcMin   float64
	}{
		{0, 1735.0 / 60}, // horizon: 28.9'
		{10, 317.3 / 60}, // 5.3' at 10°
		{86, 0},          // above 85° no correction
	}
	for _, c := range cases {
		if got := refraction(c.elevationDeg) * 60; math.Abs(got-c.wantArcMin) > 0.1 {
			t.Errorf("refraction(%.3f°) = %.3f', want %.3f'", c.elevationDeg, got, c.wantArcMin)
		}
	}
}

// TestAzimuthRange sweeps a few days and asserts the azimuth stays in
// [0, 360) and the elevation stays within [-91, 91].
func TestAzimuthRange(t *testing.T) {
	at := time.Date(2024, time.March, 20, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4*24; i++ {
		p := Position(at.Add(time.Duration(i)*15*time.Minute), 33.7455, -117.8677)
		if p.Azimuth < 0 || p.Azimuth >= 360 {
			t.Fatalf("azimuth out of [0,360): %.3f at %s", p.Azimuth, at)
		}
		if p.Elevation < -91 || p.Elevation > 91 {
			t.Fatalf("elevation out of range: %.3f at %s", p.Elevation, at)
		}
	}
}
