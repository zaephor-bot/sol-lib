// Package solar computes the sun's position and the day's sun events from
// the equations published on the calculation-details page of NOAA's solar
// calculator:
//
//	https://gml.noaa.gov/grad/solcalc/calcdetails.html
//
// The package is pure Go and depends on the standard library only.
//
// Conventions:
//
//   - Latitudes are degrees, positive north; longitudes are degrees,
//     positive east.
//   - Sunrise and sunset use the standard sunrise/sunset horizon of
//     −0.833° elevation (atmospheric refraction plus the sun's apparent
//     radius), the horizon NOAA's calculator uses.
//   - The elevation reported by [Position] is the apparent elevation:
//     NOAA's atmospheric refraction correction is included. Azimuth is
//     measured clockwise from true north.
//   - The event functions ([Sunrise], [Sunset], [SolarNoon]) take a date
//     whose calendar day, read in the time value's location, selects the
//     local day; results are returned in that same location. Pass a time
//     carrying the location's *time.Location to get events on the
//     location's calendar day in its wall clock.
//   - When the sun never rises or never sets on the requested day (the
//     polar night or the midnight sun), the event functions report
//     [ErrSunNeverRises] or [ErrSunNeverSets] instead of a bogus time. On
//     a never-sets day both [Sunrise] and [Sunset] report
//     ErrSunNeverSets; on a never-rises day both report ErrSunNeverRises.
//   - For an extreme longitude paired with a mismatched UTC offset, an
//     event's wall clock can fall on the adjacent calendar day; that
//     adjacent date is returned, matching how NOAA's calculator reports
//     it.
package solar
