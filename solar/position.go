package solar

import (
	"math"
	"time"
)

const (
	degToRad = math.Pi / 180
	radToDeg = 180 / math.Pi
)

// SunPosition is the sun's apparent position for one instant and place.
type SunPosition struct {
	// Elevation is the sun's apparent elevation in degrees above the
	// horizon: the geometric elevation plus NOAA's atmospheric refraction
	// correction, the elevation NOAA's calculator reports. Negative
	// values are below the horizon.
	Elevation float64

	// Azimuth is the sun's bearing in degrees clockwise from true north,
	// in [0, 360).
	Azimuth float64
}

// Position returns the sun's apparent position at instant t for the given
// latitude and longitude (degrees, east-positive longitude). t may carry
// any location.
func Position(t time.Time, latitude, longitude float64) SunPosition {
	angles := sunAnglesAt(t, latitude, longitude)
	return SunPosition{
		Elevation: angles.elevation + refraction(angles.elevation),
		Azimuth:   angles.azimuth,
	}
}

// sunAngles holds the sun's geometric (unrefracted) angles in degrees.
type sunAngles struct {
	elevation float64
	azimuth   float64
}

func sunAnglesAt(t time.Time, latitude, longitude float64) sunAngles {
	ut := t.UTC()
	jd := julianDay(ut.Year(), ut.Month(), ut.Day()) + utMinutes(ut)/1440.0
	tc := julianCentury(jd)
	return sunAnglesFor(utMinutes(ut), equationOfTime(tc), declination(tc), latitude, longitude)
}

// sunAnglesFor evaluates NOAA's solar-position geometry (the calcAzEl
// equations). minutesUTC is minutes after 0h UT of the Julian day the
// equation of time and declination were evaluated at.
func sunAnglesFor(minutesUTC, eqTime, dec, latitude, longitude float64) sunAngles {
	trueSolarTime := math.Mod(minutesUTC+eqTime+4*longitude, 1440)
	if trueSolarTime < 0 {
		trueSolarTime += 1440
	}
	hourAngle := trueSolarTime/4 - 180 // degrees; negative before local apparent noon

	cosZenith := math.Sin(degToRad*latitude)*math.Sin(degToRad*dec) +
		math.Cos(degToRad*latitude)*math.Cos(degToRad*dec)*math.Cos(degToRad*hourAngle)
	cosZenith = clamp(cosZenith, -1, 1)
	zenith := radToDeg * math.Acos(cosZenith)

	// Azimuth, measured clockwise from north. NOAA resolves the
	// near-vertical case (sun at the zenith) with a fixed bearing.
	azimuth := 180.0
	azDenom := math.Cos(degToRad*latitude) * math.Sin(degToRad*zenith)
	if math.Abs(azDenom) > 0.001 {
		cosAz := (math.Sin(degToRad*latitude)*math.Cos(degToRad*zenith) -
			math.Sin(degToRad*dec)) / azDenom
		cosAz = clamp(cosAz, -1, 1)
		azimuth = 180 - radToDeg*math.Acos(cosAz)
		if hourAngle > 0 {
			azimuth = -azimuth
		}
	} else if latitude <= 0 {
		azimuth = 0
	}
	if azimuth < 0 {
		azimuth += 360
	}
	return sunAngles{elevation: 90 - zenith, azimuth: azimuth}
}

// refraction returns NOAA's atmospheric refraction correction for a
// geometric elevation in degrees, as a (positive) angle in degrees.
func refraction(elevationDeg float64) float64 {
	var correction float64 // arc-seconds
	switch {
	case elevationDeg > 85:
		correction = 0
	case elevationDeg > 5:
		t := math.Tan(degToRad * elevationDeg)
		correction = 58.1/t - 0.07/(t*t*t) + 0.000086/(t*t*t*t*t)
	case elevationDeg > -0.575:
		correction = 1735 + elevationDeg*(-518.2+
			elevationDeg*(103.4+elevationDeg*(-12.79+elevationDeg*0.711)))
	default:
		correction = -20.774 / math.Tan(degToRad*elevationDeg)
	}
	return correction / 3600
}

// utMinutes returns minutes after midnight UT as a float.
func utMinutes(ut time.Time) float64 {
	return float64(ut.Hour()*60+ut.Minute()) +
		float64(ut.Second())/60 + float64(ut.Nanosecond())/(60*1e9)
}

// julianDay returns the Julian Day at 0h UT of the given Gregorian
// calendar date (NOAA's date2julian).
func julianDay(year int, month time.Month, day int) float64 {
	if month <= 2 {
		year--
		month += 12
	}
	a := math.Floor(float64(year / 100))
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*float64(year+4716)) +
		math.Floor(30.6001*float64(month+1)) +
		float64(day) + b - 1524.5
}

// julianCentury converts a Julian Day to Julian centuries since J2000.0.
func julianCentury(jd float64) float64 {
	return (jd - 2451545.0) / 36525.0
}

// geomMeanLongSun returns the geometric mean longitude of the sun in
// degrees, normalized to [0, 360).
func geomMeanLongSun(t float64) float64 {
	l0 := math.Mod(280.46646+t*(36000.76983+t*0.0003032), 360)
	if l0 < 0 {
		l0 += 360
	}
	return l0
}

// geomMeanAnomalySun returns the geometric mean anomaly of the sun in degrees.
func geomMeanAnomalySun(t float64) float64 {
	return 357.52911 + t*(35999.05029-0.0001537*t)
}

// eccentricity returns the eccentricity of the earth's orbit.
func eccentricity(t float64) float64 {
	return 0.016708634 - t*(0.000042037+0.0000001267*t)
}

// sunEqOfCenter returns the sun's equation of center in degrees.
func sunEqOfCenter(t float64) float64 {
	m := degToRad * geomMeanAnomalySun(t)
	return math.Sin(m)*(1.914602-t*(0.004817+0.000014*t)) +
		math.Sin(2*m)*(0.019993-0.000101*t) +
		math.Sin(3*m)*0.000289
}

// sunTrueLong returns the sun's true longitude in degrees.
func sunTrueLong(t float64) float64 {
	return geomMeanLongSun(t) + sunEqOfCenter(t)
}

// omega returns the argument of the perihelion of the earth's orbit
// (NOAA's omega, used for the nutation in longitude correction).
func omega(t float64) float64 {
	return 125.04 - 1934.136*t
}

// sunApparentLongitude returns the sun's apparent longitude in degrees.
func sunApparentLongitude(t float64) float64 {
	return sunTrueLong(t) - 0.00569 - 0.00478*math.Sin(degToRad*omega(t))
}

// meanObliquityOfEcliptic returns the mean obliquity in degrees.
func meanObliquityOfEcliptic(t float64) float64 {
	seconds := 21.448 - t*(46.8150+t*(0.00059-t*0.001813))
	return 23 + (26+seconds/60)/60
}

// obliquityCorrection returns the obliquity of the ecliptic corrected for
// nutation, in degrees.
func obliquityCorrection(t float64) float64 {
	return meanObliquityOfEcliptic(t) + 0.00256*math.Cos(degToRad*omega(t))
}

// declination returns the sun's declination in degrees.
func declination(t float64) float64 {
	e := obliquityCorrection(t)
	lambda := sunApparentLongitude(t)
	return radToDeg * math.Asin(math.Sin(degToRad*e)*math.Sin(degToRad*lambda))
}

// equationOfTime returns the equation of time in minutes.
func equationOfTime(t float64) float64 {
	eps := obliquityCorrection(t)
	l0 := geomMeanLongSun(t)
	e := eccentricity(t)
	m := geomMeanAnomalySun(t)

	y := math.Tan(degToRad * eps / 2)
	y *= y

	sin2l0 := math.Sin(2 * degToRad * l0)
	sinm := math.Sin(degToRad * m)
	cos2l0 := math.Cos(2 * degToRad * l0)
	sin4l0 := math.Sin(4 * degToRad * l0)
	sin2m := math.Sin(2 * degToRad * m)

	eqTime := y*sin2l0 - 2*e*sinm + 4*e*y*sinm*cos2l0 -
		0.5*y*y*sin4l0 - 1.25*e*e*sin2m
	return radToDeg * eqTime * 4
}

func clamp(v, low, high float64) float64 {
	switch {
	case v < low:
		return low
	case v > high:
		return high
	default:
		return v
	}
}
