package httpapi

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Alexandre-arocha/f1-telemetry-analyzer/api/internal/analysis"
	"github.com/Alexandre-arocha/f1-telemetry-analyzer/api/internal/db"
)

const (
	minLapsForConsistency = 5
	minLapsForSlope       = 4
	reportTopN            = 3
)

type Server struct{ q db.Querier }

// New builds the router. corsOrigin is the web app origin allowed to call the API.
func New(q db.Querier, corsOrigin string) *gin.Engine {
	s := &Server{q: q}
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), cors(corsOrigin))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	api := r.Group("/api")
	api.GET("/races", s.listRaces)
	api.GET("/races/:id/drivers", s.listDrivers)
	api.GET("/races/:id/laps", s.listLaps)
	api.GET("/races/:id/degradation", s.degradation)
	api.GET("/compare", s.compare)
	api.GET("/report", s.report)
	return r
}

func cors(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) fail(c *gin.Context, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	_ = c.Error(err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func idParam(c *gin.Context, v string) (int32, bool) {
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil || n < 1 {
		badRequest(c, "invalid id")
		return 0, false
	}
	return int32(n), true
}

func int4(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

func int2(v pgtype.Int2) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int16)
	return &n
}

// ---- races, drivers, laps ----

type raceDTO struct {
	ID        int32  `json:"id"`
	Year      int16  `json:"year"`
	Round     int16  `json:"round"`
	Name      string `json:"name"`
	Circuit   string `json:"circuit"`
	Country   string `json:"country"`
	EventDate string `json:"event_date,omitempty"`
	TotalLaps *int   `json:"total_laps"`
}

func (s *Server) listRaces(c *gin.Context) {
	races, err := s.q.ListRaces(c)
	if err != nil {
		s.fail(c, err)
		return
	}
	out := make([]raceDTO, len(races))
	for i, r := range races {
		out[i] = raceDTO{ID: r.ID, Year: r.Year, Round: r.Round, Name: r.Name, Circuit: r.Circuit,
			Country: r.Country, TotalLaps: int2(r.TotalLaps)}
		if r.EventDate.Valid {
			out[i].EventDate = r.EventDate.Time.Format("2006-01-02")
		}
	}
	c.JSON(http.StatusOK, out)
}

type lapDTO struct {
	ID           int32  `json:"id"`
	Driver       string `json:"driver"`
	Lap          int16  `json:"lap"`
	LapTimeMs    *int   `json:"lap_time_ms"`
	S1Ms         *int   `json:"s1_ms"`
	S2Ms         *int   `json:"s2_ms"`
	S3Ms         *int   `json:"s3_ms"`
	Compound     string `json:"compound"`
	TyreLife     *int   `json:"tyre_life"`
	Stint        *int   `json:"stint"`
	Position     *int   `json:"position"`
	Clean        bool   `json:"clean"`
	HasTelemetry bool   `json:"has_telemetry"`
}

func (s *Server) listLaps(c *gin.Context) {
	raceID, ok := idParam(c, c.Param("id"))
	if !ok {
		return
	}
	laps, err := s.q.ListLaps(c, raceID)
	if err != nil {
		s.fail(c, err)
		return
	}
	out := make([]lapDTO, len(laps))
	for i, l := range laps {
		out[i] = lapDTO{ID: l.ID, Driver: l.DriverCode, Lap: l.LapNumber, LapTimeMs: int4(l.LapTimeMs),
			S1Ms: int4(l.S1Ms), S2Ms: int4(l.S2Ms), S3Ms: int4(l.S3Ms), Compound: l.Compound,
			TyreLife: int2(l.TyreLife), Stint: int2(l.Stint), Position: int2(l.Position),
			Clean: l.Clean, HasTelemetry: l.HasTelemetry}
	}
	c.JSON(http.StatusOK, out)
}

type driverDTO struct {
	Code         string   `json:"code"`
	Number       string   `json:"number"`
	Name         string   `json:"name"`
	Team         string   `json:"team"`
	Laps         int      `json:"laps"`
	CleanLaps    int      `json:"clean_laps"`
	BestLapID    *int32   `json:"best_lap_id"` // fastest lap that has telemetry
	BestLapMs    *int     `json:"best_lap_ms"`
	ConsistencyS *float64 `json:"consistency_s"` // std dev of clean lap times
}

func (s *Server) listDrivers(c *gin.Context) {
	raceID, ok := idParam(c, c.Param("id"))
	if !ok {
		return
	}
	drivers, err := s.q.ListDrivers(c, raceID)
	if err != nil {
		s.fail(c, err)
		return
	}
	laps, err := s.q.ListLaps(c, raceID)
	if err != nil {
		s.fail(c, err)
		return
	}
	byDriver := map[string][]db.ListLapsRow{}
	for _, l := range laps {
		byDriver[l.DriverCode] = append(byDriver[l.DriverCode], l)
	}

	out := make([]driverDTO, len(drivers))
	for i, d := range drivers {
		dto := driverDTO{Code: d.Code, Number: d.Number, Name: d.FullName, Team: d.Team}
		var clean []float64
		for _, l := range byDriver[d.Code] {
			dto.Laps++
			if l.Clean && l.LapTimeMs.Valid {
				dto.CleanLaps++
				clean = append(clean, float64(l.LapTimeMs.Int32)/1000)
			}
			if l.HasTelemetry && l.LapTimeMs.Valid && (dto.BestLapMs == nil || int(l.LapTimeMs.Int32) < *dto.BestLapMs) {
				id, ms := l.ID, int(l.LapTimeMs.Int32)
				dto.BestLapID, dto.BestLapMs = &id, &ms
			}
		}
		if len(clean) >= minLapsForConsistency {
			sd := round(analysis.StdDev(clean), 3)
			dto.ConsistencyS = &sd
		}
		out[i] = dto
	}
	c.JSON(http.StatusOK, out)
}

type stintDTO struct {
	Driver       string       `json:"driver"`
	Stint        int          `json:"stint"`
	Compound     string       `json:"compound"`
	Laps         int          `json:"laps"`
	SlopeSPerLap *float64     `json:"slope_s_per_lap"` // tyre degradation: lap time gained per lap of tyre age
	Points       []stintPoint `json:"points"`
}

type stintPoint struct {
	TyreLife int     `json:"tyre_life"`
	LapTimeS float64 `json:"lap_time_s"`
}

func (s *Server) degradation(c *gin.Context) {
	raceID, ok := idParam(c, c.Param("id"))
	if !ok {
		return
	}
	laps, err := s.q.ListLaps(c, raceID)
	if err != nil {
		s.fail(c, err)
		return
	}
	type key struct {
		driver string
		stint  int16
	}
	stints := map[key]*stintDTO{}
	for _, l := range laps {
		if !l.Clean || !l.LapTimeMs.Valid || !l.TyreLife.Valid || !l.Stint.Valid {
			continue
		}
		k := key{l.DriverCode, l.Stint.Int16}
		st := stints[k]
		if st == nil {
			st = &stintDTO{Driver: l.DriverCode, Stint: int(l.Stint.Int16), Compound: l.Compound, Points: []stintPoint{}}
			stints[k] = st
		}
		st.Points = append(st.Points, stintPoint{int(l.TyreLife.Int16), float64(l.LapTimeMs.Int32) / 1000})
	}

	out := make([]stintDTO, 0, len(stints))
	for _, st := range stints {
		st.Laps = len(st.Points)
		if st.Laps >= minLapsForSlope {
			x, y := make([]float64, st.Laps), make([]float64, st.Laps)
			for i, p := range st.Points {
				x[i], y[i] = float64(p.TyreLife), p.LapTimeS
			}
			if slope, ok := analysis.Slope(x, y); ok {
				v := round(slope, 3)
				st.SlopeSPerLap = &v
			}
		}
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Driver != out[j].Driver {
			return out[i].Driver < out[j].Driver
		}
		return out[i].Stint < out[j].Stint
	})
	c.JSON(http.StatusOK, out)
}

// ---- compare and report ----

type lapInfo struct {
	ID        int32  `json:"id"`
	Driver    string `json:"driver"`
	Lap       int16  `json:"lap"`
	LapTimeMs *int   `json:"lap_time_ms"`
}

type lapSeries struct {
	lapInfo
	Speed    []float32 `json:"speed"`
	Throttle []float32 `json:"throttle"`
	Brake    []int16   `json:"brake"`
	Gear     []int16   `json:"gear"`
	RPM      []float32 `json:"rpm"`
}

type compareResp struct {
	StepM    float32   `json:"step_m"`
	Distance []float32 `json:"distance"`
	A        lapSeries `json:"a"`
	B        lapSeries `json:"b"`
	Delta    []float64 `json:"delta"` // seconds, B minus A, accumulated along the lap
	Track    struct {
		X []float32 `json:"x"`
		Y []float32 `json:"y"`
	} `json:"track"`
}

type loadedLap struct {
	info  db.GetLapRow
	tel   db.Telemetry
	trace analysis.Trace
}

func (s *Server) loadLap(ctx context.Context, id int32) (loadedLap, error) {
	info, err := s.q.GetLap(ctx, id)
	if err != nil {
		return loadedLap{}, err
	}
	tel, err := s.q.GetTelemetry(ctx, id)
	if err != nil {
		return loadedLap{}, err
	}
	n := min(len(tel.TimeS), len(tel.Speed), len(tel.Throttle), len(tel.Brake), len(tel.Gear), len(tel.Rpm), len(tel.X), len(tel.Y))
	t := analysis.Trace{StepM: float64(tel.StepM), Time: make([]float64, n), Speed: make([]float64, n),
		Throttle: make([]float64, n), Brake: make([]bool, n)}
	for i := 0; i < n; i++ {
		t.Time[i], t.Speed[i], t.Throttle[i], t.Brake[i] = float64(tel.TimeS[i]), float64(tel.Speed[i]), float64(tel.Throttle[i]), tel.Brake[i] > 0
	}
	return loadedLap{info: info, tel: tel, trace: t}, nil
}

// loadPair reads ?a=<lap id>&b=<lap id>, both laps of the same race.
func (s *Server) loadPair(c *gin.Context) (a, b loadedLap, ok bool) {
	ida, oka := idParam(c, c.Query("a"))
	if !oka {
		return
	}
	idb, okb := idParam(c, c.Query("b"))
	if !okb {
		return
	}
	var err error
	if a, err = s.loadLap(c, ida); err != nil {
		s.fail(c, err)
		return
	}
	if b, err = s.loadLap(c, idb); err != nil {
		s.fail(c, err)
		return
	}
	if a.info.RaceID != b.info.RaceID {
		badRequest(c, "laps belong to different races")
		return
	}
	return a, b, true
}

func info(l loadedLap) lapInfo {
	return lapInfo{ID: l.info.ID, Driver: l.info.DriverCode, Lap: l.info.LapNumber, LapTimeMs: int4(l.info.LapTimeMs)}
}

func (s *Server) compare(c *gin.Context) {
	a, b, ok := s.loadPair(c)
	if !ok {
		return
	}
	n := min(a.trace.Len(), b.trace.Len())
	series := func(l loadedLap) lapSeries {
		return lapSeries{lapInfo: info(l), Speed: l.tel.Speed[:n], Throttle: l.tel.Throttle[:n],
			Brake: l.tel.Brake[:n], Gear: l.tel.Gear[:n], RPM: l.tel.Rpm[:n]}
	}
	resp := compareResp{StepM: a.tel.StepM, Distance: make([]float32, n), A: series(a), B: series(b), Delta: analysis.Delta(a.trace, b.trace)}
	for i := range resp.Distance {
		resp.Distance[i] = float32(i) * a.tel.StepM
	}
	for i, d := range resp.Delta {
		resp.Delta[i] = round(d, 3)
	}
	resp.Track.X, resp.Track.Y = a.tel.X[:n], a.tel.Y[:n]
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, resp)
}

type reportResp struct {
	A           lapInfo         `json:"a"`
	B           lapInfo         `json:"b"`
	TotalDeltaS float64         `json:"total_delta_s"`
	Gains       []analysis.Gain `json:"gains"`
}

// report: where lap B (the one to improve) loses the most time against lap A (the reference).
func (s *Server) report(c *gin.Context) {
	a, b, ok := s.loadPair(c)
	if !ok {
		return
	}
	r := analysis.BuildReport(a.trace, b.trace, reportTopN)
	for i := range r.Gains {
		g := &r.Gains[i]
		g.TimeGainS = round(g.TimeGainS, 3)
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, reportResp{A: info(a), B: info(b), TotalDeltaS: round(r.TotalDeltaS, 3), Gains: r.Gains})
}

func round(v float64, places int) float64 {
	p := math.Pow10(places)
	return math.Round(v*p) / p
}
