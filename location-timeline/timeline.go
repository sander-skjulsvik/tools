package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

const (
	dateLayout  = "2006-01-02"
	localLayout = "2006-01-02T15:04:05"
)

// Point is one piece of evidence of where you were, from any source. Only points
// that pass the config are made, so every point counts.
type Point struct {
	Source string `json:"source"`
	ID     string `json:"id"`
	// Time is in UTC; LocalTime is the wall clock where the point was made.
	Time      time.Time `json:"time"`
	LocalTime string    `json:"local_time"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	City      string    `json:"city,omitempty"`
	State     string    `json:"state,omitempty"`
	Country   string    `json:"country,omitempty"`
	// AccuracyM is the radius in metres the location is accurate to, when the
	// source knows it. Photos do not.
	AccuracyM float64 `json:"accuracy_m,omitempty"`
	// Evidence says why the source counted the point. Its values belong to the
	// source: for Immich, "camera", "person" or both.
	Evidence []string `json:"evidence"`
}

// Date is the local calendar day of the point.
func (p Point) Date() string {
	return p.LocalTime[:len(dateLayout)]
}

// Place names the point's town, or its coordinates when Immich has no name for it.
func (p Point) Place() string {
	var parts []string
	for _, part := range []string{p.City, p.Country} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%.3f, %.3f", p.Latitude, p.Longitude)
	}
	return strings.Join(parts, ", ")
}

func (p Point) countryName() string {
	return cmp.Or(p.Country, "Unknown country")
}

// Range is an inclusive range of local dates. Empty ends are open.
type Range struct {
	Since, Until string
}

func (r Range) contains(date string) bool {
	return (r.Since == "" || date >= r.Since) && (r.Until == "" || date <= r.Until)
}

func (r Range) overlaps(start, end string) bool {
	return (r.Until == "" || start <= r.Until) && (r.Since == "" || end >= r.Since)
}

// ParseDate takes YYYY-MM-DD, or a time before today such as 30d, 6w, 6m or 1y.
func ParseDate(value string, today time.Time) (string, error) {
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse(dateLayout, value); err == nil {
		return value, nil
	}
	var amount int
	var unit string
	if n, _ := fmt.Sscanf(value, "%d%s", &amount, &unit); n == 2 && amount >= 0 {
		switch unit {
		case "d":
			return today.AddDate(0, 0, -amount).Format(dateLayout), nil
		case "w":
			return today.AddDate(0, 0, -7*amount).Format(dateLayout), nil
		case "m":
			return today.AddDate(0, -amount, 0).Format(dateLayout), nil
		case "y":
			return today.AddDate(-amount, 0, 0).Format(dateLayout), nil
		}
	}
	return "", fmt.Errorf("%q is not a date: use YYYY-MM-DD, or a time ago such as 30d, 6w, 6m or 1y", value)
}

// distanceKm is the great-circle distance between two coordinates.
func distanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371
	toRad := math.Pi / 180
	dLat := (lat2 - lat1) * toRad
	dLon := (lon2 - lon1) * toRad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

// PlaceCount is how many points a stay or day has at one place.
type PlaceCount struct {
	Place  string `json:"place"`
	Points int    `json:"points"`
}

// Entry is a stay in one place, or a gap with no evidence. A gap is unknown: it
// says nothing about where you were.
type Entry struct {
	Kind       string    `json:"kind"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	StartLocal string    `json:"start_local"`
	EndLocal   string    `json:"end_local"`
	// The rest is set for stays only. Place is the most common place.
	Place     string       `json:"place,omitempty"`
	Country   string       `json:"country,omitempty"`
	Places    []PlaceCount `json:"places,omitempty"`
	Latitude  float64      `json:"latitude,omitempty"`
	Longitude float64      `json:"longitude,omitempty"`
	Points    int          `json:"points,omitempty"`
	Days      int          `json:"days_with_points,omitempty"`
}

// StayOptions decide when a new stay begins.
type StayOptions struct {
	// Level is "city", grouping points within RadiusKm, or "country".
	Level    string
	RadiusKm float64
	// MaxGap is the longest time without points inside a stay; longer becomes a gap.
	MaxGap time.Duration
}

type stay struct {
	points           []Point
	latSum, lonSum   float64
	countries, dates map[string]int
}

func (s *stay) add(p Point) {
	s.points = append(s.points, p)
	s.latSum += p.Latitude
	s.lonSum += p.Longitude
	s.countries[p.countryName()]++
	s.dates[p.Date()]++
}

func (s *stay) centre() (float64, float64) {
	n := float64(len(s.points))
	return s.latSum / n, s.lonSum / n
}

func (s *stay) fits(p Point, options StayOptions) bool {
	if options.Level == "country" {
		return p.countryName() == s.points[0].countryName()
	}
	lat, lon := s.centre()
	return distanceKm(lat, lon, p.Latitude, p.Longitude) <= options.RadiusKm
}

func mostFirst(counts map[string]int) []PlaceCount {
	places := make([]PlaceCount, 0, len(counts))
	for place, count := range counts {
		places = append(places, PlaceCount{Place: place, Points: count})
	}
	slices.SortFunc(places, func(a, b PlaceCount) int {
		return cmp.Or(cmp.Compare(b.Points, a.Points), cmp.Compare(a.Place, b.Place))
	})
	return places
}

func (s *stay) entry(options StayOptions) Entry {
	first, last := s.points[0], s.points[len(s.points)-1]
	counts := map[string]int{}
	for _, p := range s.points {
		if options.Level == "country" {
			counts[p.countryName()]++
		} else {
			counts[p.Place()]++
		}
	}
	places := mostFirst(counts)
	lat, lon := s.centre()
	return Entry{
		Kind: "stay", Start: first.Time, End: last.Time, StartLocal: first.LocalTime, EndLocal: last.LocalTime,
		Place: places[0].Place, Country: mostFirst(s.countries)[0].Place, Places: places,
		Latitude: math.Round(lat*1e4) / 1e4, Longitude: math.Round(lon*1e4) / 1e4,
		Points: len(s.points), Days: len(s.dates),
	}
}

// Stays groups points into stays, with a gap wherever there is no point for
// longer than MaxGap. Points must be in time order.
func Stays(points []Point, options StayOptions) []Entry {
	var entries []Entry
	var current *stay
	for _, p := range points {
		if current != nil {
			last := current.points[len(current.points)-1]
			switch {
			case p.Time.Sub(last.Time) > options.MaxGap:
				entries = append(entries, current.entry(options), Entry{
					Kind: "gap", Start: last.Time, End: p.Time, StartLocal: last.LocalTime, EndLocal: p.LocalTime,
				})
				current = nil
			case !current.fits(p, options):
				entries = append(entries, current.entry(options))
				current = nil
			}
		}
		if current == nil {
			current = &stay{countries: map[string]int{}, dates: map[string]int{}}
		}
		current.add(p)
	}
	if current != nil {
		entries = append(entries, current.entry(options))
	}
	return entries
}

// InRange keeps the entries that overlap the range. A gap that only touches the
// range on the day it starts or ends is left out, since that day has points.
func InRange(entries []Entry, r Range) []Entry {
	kept := []Entry{}
	for _, entry := range entries {
		start, end := entry.StartLocal[:len(dateLayout)], entry.EndLocal[:len(dateLayout)]
		if !r.overlaps(start, end) {
			continue
		}
		if entry.Kind == "gap" && ((r.Since != "" && end == r.Since) || (r.Until != "" && start == r.Until)) {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

// DayPlace is one place seen on a day.
type DayPlace struct {
	Place   string `json:"place"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Country string `json:"country,omitempty"`
	Points  int    `json:"points"`
	// First and Last are local times, as 15:04.
	First string `json:"first"`
	Last  string `json:"last"`
}

// Day lists the places seen on one local date, in the order first seen. A day
// with no places has no evidence, which is not the same as staying put.
type Day struct {
	Date   string     `json:"date"`
	Places []DayPlace `json:"places"`
}

// Days returns every date in the range, defaulting to the dates of the points.
func Days(points []Point, r Range) []Day {
	byDate := map[string][]Point{}
	for _, p := range points {
		byDate[p.Date()] = append(byDate[p.Date()], p)
	}
	if len(points) > 0 {
		r.Since = cmp.Or(r.Since, points[0].Date())
		r.Until = cmp.Or(r.Until, points[len(points)-1].Date())
	}
	start, err1 := time.Parse(dateLayout, r.Since)
	end, err2 := time.Parse(dateLayout, r.Until)
	days := []Day{}
	if err1 != nil || err2 != nil {
		return days
	}
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		day := Day{Date: date.Format(dateLayout), Places: []DayPlace{}}
		dayPoints := byDate[day.Date]
		// Local times order a day better than UTC when crossing time zones.
		slices.SortStableFunc(dayPoints, func(a, b Point) int { return strings.Compare(a.LocalTime, b.LocalTime) })
		index := map[string]int{}
		for _, p := range dayPoints {
			clock := p.LocalTime[len("2006-01-02T") : len("2006-01-02T")+5]
			i, seen := index[p.Place()]
			if !seen {
				i = len(day.Places)
				index[p.Place()] = i
				day.Places = append(day.Places, DayPlace{
					Place: p.Place(), City: p.City, State: p.State, Country: p.Country, First: clock,
				})
			}
			day.Places[i].Points++
			day.Places[i].Last = clock
		}
		days = append(days, day)
	}
	return days
}

// CountryVisit is one country's points within a range.
type CountryVisit struct {
	Country string `json:"country"`
	First   string `json:"first"`
	Last    string `json:"last"`
	Days    int    `json:"days_with_points"`
}

// Coverage tells how much of a range has evidence at all.
type Coverage struct {
	Since          string `json:"since"`
	Until          string `json:"until"`
	Days           int    `json:"days"`
	DaysWithPoints int    `json:"days_with_points"`
	// LongestGap is the longest run of days without points.
	LongestGapDays  int    `json:"longest_gap_days"`
	LongestGapSince string `json:"longest_gap_since,omitempty"`
}

// CountriesReport answers "which countries have you been to since".
type CountriesReport struct {
	Coverage  Coverage       `json:"coverage"`
	Countries []CountryVisit `json:"countries"`
}

// Countries lists the countries with points in the range, most recent first.
func Countries(points []Point, r Range) CountriesReport {
	days := Days(points, r)
	report := CountriesReport{Countries: []CountryVisit{}}
	if len(days) == 0 {
		return report
	}
	report.Coverage = Coverage{Since: days[0].Date, Until: days[len(days)-1].Date, Days: len(days)}
	visits := map[string]*CountryVisit{}
	gap, gapSince := 0, ""
	for _, day := range days {
		if len(day.Places) == 0 {
			if gap == 0 {
				gapSince = day.Date
			}
			gap++
			if gap > report.Coverage.LongestGapDays {
				report.Coverage.LongestGapDays, report.Coverage.LongestGapSince = gap, gapSince
			}
			continue
		}
		gap = 0
		report.Coverage.DaysWithPoints++
		seen := map[string]bool{}
		for _, place := range day.Places {
			country := cmp.Or(place.Country, "Unknown country")
			if seen[country] {
				continue
			}
			seen[country] = true
			visit, ok := visits[country]
			if !ok {
				visit = &CountryVisit{Country: country, First: day.Date}
				visits[country] = visit
			}
			visit.Last = day.Date
			visit.Days++
		}
	}
	for _, visit := range visits {
		report.Countries = append(report.Countries, *visit)
	}
	slices.SortFunc(report.Countries, func(a, b CountryVisit) int {
		return cmp.Or(cmp.Compare(b.Last, a.Last), cmp.Compare(a.Country, b.Country))
	})
	return report
}

// PointsInRange keeps the points on dates in the range.
func PointsInRange(points []Point, r Range) []Point {
	kept := []Point{}
	for _, p := range points {
		if r.contains(p.Date()) {
			kept = append(kept, p)
		}
	}
	return kept
}
