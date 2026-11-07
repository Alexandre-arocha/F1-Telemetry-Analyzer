package analysis

import (
	"math"
	"testing"
)

const step = 5.0

// lapProfile builds a 2 km lap with two corners (apex at 500 m and 1300 m).
// shift1 makes the driver start braking for corner 1 that many metres earlier;
// min1/min2 are the apex speeds in km/h.
func lapProfile(shift1, min1, min2 float64) Trace {
	const n = 400
	t := Trace{StepM: step, Time: make([]float64, n), Speed: make([]float64, n), Throttle: make([]float64, n), Brake: make([]bool, n)}
	apexes := []float64{500, 1300}
	mins := []float64{min1, min2}
	shifts := []float64{shift1, 0}
	for i := 0; i < n; i++ {
		d := float64(i) * step
		v, thr, brk := 300.0, 100.0, false
		for k, apex := range apexes {
			wl := 250 + shifts[k]
			switch {
			case d >= apex-wl && d < apex: // braking and turn-in
				v = mins[k] + (300-mins[k])*(apex-d)/wl
				thr, brk = 0, d < apex-30
				if !brk {
					thr = 50
				}
			case d >= apex && d < apex+250: // exit
				v = mins[k] + (300-mins[k])*(d-apex)/250
				if d < apex+50 {
					thr = 50
				}
			}
		}
		t.Speed[i], t.Throttle[i], t.Brake[i] = v, thr, brk
		if i > 0 {
			t.Time[i] = t.Time[i-1] + step/(t.Speed[i-1]/3.6)
		}
	}
	return t
}

func TestCornersFindsBothDips(t *testing.T) {
	cs := Corners(lapProfile(0, 90, 90))
	if len(cs) != 2 {
		t.Fatalf("want 2 corners, got %d: %+v", len(cs), cs)
	}
	for i, apexIdx := range []int{100, 260} {
		if math.Abs(float64(cs[i].Apex-apexIdx)) > 3 {
			t.Errorf("corner %d apex at sample %d, want ~%d", i+1, cs[i].Apex, apexIdx)
		}
	}
	if cs[0].Start != 0 || cs[1].End != 399 || cs[0].End != cs[1].Start {
		t.Errorf("segments must tile the lap: %+v", cs)
	}
}

func TestBrakeZonesMergesReleasesAndDropsBlips(t *testing.T) {
	b := make([]bool, 100)
	for _, i := range []int{10, 11, 12, 13, 14, 15, 18, 19, 20, 21} { // two runs, 2-sample gap (10 m)
		b[i] = true
	}
	b[60], b[61] = true, true // 10 m blip
	z := BrakeZones(b, step)
	if len(z) != 1 || z[0].Start != 10 || z[0].End != 21 {
		t.Fatalf("got %+v, want one zone 10-21", z)
	}
}

func TestDeltaIsPositiveWhereBIsSlower(t *testing.T) {
	a, b := lapProfile(0, 90, 90), lapProfile(0, 90, 70)
	d := Delta(a, b)
	if len(d) != 400 || d[0] != 0 || d[399] <= 0 {
		t.Fatalf("unexpected delta: len=%d first=%v last=%v", len(d), d[0], d[len(d)-1])
	}
}

func TestReportExplainsEachLoss(t *testing.T) {
	a := lapProfile(0, 90, 90)
	b := lapProfile(40, 90, 70) // brakes 40 m earlier in corner 1, slower apex in corner 2
	r := BuildReport(a, b, 3)
	if len(r.Gains) != 2 {
		t.Fatalf("want 2 gains, got %+v", r.Gains)
	}
	if r.Gains[0].TimeGainS < r.Gains[1].TimeGainS {
		t.Errorf("gains must be sorted by time, got %+v", r.Gains)
	}
	byCorner := map[int]Gain{}
	for _, g := range r.Gains {
		byCorner[g.Corner] = g
	}
	if g := byCorner[1]; g.Reason != EarlyBraking || g.BrakeDiffM == nil || math.Abs(*g.BrakeDiffM-40) > 1 {
		t.Errorf("corner 1 should be early braking by ~40 m: %+v", g)
	}
	if g := byCorner[2]; g.Reason != LowMinSpeed || g.MinSpeedA-g.MinSpeedB < 19 {
		t.Errorf("corner 2 should be a lower minimum speed: %+v", g)
	}
	for _, g := range r.Gains {
		if g.Summary == "" || g.TimeGainS <= 0 {
			t.Errorf("gain without summary or time: %+v", g)
		}
	}
}

func TestReportDetectsSlowExit(t *testing.T) {
	a := lapProfile(0, 90, 90)
	b := lapProfile(0, 90, 90)
	for i := 270; i < 278; i++ { // B gets on the power 40 m later (A is back on it at sample 270)
		b.Throttle[i] = 60
		b.Speed[i] -= 10
	}
	for i := 1; i < len(b.Time); i++ {
		b.Time[i] = b.Time[i-1] + step/(b.Speed[i-1]/3.6)
	}
	r := BuildReport(a, b, 3)
	if len(r.Gains) == 0 || r.Gains[0].Reason != SlowExit || r.Gains[0].ThrottleDiffM == nil {
		t.Fatalf("want a slow_exit gain, got %+v", r.Gains)
	}
}

func TestReportIsEmptyForIdenticalLaps(t *testing.T) {
	a := lapProfile(0, 90, 90)
	r := BuildReport(a, a, 3)
	if r.Gains == nil || len(r.Gains) != 0 || r.TotalDeltaS != 0 {
		t.Fatalf("want no gains, got %+v", r)
	}
}

func TestStdDevAndSlope(t *testing.T) {
	if got := StdDev([]float64{2, 4, 4, 4, 5, 5, 7, 9}); math.Abs(got-2.138) > 0.001 {
		t.Errorf("StdDev = %v", got)
	}
	if StdDev([]float64{3}) != 0 {
		t.Error("StdDev of one value must be 0")
	}
	s, ok := Slope([]float64{1, 2, 3, 4}, []float64{90.0, 90.2, 90.4, 90.6})
	if !ok || math.Abs(s-0.2) > 1e-9 {
		t.Errorf("Slope = %v ok=%v", s, ok)
	}
	if _, ok := Slope([]float64{1, 1, 1}, []float64{1, 2, 3}); ok {
		t.Error("Slope without spread in x must not be ok")
	}
}
