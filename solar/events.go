package solar

import (
	"errors"
	"math"
	"time"
)

// Sentinel errors reported by the event functions where an event does not
// happen.
var (
	// ErrSunNeverRises is reported when the sun stays below the −0.833°
	// horizon for the whole local day (the polar night). Both Sunrise and
	// Sunset report it on such a day.
	ErrSunNeverRises = errors.New("solar: the sun never rises at this location on this date")

	// ErrSunNeverSets is reported when the sun stays above the −0.833°
	// horizon for the whole local day (the midnight sun). Both Sunrise and
	// Sunset report it on such a day.
	ErrSunNeverSets = errors.New("solar: the sun never sets at this location on this date")
)

// riseSetZenith is the zenith angle of the sunrise/sunset horizon:
// 90° + 0.833°, folding in atmospheric refraction and the sun's apparent
// radius, exactly as NOAA's calculator does.
const riseSetZenith = 90.833

// Sunrise returns the time the sun's upper limb first reaches the −0.833°
// horizon on the calendar day of date (read in date's location). See the
// package documentation for conventions. It reports ErrSunNeverRises on
// the polar night and ErrSunNeverSets on the midnight sun.
func Sunrise(date time.Time, latitude, longitude float64) (time.Time, error) {
	return riseSet(date, latitude, longitude, true)
}

// Sunset returns the time the sun's upper limb last reaches the −0.833°
// horizon on the calendar day of date (read in date's location). See the
// package documentation for conventions. It reports ErrSunNeverSets on
// the midnight sun and ErrSunNeverRises on the polar night.
func Sunset(date time.Time, latitude, longitude float64) (time.Time, error) {
	return riseSet(date, latitude, longitude, false)
}

// SolarNoon returns the time the sun crosses the local meridian on the
// calendar day of date (read in date's location), the day's transit, which
// happens even when the sun never rises or sets. latitude is not used in
// the transit computation; the parameter keeps the event API uniform. The
// error is always nil.
func SolarNoon(date time.Time, latitude, longitude float64) (time.Time, error) {
	y, month, d := date.Date()
	jd := julianDay(y, month, d)

	// The equation of time is evaluated at the transit estimate and
	// refined once, mirroring NOAA's calcSolNoon. Over a day the equation
	// of time changes by well under a second of time, so the refinement
	// leaves the result within seconds of the true transit.
	tc := julianCentury(jd)
	minutes := 720 - 4*longitude - equationOfTime(tc)
	tc = julianCentury(jd + minutes/1440)
	minutes = 720 - 4*longitude - equationOfTime(tc)

	return wallClock(date, minutes), nil
}

// riseSet computes sunrise or sunset in minutes after 0h UT using NOAA's
// two-pass refinement: estimate from the equation of time and declination
// at 0h UT, then re-evaluate at the estimated instant.
func riseSet(date time.Time, latitude, longitude float64, rise bool) (time.Time, error) {
	y, month, d := date.Date()
	jd := julianDay(y, month, d)

	// Classify the day once, from the declination near local midday
	// (jd + 0.5 - longitude/360), as NOAA's calculator evaluates it. The
	// declination at 0h UT can differ enough on the first or last
	// crossing day near the polar circles to misclassify the day, so
	// this evaluation owns the sentinel decision.
	if state, _ := horizonState(latitude, declination(julianCentury(jd+0.5-longitude/360.0))); state != risesAndSets {
		return time.Time{}, state.err()
	}

	// Two-pass refinement, as NOAA's calculator does: estimate from the
	// equation of time and declination at 0h UT, then re-evaluate both at
	// the estimated instant. On a boundary day the refined declination
	// can leave the acos domain; horizonState clamps the hour angle so the
	// event degenerates to a grazing transit instead of NaN.
	tc := julianCentury(jd)
	_, ha := horizonState(latitude, declination(tc))
	minutes := 720 - 4*(longitude+hourAngleTerm(rise, ha)) - equationOfTime(tc)

	// Second pass.
	tc = julianCentury(jd + minutes/1440)
	_, ha = horizonState(latitude, declination(tc))
	minutes = 720 - 4*(longitude+hourAngleTerm(rise, ha)) - equationOfTime(tc)

	return wallClock(date, minutes), nil
}

// hourAngleTerm maps the day's half-day hour angle onto the sunrise/sunset
// term of NOAA's event formula: +HA for sunrise, −HA for sunset.
func hourAngleTerm(rise bool, ha float64) float64 {
	if rise {
		return ha
	}
	return -ha
}

type dayState int

const (
	risesAndSets dayState = iota
	neverRises
	neverSets
)

func (s dayState) err() error {
	switch s {
	case neverRises:
		return ErrSunNeverRises
	case neverSets:
		return ErrSunNeverSets
	}
	return nil
}

// horizonState returns whether the sun rises and sets at the given
// latitude with the given declination, and the half-day hour angle in
// degrees for the −0.833° horizon. An acos argument outside [-1, 1] is
// the polar day/night case; the returned hour angle is clamped there (a
// grazing event), so callers that already classified the day still get
// finite results on boundary days.
func horizonState(latitude, dec float64) (dayState, float64) {
	arg := math.Cos(degToRad*riseSetZenith)/(math.Cos(degToRad*latitude)*math.Cos(degToRad*dec)) -
		math.Tan(degToRad*latitude)*math.Tan(degToRad*dec)
	switch {
	case arg > 1:
		return neverRises, 0
	case arg < -1:
		return neverSets, 180
	default:
		return risesAndSets, radToDeg * math.Acos(arg)
	}
}

// wallClock converts an event time given as minutes after 0h UT of the
// calendar day of date (the value may lie outside [0, 1440)) into a local
// wall-clock time.
//
// The UTC offset is sampled at the event's instant, and the result is
// built from wall-clock components with time.Date, so days containing a
// DST transition resolve to the correct wall clock. Events whose wall
// clock falls on the adjacent calendar day (an extreme longitude paired
// with a mismatched offset) are returned on that adjacent date, matching
// NOAA's calculator.
func wallClock(date time.Time, minutesUTC float64) time.Time {
	y, month, d := date.Date()
	midnightUT := time.Date(y, month, d, 0, 0, 0, 0, time.UTC)

	_, offset := midnightUT.Add(time.Duration(minutesUTC * float64(time.Minute))).In(date.Location()).Zone()
	local := minutesUTC + float64(offset)/60

	dayShift := 0
	for local < 0 {
		local += 1440
		dayShift--
	}
	for local >= 1440 {
		local -= 1440
		dayShift++
	}

	wy, wm, wd := midnightUT.AddDate(0, 0, dayShift).Date()
	secs := int(math.Round(local * 60))
	return time.Date(wy, wm, wd, secs/3600, (secs%3600)/60, secs%60, 0, date.Location())
}
