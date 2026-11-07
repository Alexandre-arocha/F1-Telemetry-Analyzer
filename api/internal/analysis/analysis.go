// Package analysis compares two laps that were resampled on the same distance grid.
//
// It is pure (no I/O): it detects braking zones and corners from the telemetry, measures where
// lap B loses time against lap A, and explains each loss with one of three causes.
package analysis

import (
	"fmt"
	"math"
	"sort"
)

const (
	minCornerDropKmh = 15.0 // a speed dip needs this much prominence to count as a corner
	minCornerGapM    = 100.0
	throttleOnPct    = 90.0 // "on the power" again
	minBrakeZoneM    = 20.0
	brakeMergeGapM   = 15.0
	smoothSamples    = 5

	// OpenF1 samples at ~3.7 Hz (a car covers 15-20 m between samples), so smaller gaps are noise.
	minBrakeDiffM    = 20.0
	minSpeedDiffKmh  = 3.0
	minThrottleDiffM = 20.0
)

// Trace is one lap: sample i is at distance i*StepM.
type Trace struct {
	StepM    float64
	Time     []float64 // seconds since lap start
	Speed    []float64 // km/h
	Throttle []float64 // 0-100
	Brake    []bool
}

// Len is the number of samples every channel has.
func (t Trace) Len() int {
	return min(len(t.Time), len(t.Speed), len(t.Throttle), len(t.Brake))
}

func (t Trace) trim(n int) Trace {
	t.Time, t.Speed, t.Throttle, t.Brake = t.Time[:n], t.Speed[:n], t.Throttle[:n], t.Brake[:n]
	return t
}

// Delta is the accumulated time gap B-A at each sample; positive means B is behind.
func Delta(a, b Trace) []float64 {
	n := min(a.Len(), b.Len())
	d := make([]float64, n)
	for i := range d {
		d[i] = b.Time[i] - a.Time[i]
	}
	return d
}

// Zone is a run of samples (inclusive) under braking.
type Zone struct{ Start, End int }

// BrakeZones finds braking zones, merging short releases and dropping blips.
func BrakeZones(brake []bool, stepM float64) []Zone {
	var runs []Zone
	start := -1
	for i, on := range brake {
		if on && start < 0 {
			start = i
		} else if !on && start >= 0 {
			runs = append(runs, Zone{start, i - 1})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, Zone{start, len(brake) - 1})
	}

	gap := int(math.Ceil(brakeMergeGapM / stepM))
	var merged []Zone
	for _, z := range runs {
		if n := len(merged); n > 0 && z.Start-merged[n-1].End <= gap {
			merged[n-1].End = z.End
			continue
		}
		merged = append(merged, z)
	}

	minLen := int(math.Ceil(minBrakeZoneM / stepM))
	var zones []Zone
	for _, z := range merged {
		if z.End-z.Start+1 >= minLen {
			zones = append(zones, z)
		}
	}
	return zones
}

// Corner is a stretch of the lap around one speed dip, from the preceding speed peak to the next.
type Corner struct{ Number, Start, Apex, End int }

// Corners detects corners from the speed trace of t.
func Corners(t Trace) []Corner {
	n := len(t.Speed)
	if n < 3 || t.StepM <= 0 {
		return nil
	}
	sm := smooth(t.Speed, smoothSamples)
	minGap := int(minCornerGapM / t.StepM)

	var apexes []int
	for i := 1; i < n-1; i++ {
		if sm[i] > sm[i-1] || sm[i] >= sm[i+1] || prominence(sm, i) < minCornerDropKmh {
			continue
		}
		if k := len(apexes); k > 0 && i-apexes[k-1] < minGap {
			if sm[i] < sm[apexes[k-1]] {
				apexes[k-1] = i
			}
			continue
		}
		apexes = append(apexes, i)
	}

	bounds := make([]int, len(apexes)+1)
	bounds[len(apexes)] = n - 1
	for k := 1; k < len(apexes); k++ {
		bounds[k] = argmax(sm, apexes[k-1], apexes[k])
	}
	corners := make([]Corner, len(apexes))
	for k, apex := range apexes {
		corners[k] = Corner{Number: k + 1, Start: bounds[k], Apex: apex, End: bounds[k+1]}
	}
	return corners
}

func smooth(v []float64, w int) []float64 {
	out := make([]float64, len(v))
	half := w / 2
	for i := range v {
		lo, hi := max(0, i-half), min(len(v)-1, i+half)
		var s float64
		for j := lo; j <= hi; j++ {
			s += v[j]
		}
		out[i] = s / float64(hi-lo+1)
	}
	return out
}

// prominence is how far the dip at i sits below the lower of the two surrounding peaks.
func prominence(sm []float64, i int) float64 {
	left, right := sm[i], sm[i]
	for j := i - 1; j >= 0 && sm[j] >= sm[i]; j-- {
		left = max(left, sm[j])
	}
	for j := i + 1; j < len(sm) && sm[j] >= sm[i]; j++ {
		right = max(right, sm[j])
	}
	return min(left, right) - sm[i]
}

func argmax(v []float64, lo, hi int) int {
	best := lo
	for i := lo; i <= hi; i++ {
		if v[i] > v[best] {
			best = i
		}
	}
	return best
}

func argmin(v []float64, lo, hi int) int {
	best := lo
	for i := lo; i <= hi; i++ {
		if v[i] < v[best] {
			best = i
		}
	}
	return best
}

// Reason codes for why lap B lost time in a corner.
const (
	EarlyBraking = "early_braking"
	LowMinSpeed  = "low_min_speed"
	SlowExit     = "slow_exit"
	Mixed        = "mixed"
)

// Gain is a stretch where B could gain time by driving like A.
type Gain struct {
	Corner        int      `json:"corner"`
	StartM        float64  `json:"start_m"`
	ApexM         float64  `json:"apex_m"`
	EndM          float64  `json:"end_m"`
	TimeGainS     float64  `json:"time_gain_s"`
	Reason        string   `json:"reason"`
	Summary       string   `json:"summary"`
	BrakeDiffM    *float64 `json:"brake_diff_m,omitempty"`    // + = B brakes earlier than A
	ThrottleDiffM *float64 `json:"throttle_diff_m,omitempty"` // + = B gets on the power later than A
	MinSpeedA     float64  `json:"min_speed_a"`
	MinSpeedB     float64  `json:"min_speed_b"`
}

// Report is the automatic comparison of B against the reference lap A.
type Report struct {
	TotalDeltaS float64 `json:"total_delta_s"`
	Gains       []Gain  `json:"gains"`
}

// BuildReport returns the topN corners where B loses the most time to A.
//
// The size of the loss is measured (delta at the end minus delta at the start of the corner).
// The cause is an estimate: each candidate cause is converted into the seconds it would cost
// (extra metres at speed, or the slower speed through the corner) and the biggest one wins.
func BuildReport(a, b Trace, topN int) Report {
	n := min(a.Len(), b.Len())
	if n < 3 {
		return Report{Gains: []Gain{}}
	}
	a, b = a.trim(n), b.trim(n)
	delta := Delta(a, b)
	step := a.StepM
	za, zb := BrakeZones(a.Brake, step), BrakeZones(b.Brake, step)

	var gains []Gain
	for _, c := range Corners(a) {
		loss := delta[c.End] - delta[c.Start]
		if loss <= 0 {
			continue
		}
		g := diagnose(a, b, za, zb, c)
		g.Corner, g.TimeGainS = c.Number, loss
		g.StartM, g.EndM = float64(c.Start)*step, float64(c.End)*step
		gains = append(gains, g)
	}
	sort.Slice(gains, func(i, j int) bool { return gains[i].TimeGainS > gains[j].TimeGainS })
	if len(gains) > topN {
		gains = gains[:topN]
	}
	if gains == nil {
		gains = []Gain{}
	}
	return Report{TotalDeltaS: delta[n-1], Gains: gains}
}

func diagnose(a, b Trace, za, zb []Zone, c Corner) Gain {
	step := a.StepM
	apexA, apexB := argmin(a.Speed, c.Start, c.End), argmin(b.Speed, c.Start, c.End)
	minA, minB := a.Speed[apexA], b.Speed[apexB]
	onA, onB := brakeOnset(za, c.Start, apexA), brakeOnset(zb, c.Start, apexB)
	thA, thB := throttleOn(a, apexA, c.End), throttleOn(b, apexB, c.End)

	g := Gain{ApexM: float64(apexA) * step, MinSpeedA: minA, MinSpeedB: minB}
	reason, best := Mixed, 0.0
	consider := func(r string, seconds float64) {
		if seconds > best {
			reason, best = r, seconds
		}
	}

	if onA >= 0 && onB >= 0 {
		d := float64(onA-onB) * step
		g.BrakeDiffM = &d
		if d >= minBrakeDiffM {
			consider(EarlyBraking, d/ms(b.Speed[onB]))
		}
	}
	if dv := minA - minB; dv >= minSpeedDiffKmh {
		zone := 100.0
		if onA >= 0 && thA > onA {
			zone = min(max(float64(thA-onA)*step, 60), 300)
		}
		consider(LowMinSpeed, zone*(1/ms(minB)-1/ms(minA)))
	}
	if thA >= 0 && thB >= 0 {
		d := float64(thB-thA) * step
		g.ThrottleDiffM = &d
		if d >= minThrottleDiffM {
			consider(SlowExit, d/ms(b.Speed[thB]))
		}
	}

	g.Reason = reason
	switch reason {
	case EarlyBraking:
		g.Summary = fmt.Sprintf("Frenagem antecipada: freou %.0f m antes da referência.", *g.BrakeDiffM)
	case LowMinSpeed:
		g.Summary = fmt.Sprintf("Velocidade mínima menor: %.0f km/h contra %.0f km/h da referência.", minB, minA)
	case SlowExit:
		g.Summary = fmt.Sprintf("Saída de curva lenta: acelerou %.0f m depois da referência.", *g.ThrottleDiffM)
	default:
		g.Summary = "Perda distribuída pelo trecho, sem uma causa dominante."
	}
	return g
}

// ms converts km/h to m/s, never returning zero.
func ms(kmh float64) float64 { return max(kmh, 1) / 3.6 }

// brakeOnset is the start of the last braking zone before the apex, or -1 (lifted off only).
func brakeOnset(zones []Zone, from, apex int) int {
	onset := -1
	for _, z := range zones {
		if z.Start >= from-3 && z.Start <= apex {
			onset = z.Start
		}
	}
	return onset
}

// throttleOn is the first sample in [from, to] with the throttle back on, or -1.
func throttleOn(t Trace, from, to int) int {
	for i := from; i <= to; i++ {
		if t.Throttle[i] >= throttleOnPct {
			return i
		}
	}
	return -1
}

// StdDev is the sample standard deviation (0 for fewer than two values).
func StdDev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var mean float64
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	var ss float64
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	return math.Sqrt(ss / float64(len(xs)-1))
}

// Slope is the least-squares slope of y over x; ok is false without enough spread in x.
func Slope(x, y []float64) (slope float64, ok bool) {
	n := float64(len(x))
	if len(x) < 3 || len(x) != len(y) {
		return 0, false
	}
	var sx, sy float64
	for i := range x {
		sx += x[i]
		sy += y[i]
	}
	mx, my := sx/n, sy/n
	var sxy, sxx float64
	for i := range x {
		sxy += (x[i] - mx) * (y[i] - my)
		sxx += (x[i] - mx) * (x[i] - mx)
	}
	if sxx == 0 {
		return 0, false
	}
	return sxy / sxx, true
}
