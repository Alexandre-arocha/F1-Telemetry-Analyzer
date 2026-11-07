package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Alexandre-arocha/f1-telemetry-analyzer/api/internal/db"
)

// fakeQ implements only the queries the handlers use; any other call panics on the nil interface.
type fakeQ struct{ db.Querier }

func (fakeQ) ListRaces(context.Context) ([]db.Race, error) {
	day := time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC)
	return []db.Race{{ID: 1, Year: 2024, Round: 1, Name: "Bahrain Grand Prix", Circuit: "Sakhir", Country: "Bahrain",
		EventDate: pgtype.Date{Time: day, Valid: true}}}, nil
}

func (fakeQ) ListDrivers(context.Context, int32) ([]db.ListDriversRow, error) {
	return []db.ListDriversRow{{ID: 1, Code: "VER", FullName: "Max Verstappen", Team: "Red Bull"}}, nil
}

func (fakeQ) ListLaps(context.Context, int32) ([]db.ListLapsRow, error) {
	var laps []db.ListLapsRow
	for i := int16(1); i <= 6; i++ { // 6 clean laps, 0.1 s slower each, on one stint
		laps = append(laps, db.ListLapsRow{ID: int32(i), DriverCode: "VER", LapNumber: i,
			LapTimeMs: pgtype.Int4{Int32: 90000 + int32(i)*100, Valid: true},
			TyreLife:  pgtype.Int2{Int16: i, Valid: true}, Stint: pgtype.Int2{Int16: 1, Valid: true},
			Compound: "HARD", Clean: true, HasTelemetry: true})
	}
	return laps, nil
}

func (fakeQ) GetLap(_ context.Context, id int32) (db.GetLapRow, error) {
	if id > 6 {
		return db.GetLapRow{}, pgx.ErrNoRows
	}
	return db.GetLapRow{ID: id, RaceID: 1, DriverCode: "VER", LapNumber: int16(id), HasTelemetry: true}, nil
}

// a straight, flat lap: 100 samples, 5 m apart; the higher the lap id, the slower
func (fakeQ) GetTelemetry(_ context.Context, id int32) (db.Telemetry, error) {
	const n = 100
	t := db.Telemetry{LapID: id, StepM: 5}
	for i := 0; i < n; i++ {
		t.TimeS = append(t.TimeS, float32(i)*0.05*float32(id))
		t.Speed = append(t.Speed, 300)
		t.Rpm = append(t.Rpm, 11000)
		t.Gear = append(t.Gear, 8)
		t.Throttle = append(t.Throttle, 100)
		t.Brake = append(t.Brake, 0)
		t.X = append(t.X, float32(i))
		t.Y = append(t.Y, 0)
	}
	return t, nil
}

func get(t *testing.T, path string, out any) int {
	t.Helper()
	w := httptest.NewRecorder()
	New(fakeQ{}, "*").ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if out != nil && w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("%s: bad json: %v\n%s", path, err, w.Body.String())
		}
	}
	return w.Code
}

func TestRacesAndLaps(t *testing.T) {
	var races []raceDTO
	if code := get(t, "/api/races", &races); code != 200 || len(races) != 1 || races[0].EventDate != "2024-03-02" {
		t.Fatalf("races: %d %+v", code, races)
	}
	var laps []lapDTO
	if code := get(t, "/api/races/1/laps", &laps); code != 200 || len(laps) != 6 || laps[0].Driver != "VER" {
		t.Fatalf("laps: %d %+v", code, laps)
	}
}

func TestDriversHaveConsistencyAndBestLap(t *testing.T) {
	var ds []driverDTO
	if code := get(t, "/api/races/1/drivers", &ds); code != 200 || len(ds) != 1 {
		t.Fatalf("drivers: %d %+v", code, ds)
	}
	d := ds[0]
	if d.CleanLaps != 6 || d.BestLapID == nil || *d.BestLapID != 1 || d.ConsistencyS == nil || *d.ConsistencyS <= 0 {
		t.Errorf("unexpected stats: %+v", d)
	}
}

func TestDegradationSlope(t *testing.T) {
	var st []stintDTO
	if code := get(t, "/api/races/1/degradation", &st); code != 200 || len(st) != 1 {
		t.Fatalf("degradation: %d %+v", code, st)
	}
	if st[0].SlopeSPerLap == nil || *st[0].SlopeSPerLap != 0.1 || st[0].Laps != 6 {
		t.Errorf("want slope 0.1 s/lap over 6 laps: %+v", st[0])
	}
}

func TestCompareAlignsLapsAndComputesDelta(t *testing.T) {
	var r compareResp
	if code := get(t, "/api/compare?a=1&b=2", &r); code != 200 {
		t.Fatalf("compare: %d", code)
	}
	if len(r.Distance) != 100 || len(r.A.Speed) != 100 || len(r.Delta) != 100 || r.Distance[99] != 495 {
		t.Errorf("series not aligned: dist=%d speed=%d delta=%d", len(r.Distance), len(r.A.Speed), len(r.Delta))
	}
	if r.Delta[99] <= 0 || len(r.Track.X) != 100 {
		t.Errorf("lap 2 is slower, delta must be positive: %v", r.Delta[99])
	}
}

func TestReportReturnsAGainsArray(t *testing.T) {
	var r reportResp
	if code := get(t, "/api/report?a=1&b=2", &r); code != 200 || r.Gains == nil {
		t.Fatalf("report: %d %+v", code, r)
	}
}

func TestBadAndMissingIDs(t *testing.T) {
	for path, want := range map[string]int{
		"/api/compare?a=x&b=2":  http.StatusBadRequest,
		"/api/compare?a=1":      http.StatusBadRequest,
		"/api/compare?a=1&b=99": http.StatusNotFound,
		"/api/races/0/laps":     http.StatusBadRequest,
		"/healthz":              http.StatusOK,
	} {
		if code := get(t, path, nil); code != want {
			t.Errorf("%s: got %d, want %d", path, code, want)
		}
	}
}
