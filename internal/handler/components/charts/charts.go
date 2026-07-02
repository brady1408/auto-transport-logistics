// Package charts provides small, dependency-free server-rendered SVG chart
// components (templ) and the layout math behind them. No external JS libraries
// or CDNs are used — charts are plain inline SVG generated at render time.
//
// Colors are hardcoded brand hex values (rather than CSS tokens) so the same
// component renders identically whether it is embedded in a Tailwind-styled
// dashboard card or a plain app.css report page.
package charts

import (
	"fmt"
	"strconv"
	"strings"
)

// Brand palette (mirrors app.css / the Tailwind config in layout.templ).
const (
	colorNavy   = "#00174b" // primary-container — default bar fill
	colorAccent = "#497cff" // on-primary-container — the brand blue
	colorMint   = "#009668" // on-tertiary-container — mint highlight
	colorAxis   = "#76777d" // outline — axis lines / labels
	colorTrack  = "#eceef0" // surface-container — bar track / gridline
	colorText   = "#191c1e" // on-surface — value labels
	colorMuted  = "#76777d" // muted label text
)

// Datum is a single labeled value in a chart series. Value is already prepared
// (parsed/aggregated) by the caller — chart components do no data access.
type Datum struct {
	Label     string  // category label (x-axis tick or row label)
	Value     float64 // magnitude
	Highlight bool    // if true, render this bar in the mint highlight color
	// Display, when non-empty, is shown in the hover tooltip instead of a
	// plain number (e.g. a pre-formatted money string like "$1,234.50").
	Display string
}

// hasData reports whether a series has at least one strictly-positive value.
// An all-zero (or empty) series renders the quiet "No data yet" placeholder.
func hasData(data []Datum) bool {
	for _, d := range data {
		if d.Value > 0 {
			return true
		}
	}
	return false
}

// maxValue returns the largest value in the series (0 if empty).
func maxValue(data []Datum) float64 {
	m := 0.0
	for _, d := range data {
		if d.Value > m {
			m = d.Value
		}
	}
	return m
}

// barFill picks the fill color for a datum.
func barFill(d Datum) string {
	if d.Highlight {
		return colorMint
	}
	return colorNavy
}

// tooltip returns the hover text for a datum: the label plus a value. If the
// caller supplied a preformatted Display, it is used verbatim.
func tooltip(d Datum) string {
	if d.Display != "" {
		return d.Label + ": " + d.Display
	}
	return d.Label + ": " + trimFloat(d.Value)
}

// trimFloat renders a float without trailing zeros (e.g. 12 not 12.00).
func trimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return s
}

// truncate shortens a label to n runes, appending an ellipsis when clipped.
// Used for horizontal-bar row labels which have a fixed-width gutter.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// ParseMoney parses a money string (as stored by the invoice queries, e.g.
// "1234.50" or "$1,234.50" or "") into a float64. Unparseable input yields 0.
// This tolerates the currency symbol and thousands separators so callers can
// pass the raw AgingBucket / report strings directly.
func ParseMoney(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// FormatMoney formats a float as a US currency string with thousands
// separators and two decimals, e.g. 1234.5 -> "$1,234.50". Used for tooltips
// and axis value labels so chart figures match the app's "$"+string display.
func FormatMoney(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	// Round to cents.
	cents := int64(f*100 + 0.5)
	dollars := cents / 100
	frac := cents % 100
	intStr := strconv.FormatInt(dollars, 10)
	// Insert thousands separators.
	var b strings.Builder
	n := len(intStr)
	for i, c := range intStr {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s$%s.%02d", sign, b.String(), frac)
}

// --- Column (vertical bar) chart geometry ---
//
// The column chart draws each bar within an evenly divided horizontal band and
// scales bar height to the series max. Geometry is computed in Go so the templ
// component stays declarative.

type columnBar struct {
	X, Y, W, H float64
	Fill       string
	Label      string // x-axis tick label
	Tip        string // hover tooltip text
	LabelX     float64
}

type columnLayout struct {
	Width, Height float64
	PlotTop       float64
	PlotBottom    float64
	Bars          []columnBar
}

// Column chart dimensions.
const (
	colChartW      = 520.0
	colChartH      = 180.0
	colPadTop      = 12.0
	colPadBottom   = 26.0 // room for x-axis labels
	colPadX        = 8.0
	colBarGapRatio = 0.35 // fraction of each slot left as gap
)

func computeColumnLayout(data []Datum) columnLayout {
	l := columnLayout{
		Width:      colChartW,
		Height:     colChartH,
		PlotTop:    colPadTop,
		PlotBottom: colChartH - colPadBottom,
	}
	if len(data) == 0 {
		return l
	}
	max := maxValue(data)
	if max <= 0 {
		max = 1
	}
	plotH := l.PlotBottom - l.PlotTop
	slotW := (colChartW - 2*colPadX) / float64(len(data))
	barW := slotW * (1 - colBarGapRatio)
	for i, d := range data {
		slotX := colPadX + float64(i)*slotW
		h := 0.0
		if d.Value > 0 {
			h = (d.Value / max) * plotH
		}
		// Always keep at least a 1px nub for positive-but-tiny values.
		if d.Value > 0 && h < 2 {
			h = 2
		}
		l.Bars = append(l.Bars, columnBar{
			X:      slotX + (slotW-barW)/2,
			Y:      l.PlotBottom - h,
			W:      barW,
			H:      h,
			Fill:   barFill(d),
			Label:  d.Label,
			Tip:    tooltip(d),
			LabelX: slotX + slotW/2,
		})
	}
	return l
}

// --- Horizontal bar chart geometry ---
//
// Each row is a label + a proportional bar + a value. Height grows with the
// number of rows. Labels sit to the left, values to the right of the bar.

type hbar struct {
	Y, W, H float64
	Fill    string
	Label   string
	Value   string // right-aligned value text
	Tip     string
	ValueX  float64
}

type hbarLayout struct {
	Width, Height float64
	LabelW        float64
	BarX          float64
	Bars          []hbar
}

const (
	hbarChartW = 520.0
	hbarRowH   = 26.0
	hbarBarH   = 16.0
	hbarLabelW = 150.0
	hbarValueW = 90.0
	hbarGap    = 8.0
	hbarTopPad = 6.0
	hbarBotPad = 6.0
)

func computeHBarLayout(data []Datum) hbarLayout {
	l := hbarLayout{
		Width:  hbarChartW,
		LabelW: hbarLabelW,
		BarX:   hbarLabelW + hbarGap,
	}
	l.Height = hbarTopPad + hbarBotPad + float64(len(data))*hbarRowH
	max := maxValue(data)
	if max <= 0 {
		max = 1
	}
	barTrackW := hbarChartW - l.BarX - hbarValueW - hbarGap
	for i, d := range data {
		rowY := hbarTopPad + float64(i)*hbarRowH
		w := 0.0
		if d.Value > 0 {
			w = (d.Value / max) * barTrackW
		}
		if d.Value > 0 && w < 2 {
			w = 2
		}
		val := d.Display
		if val == "" {
			val = trimFloat(d.Value)
		}
		l.Bars = append(l.Bars, hbar{
			Y:      rowY + (hbarRowH-hbarBarH)/2,
			W:      w,
			H:      hbarBarH,
			Fill:   barFill(d),
			Label:  d.Label,
			Value:  val,
			Tip:    tooltip(d),
			ValueX: l.BarX + barTrackW + hbarGap,
		})
	}
	return l
}

// --- Sparkline geometry ---

const (
	sparkW    = 520.0
	sparkH    = 60.0
	sparkPadX = 4.0
	sparkPadY = 6.0
)

type sparkline struct {
	Width, Height float64
	Points        string  // "x,y x,y ..." for <polyline>
	AreaPoints    string  // closed path points for the soft fill
	LastX, LastY  float64 // end-cap dot
	HasDot        bool
}

func computeSparkline(data []Datum) sparkline {
	s := sparkline{Width: sparkW, Height: sparkH}
	if len(data) == 0 {
		return s
	}
	max := maxValue(data)
	if max <= 0 {
		max = 1
	}
	plotW := sparkW - 2*sparkPadX
	plotH := sparkH - 2*sparkPadY
	n := len(data)
	step := plotW
	if n > 1 {
		step = plotW / float64(n-1)
	}
	var pts []string
	for i, d := range data {
		x := sparkPadX + float64(i)*step
		y := sparkPadY + (1-d.Value/max)*plotH
		pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
		s.LastX, s.LastY = x, y
	}
	s.Points = strings.Join(pts, " ")
	s.HasDot = true
	// Area fill: line points plus baseline corners.
	area := make([]string, 0, len(pts)+2)
	area = append(area, fmt.Sprintf("%.1f,%.1f", sparkPadX, sparkH-sparkPadY))
	area = append(area, pts...)
	area = append(area, fmt.Sprintf("%.1f,%.1f", s.LastX, sparkH-sparkPadY))
	s.AreaPoints = strings.Join(area, " ")
	return s
}
