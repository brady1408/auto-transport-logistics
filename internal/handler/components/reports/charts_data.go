package reports

import (
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/charts"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

// arAgingBucketData sums the per-customer AR aging rows into the four aging
// buckets and returns them as chart data. Summing the same rows the table
// renders keeps the chart and table perfectly consistent. The 90+ bucket is
// highlighted to flag the most overdue balances.
func arAgingBucketData(rows []store.ArAgingRow) []charts.Datum {
	var current, d31, d61, d90 float64
	for _, r := range rows {
		current += charts.ParseMoney(r.Current)
		d31 += charts.ParseMoney(r.Days31)
		d61 += charts.ParseMoney(r.Days61)
		d90 += charts.ParseMoney(r.Days90)
	}
	mk := func(label string, v float64, hl bool) charts.Datum {
		return charts.Datum{Label: label, Value: v, Display: charts.FormatMoney(v), Highlight: hl}
	}
	return []charts.Datum{
		mk("0-30", current, false),
		mk("31-60", d31, false),
		mk("61-90", d61, false),
		mk("90+", d90, true),
	}
}

// revenueTopCustomersData returns the top-N customers by total revenue as
// horizontal-bar chart data. Rows arrive already sorted by revenue DESC from
// the store query, so we simply take the first `n`.
func revenueTopCustomersData(rows []store.RevenueByCustomerRow, n int) []charts.Datum {
	if n > len(rows) {
		n = len(rows)
	}
	data := make([]charts.Datum, 0, n)
	for _, r := range rows[:n] {
		v := charts.ParseMoney(r.TotalRevenue)
		label := r.CustomerName
		if label == "" {
			label = r.CustomerNumber
		}
		data = append(data, charts.Datum{
			Label:   label,
			Value:   v,
			Display: charts.FormatMoney(v),
		})
	}
	return data
}
