package pages

import (
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/charts"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

// ordersPerWeekData shapes the last-N-weeks order counts into chart data.
// Labels use the week-start date as "M/D" (e.g. "6/30").
func ordersPerWeekData(rows []store.OrdersPerWeekRow) []charts.Datum {
	data := make([]charts.Datum, 0, len(rows))
	for _, r := range rows {
		data = append(data, charts.Datum{
			Label: r.WeekStart.Format("1/2"),
			Value: float64(r.Count),
		})
	}
	return data
}

// revenuePerMonthData shapes the last-N-months revenue totals into chart data.
// Labels use the short month name (e.g. "Jul"); tooltips show formatted money.
func revenuePerMonthData(rows []store.RevenuePerMonthRow) []charts.Datum {
	data := make([]charts.Datum, 0, len(rows))
	for _, r := range rows {
		v := charts.ParseMoney(r.Total)
		data = append(data, charts.Datum{
			Label:   r.MonthStart.Format("Jan"),
			Value:   v,
			Display: charts.FormatMoney(v),
		})
	}
	return data
}

// agingBucketData turns the four AR aging buckets into a bar series. The 90+
// bucket is highlighted (mint) to draw the eye to the most overdue balances.
func agingBucketData(a store.AgingBucket) []charts.Datum {
	mk := func(label, raw string, highlight bool) charts.Datum {
		v := charts.ParseMoney(raw)
		return charts.Datum{
			Label:     label,
			Value:     v,
			Display:   charts.FormatMoney(v),
			Highlight: highlight,
		}
	}
	return []charts.Datum{
		mk("0-30", a.Current, false),
		mk("31-60", a.Days31, false),
		mk("61-90", a.Days61, false),
		mk("90+", a.Days90, true),
	}
}
