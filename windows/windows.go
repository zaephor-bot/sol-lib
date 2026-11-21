package windows

import (
	"fmt"
	"time"

	"github.com/zaephor-bot/sol-lib/solar"
)

// Event names one of the day's six photography windows.
type Event int

const (
	// DawnGoldenHour is the morning golden hour: from the horizon
	// crossing to the sun's +6° crossing while it rises.
	DawnGoldenHour Event = iota

	// DuskGoldenHour is the evening golden hour: from the sun's +6°
	// crossing back to the horizon while it sets.
	DuskGoldenHour

	// DawnBlueHour is the morning blue hour: from the sun's −6°
	// crossing to its −4° crossing while it rises.
	DawnBlueHour

	// DuskBlueHour is the evening blue hour: from the sun's −4°
	// crossing to its −6° crossing while it sets.
	DuskBlueHour

	// DawnCivilTwilight is morning civil twilight: from the sun's −6°
	// crossing to the horizon crossing while it rises.
	DawnCivilTwilight

	// DuskCivilTwilight is evening civil twilight: from the horizon
	// crossing to the sun's −6° crossing while it sets.
	DuskCivilTwilight
)

// String returns the window's spoken name, e.g. "dawn golden hour".
func (e Event) String() string {
	switch e {
	case DawnGoldenHour:
		return "dawn golden hour"
	case DuskGoldenHour:
		return "dusk golden hour"
	case DawnBlueHour:
		return "dawn blue hour"
	case DuskBlueHour:
		return "dusk blue hour"
	case DawnCivilTwilight:
		return "dawn civil twilight"
	case DuskCivilTwilight:
		return "dusk civil twilight"
	default:
		return fmt.Sprintf("Event(%d)", int(e))
	}
}

// Window is one photography window on a requested day: a start, an end,
// and the event it names.
type Window struct {
	// Event names which window this record is.
	Event Event

	// Start is when the window opens and End when it closes, in the
	// requested date's location. For an absent window they carry the
	// bounds of the half-day that was searched instead, so a window's
	// Start and End are never equal.
	Start time.Time
	End   time.Time

	// Absent reports that the window does not happen on the requested
	// day: the sun never crosses one of the window's thresholds (the
	// polar day or night, or a grazing path). Test Absent before
	// reading Start and End.
	Absent bool
}

// Threshold elevations, in degrees of the sun's apparent elevation as
// the engine reports it ([solar.Position]).
const (
	// horizonElevation matches the rise/set horizon the engine's
	// sunrise and sunset use.
	horizonElevation = -0.833

	// blueShallowElevation is the shallow (-4°) edge of blue hour.
	blueShallowElevation = -4.0

	// twilightElevation is the deep edge of blue hour and of civil
	// twilight (-6°).
	twilightElevation = -6.0

	// goldenElevation is the +6° edge of golden hour.
	goldenElevation = 6.0
)

// Threshold table indices; civil twilight and blue hour share the −6°
// edge, so its crossing is searched once.
const (
	thHorizon = iota
	thBlueShallow
	thTwilight
	thGolden
	thCount
)

var thresholdElevation = [thCount]float64{
	horizonElevation, blueShallowElevation, twilightElevation, goldenElevation,
}

// Search parameters.
const (
	// scanStep is the coarse bracketing step of the crossing scan. It
	// is fine enough that a minutes-scale feature — a tropical blue
	// hour, or the short night just inside the Arctic Circle at the
	// June solstice — cannot fall between two samples unnoticed.
	scanStep = 2 * time.Minute

	// halfDayPad extends each half-day search a little past solar
	// midnight, so the crossings of a midnight-spanning window near
	// the polar circles are still found.
	halfDayPad = 30 * time.Minute

	// crossingPrecision is the bracket width the bisection stops
	// below; crossings are reported to whole seconds.
	crossingPrecision = 200 * time.Millisecond
)

// GoldenHour returns the day's two golden hours: dawn, from the horizon
// crossing to the sun's +6° crossing while it rises, and dusk, from the
// +6° crossing back to the horizon while it sets. On the polar day or
// night both windows are absent.
func GoldenHour(date time.Time, latitude, longitude float64) (dawn, dusk Window) {
	a := analyzeDay(date, latitude, longitude)
	return a.morningWindow(thHorizon, thGolden, DawnGoldenHour, date.Location()),
		a.eveningWindow(thGolden, thHorizon, DuskGoldenHour, date.Location())
}

// BlueHour returns the day's two blue hours: dawn, from the sun's −6°
// crossing to its −4° crossing while it rises, and dusk, from the −4°
// crossing to the −6° crossing while it sets. When the sun never
// reaches the −6…−4° band (the polar day or night), both windows are
// absent.
func BlueHour(date time.Time, latitude, longitude float64) (dawn, dusk Window) {
	a := analyzeDay(date, latitude, longitude)
	return a.morningWindow(thTwilight, thBlueShallow, DawnBlueHour, date.Location()),
		a.eveningWindow(thBlueShallow, thTwilight, DuskBlueHour, date.Location())
}

// CivilTwilight returns the day's two civil twilights: dawn, from the
// sun's −6° crossing to the horizon crossing while it rises, and dusk,
// from the horizon crossing to the −6° crossing while it sets. When the
// sun never bridges horizon and −6° (the polar day or night), both
// windows are absent.
func CivilTwilight(date time.Time, latitude, longitude float64) (dawn, dusk Window) {
	a := analyzeDay(date, latitude, longitude)
	return a.morningWindow(thTwilight, thHorizon, DawnCivilTwilight, date.Location()),
		a.eveningWindow(thHorizon, thTwilight, DuskCivilTwilight, date.Location())
}

// All returns the day's six windows in a fixed order: the dawn trio
// (civil twilight, blue hour, golden hour), then the dusk trio in
// reverse (golden hour, blue hour, civil twilight). The order is fixed,
// not chronological: dawn blue hour opens together with dawn civil
// twilight and ends inside it, and the dusk trio mirrors that.
func All(date time.Time, latitude, longitude float64) []Window {
	a := analyzeDay(date, latitude, longitude)
	loc := date.Location()
	return []Window{
		a.morningWindow(thTwilight, thHorizon, DawnCivilTwilight, loc),
		a.morningWindow(thTwilight, thBlueShallow, DawnBlueHour, loc),
		a.morningWindow(thHorizon, thGolden, DawnGoldenHour, loc),
		a.eveningWindow(thGolden, thHorizon, DuskGoldenHour, loc),
		a.eveningWindow(thBlueShallow, thTwilight, DuskBlueHour, loc),
		a.eveningWindow(thHorizon, thTwilight, DuskCivilTwilight, loc),
	}
}

// crossing is the outcome of one half-day threshold search: whether the
// crossing happens, and when.
type crossing struct {
	found bool
	at    time.Time
}

// dayAnalysis holds the day's threshold crossings, one search per
// threshold per half-day, all read off the engine's elevation.
type dayAnalysis struct {
	noon    time.Time         // the day's solar transit, as an instant
	morning [thCount]crossing // the last rising crossing before noon
	evening [thCount]crossing // the first falling crossing after noon
}

// analyzeDay searches the engine's elevation for the day's threshold
// crossings. The day is split at its solar transit into a rising and a
// setting half; on each half the sun's elevation passes a given
// threshold at most once, except across a graze around solar midnight,
// so the morning search keeps the last rising crossing and the evening
// search the first falling one.
func analyzeDay(date time.Time, latitude, longitude float64) dayAnalysis {
	noon, _ := solar.SolarNoon(date, latitude, longitude)
	noon = noon.UTC()
	a := dayAnalysis{noon: noon}
	for th := 0; th < thCount; th++ {
		threshold := thresholdElevation[th]
		a.morning[th] = lastRising(noon.Add(-12*time.Hour-halfDayPad), noon, threshold, latitude, longitude)
		a.evening[th] = firstFalling(noon, noon.Add(12*time.Hour+halfDayPad), threshold, latitude, longitude)
	}
	return a
}

// morningWindow pairs two morning crossings into a window; when either
// crossing is missing the window is absent and carries the searched
// half-day's bounds, so it is never zero-length.
func (a dayAnalysis) morningWindow(from, to int, event Event, loc *time.Location) Window {
	if a.morning[from].found && a.morning[to].found {
		return Window{
			Event: event,
			Start: a.morning[from].at.In(loc),
			End:   a.morning[to].at.In(loc),
		}
	}
	return Window{
		Event:  event,
		Start:  a.noon.Add(-12 * time.Hour).In(loc),
		End:    a.noon.In(loc),
		Absent: true,
	}
}

// eveningWindow pairs two evening crossings into a window; when either
// crossing is missing the window is absent and carries the searched
// half-day's bounds, so it is never zero-length.
func (a dayAnalysis) eveningWindow(from, to int, event Event, loc *time.Location) Window {
	if a.evening[from].found && a.evening[to].found {
		return Window{
			Event: event,
			Start: a.evening[from].at.In(loc),
			End:   a.evening[to].at.In(loc),
		}
	}
	return Window{
		Event:  event,
		Start:  a.noon.In(loc),
		End:    a.noon.Add(12 * time.Hour).In(loc),
		Absent: true,
	}
}

// elevationGap returns the engine's apparent elevation at t minus the
// threshold: the function whose sign changes mark the crossings.
func elevationGap(t time.Time, latitude, longitude, threshold float64) float64 {
	return solar.Position(t, latitude, longitude).Elevation - threshold
}

// crossingPoint is one threshold crossing found by the scan.
type crossingPoint struct {
	at     time.Time
	rising bool
}

// scanCrossings finds every crossing of threshold between start and
// end: the engine's elevation is sampled every scanStep, each sign
// change is bracketed by the pair of samples around it, and the bracket
// is narrowed by bisection.
func scanCrossings(start, end time.Time, threshold, latitude, longitude float64) []crossingPoint {
	var found []crossingPoint
	record := func(lo, hi time.Time, rising bool) {
		found = append(found, crossingPoint{at: narrow(lo, hi, threshold, latitude, longitude), rising: rising})
	}

	prevAt := start
	prev := elevationGap(start, latitude, longitude, threshold)
	for at := start.Add(scanStep); at.Before(end); at = at.Add(scanStep) {
		cur := elevationGap(at, latitude, longitude, threshold)
		if (prev < 0) != (cur < 0) {
			record(prevAt, at, prev < 0)
		}
		prevAt, prev = at, cur
	}
	if prevAt.Before(end) {
		if cur := elevationGap(end, latitude, longitude, threshold); (prev < 0) != (cur < 0) {
			record(prevAt, end, prev < 0)
		}
	}
	return found
}

// narrow bisects a bracketed crossing down to crossingPrecision and
// reports it rounded to whole seconds.
func narrow(lo, hi time.Time, threshold, latitude, longitude float64) time.Time {
	fLo := elevationGap(lo, latitude, longitude, threshold)
	for hi.Sub(lo) > crossingPrecision {
		mid := lo.Add(hi.Sub(lo) / 2)
		if fMid := elevationGap(mid, latitude, longitude, threshold); (fLo < 0) == (fMid < 0) {
			lo, fLo = mid, fMid
		} else {
			hi = mid
		}
	}
	return lo.Add(hi.Sub(lo) / 2).Round(time.Second)
}

// lastRising returns the last rising crossing of threshold in
// [start, end]: the crossing that closes any graze around solar
// midnight and opens the morning's window.
func lastRising(start, end time.Time, threshold, latitude, longitude float64) crossing {
	var last crossing
	for _, p := range scanCrossings(start, end, threshold, latitude, longitude) {
		if p.rising {
			last = crossing{found: true, at: p.at}
		}
	}
	return last
}

// firstFalling returns the first falling crossing of threshold in
// [start, end]: the crossing that opens the evening's window.
func firstFalling(start, end time.Time, threshold, latitude, longitude float64) crossing {
	for _, p := range scanCrossings(start, end, threshold, latitude, longitude) {
		if !p.rising {
			return crossing{found: true, at: p.at}
		}
	}
	return crossing{}
}
