// Package windows computes the day's photography windows — golden
// hour, blue hour, and civil twilight — layered on the sun engine in
// the solar package. A photographer reads one answer per window: when
// it starts, when it ends, and which event it names, instead of
// studying the engine's elevation curve.
//
// The package derives every window from the engine's reported elevation
// ([solar.Position]): threshold crossings are bracketed by a coarse scan
// and narrowed by bisection, so a short window near a solstice cannot
// slip between two samples. The package contains no astronomical
// equations of its own.
//
// Windows and thresholds. All thresholds are crossings of the sun's
// apparent elevation in degrees, the elevation [solar.Position] reports,
// and every window comes in a morning (dawn) and an evening (dusk)
// pair:
//
//   - Golden hour: the sun at or below +6°, while rising and while
//     setting. Dawn golden hour runs from the horizon crossing to the
//     +6° crossing; dusk golden hour runs from the +6° crossing back to
//     the horizon. Golden hour stops at the horizon: it does not run
//     down through blue hour to −6°.
//   - Blue hour: the sun between −6° and −4°. Dawn blue hour runs from
//     the −6° crossing to the −4° crossing; dusk blue hour runs from
//     the −4° crossing to the −6° crossing.
//   - Civil twilight: the whole band between the rise/set horizon and
//     −6°, not just the −6° crossing. Dawn civil twilight runs from the
//     −6° crossing to the horizon; dusk civil twilight runs from the
//     horizon to the −6° crossing.
//
// Conventions:
//
//   - Latitudes are degrees, positive north; longitudes are degrees,
//     positive east — as in the solar package.
//   - The date argument selects the local calendar day and supplies the
//     time zone: every returned time is expressed in the date's
//     location. Pass a time carrying the location's *time.Location to
//     get windows on the location's calendar day in its wall clock.
//   - A window whose thresholds the sun never crosses on the requested
//     day (the polar day or night, or a grazing path) is returned as a
//     record flagged [Window.Absent] — not as a missing value. An
//     absent window carries the bounds of the half-day that was
//     searched in Start and End, so a window's Start and End are never
//     equal and an absent window is never zero-length.
//   - The −0.833° horizon edge is a crossing of the engine's reported
//     apparent elevation; [solar.Sunrise] and [solar.Sunset] instead use
//     the geometric −0.833° rise/set horizon, so a window's horizon
//     edge can sit a couple of minutes past the matching event.
//   - A window can run past local midnight (a high-latitude summer
//     evening); the returned time simply falls on the adjacent calendar
//     day, as the solar package's events can.
package windows
