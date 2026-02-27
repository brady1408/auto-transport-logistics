package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

func migrateInvoices(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, customerIDs, orderIDs IDMap, logStat func(Stat)) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT A00Id, InvoiceNumber, Active, G00Id, CustName,
		D00Id, InvoiceDate, Terms, TaxCode,
		SubTotal, TaxAmt, Total, Balance, Status, Comments FROM A00`)
	if err != nil {
		log.Fatalf("query A00: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	seen := make(map[string]bool)
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var invNum, custName sql.NullString
		var active, g00id, d00id sql.NullInt64
		var invDate sql.NullTime
		var terms, taxCode sql.NullString
		var subTotal, taxAmt, total, balance sql.NullFloat64
		var status, comments sql.NullString
		if err := rows.Scan(&oldID, &invNum, &active, &g00id, &custName,
			&d00id, &invDate, &terms, &taxCode,
			&subTotal, &taxAmt, &total, &balance, &status, &comments); err != nil {
			log.Printf("  WARN: scan A00 row: %v", err)
			skipCount++
			continue
		}
		invNumVal := nns(invNum)
		if invNumVal == "" {
			invNumVal = fmt.Sprintf("INV-%d", oldID.Int64)
		}
		if seen[invNumVal] {
			log.Printf("  WARN: dup invoice_number '%s' (legacy %d)", invNumVal, oldID.Int64)
			skipCount++
			continue
		}
		seen[invNumVal] = true

		var amtPaid *float64
		if total.Valid && balance.Valid {
			v := total.Float64 - balance.Float64
			if math.Abs(v) > 0.001 {
				amtPaid = &v
			}
		}

		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO invoices (legacy_id, invoice_number, active, customer_id, customer_name,
			order_id, invoice_date, terms, tax_code,
			subtotal, tax, total_amount, amount_paid, balance, status, comments, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			ON CONFLICT (company_id, invoice_number) DO NOTHING
			RETURNING id`,
			nint(oldID), invNumVal, nb(active), lookupFK(customerIDs, g00id), ns(custName),
			lookupFK(orderIDs, d00id), nt(invDate), ns(terms), ns(taxCode),
			nd(subTotal), nd(taxAmt), nd(total), amtPaid, nd(balance), ns(status), ns(comments), companyID).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert invoices (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		ids[nint(oldID)] = newID
		insCount++
	}
	logStat(Stat{Table: "invoices", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
	return ids
}

func migrateInvoiceDetails(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, invoiceIDs, orderIDs, vehicleIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	rows, err := src.QueryContext(ctx, `SELECT A02Id, A00Id, D10Id, Item, Qty,
		Description, UnitPrice, Extended, Taxable FROM A02`)
	if err != nil {
		log.Fatalf("query A02: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, a00id, d10id sql.NullInt64
		var item, desc, taxable sql.NullString
		var qty, unitPrice, extended sql.NullFloat64
		if err := rows.Scan(&oldID, &a00id, &d10id, &item, &qty,
			&desc, &unitPrice, &extended, &taxable); err != nil {
			log.Printf("  WARN: scan A02 row: %v", err)
			skipCount++
			continue
		}
		invID := lookupFK(invoiceIDs, a00id)
		if invID == nil {
			skipCount++
			continue
		}
		var qtyInt *int
		if qty.Valid && qty.Float64 != 0 {
			v := int(qty.Float64)
			qtyInt = &v
		}
		taxBool := false
		if taxable.Valid {
			s := taxable.String
			taxBool = s == "1" || s == "Y" || s == "y"
		}
		_, err := tx.Exec(ctx, `INSERT INTO invoice_details (legacy_id, invoice_id, vehicle_id,
			description, qty, rate, amount, taxable, item_code, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			nint(oldID), *invID, lookupFK(vehicleIDs, d10id),
			ns(desc), qtyInt, nd(unitPrice), nd(extended), taxBool, ns(item), companyID)
		if err != nil {
			log.Printf("  WARN: insert invoice_details (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "invoice_details", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateCreditMemos(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, customerIDs, invoiceIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "A10") {
		logStat(Stat{Table: "credit_memos", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT A10Id, ToA00Id, FromA00Id, CreditDate, Amount FROM A10`)
	if err != nil {
		log.Fatalf("query A10: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	seen := make(map[string]bool)
	for rows.Next() {
		srcCount++
		var oldID, toA00id, fromA00id sql.NullInt64
		var creditDate sql.NullTime
		var amount sql.NullFloat64
		if err := rows.Scan(&oldID, &toA00id, &fromA00id, &creditDate, &amount); err != nil {
			log.Printf("  WARN: scan A10 row: %v", err)
			skipCount++
			continue
		}
		creditNum := fmt.Sprintf("CM-%d", oldID.Int64)
		if seen[creditNum] {
			skipCount++
			continue
		}
		seen[creditNum] = true
		_, err := tx.Exec(ctx, `INSERT INTO credit_memos (legacy_id, credit_number, invoice_id, credit_date, amount, company_id)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (company_id, credit_number) DO NOTHING`,
			nint(oldID), creditNum, lookupFK(invoiceIDs, toA00id), nt(creditDate), nd(amount), companyID)
		if err != nil {
			log.Printf("  WARN: insert credit_memos (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "credit_memos", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migratePayments(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, customerIDs IDMap, logStat func(Stat)) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT A20Id, G00Id, CustName,
		PaymentDate, CheckNumber, Description, Amount, Balance, Type FROM A20`)
	if err != nil {
		log.Fatalf("query A20: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, g00id sql.NullInt64
		var custName, checkNum, desc, payType sql.NullString
		var payDate sql.NullTime
		var amount, balance sql.NullFloat64
		if err := rows.Scan(&oldID, &g00id, &custName,
			&payDate, &checkNum, &desc, &amount, &balance, &payType); err != nil {
			log.Printf("  WARN: scan A20 row: %v", err)
			skipCount++
			continue
		}
		var appliedAmt *float64
		if amount.Valid && balance.Valid {
			v := amount.Float64 - balance.Float64
			if math.Abs(v) > 0.001 {
				appliedAmt = &v
			}
		}
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO payments (legacy_id, customer_id, customer_name,
			payment_date, check_number, amount, applied_amount, unapplied_amount,
			payment_method, comments, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			RETURNING id`,
			nint(oldID), lookupFK(customerIDs, g00id), ns(custName),
			nt(payDate), ns(checkNum), nd(amount), appliedAmt, nd(balance),
			ns(payType), ns(desc), companyID).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert payments (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		ids[nint(oldID)] = newID
		insCount++
	}
	logStat(Stat{Table: "payments", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
	return ids
}

func migratePaymentDetails(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, paymentIDs, invoiceIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	rows, err := src.QueryContext(ctx, `SELECT A30Id, A20Id, A00Id, InvoiceNumber, Amount FROM A30`)
	if err != nil {
		log.Fatalf("query A30: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, a20id, a00id sql.NullInt64
		var invNum sql.NullString
		var amount sql.NullFloat64
		if err := rows.Scan(&oldID, &a20id, &a00id, &invNum, &amount); err != nil {
			log.Printf("  WARN: scan A30 row: %v", err)
			skipCount++
			continue
		}
		payID := lookupFK(paymentIDs, a20id)
		if payID == nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO payment_details (legacy_id, payment_id, invoice_id, invoice_number, amount, company_id)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			nint(oldID), *payID, lookupFK(invoiceIDs, a00id), ns(invNum), nd(amount), companyID)
		if err != nil {
			log.Printf("  WARN: insert payment_details (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "payment_details", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateDamageClaims(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, orderIDs, vehicleIDs, tripIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "A40") {
		logStat(Stat{Table: "damage_claims", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT A40Id, ClaimNumber, D10Id, VIN,
		ClaimDate, EstAmount, PaidAmount, Status, Comments FROM A40`)
	if err != nil {
		log.Fatalf("query A40: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	seen := make(map[string]bool)
	for rows.Next() {
		srcCount++
		var oldID, d10id sql.NullInt64
		var claimNum, vin, status, comments sql.NullString
		var claimDate sql.NullTime
		var estAmt, paidAmt sql.NullFloat64
		if err := rows.Scan(&oldID, &claimNum, &d10id, &vin,
			&claimDate, &estAmt, &paidAmt, &status, &comments); err != nil {
			log.Printf("  WARN: scan A40 row: %v", err)
			skipCount++
			continue
		}
		claimNumVal := nns(claimNum)
		if claimNumVal == "" {
			claimNumVal = fmt.Sprintf("CLM-%d", oldID.Int64)
		}
		if seen[claimNumVal] {
			log.Printf("  WARN: dup claim_number '%s' (legacy %d)", claimNumVal, oldID.Int64)
			skipCount++
			continue
		}
		seen[claimNumVal] = true
		_, err := tx.Exec(ctx, `INSERT INTO damage_claims (legacy_id, claim_number, vehicle_id, vin,
			claim_date, claim_amount, paid_amount, status, description, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (company_id, claim_number) DO NOTHING`,
			nint(oldID), claimNumVal, lookupFK(vehicleIDs, d10id), ns(vin),
			nt(claimDate), nd(estAmt), nd(paidAmt), ns(status), ns(comments), companyID)
		if err != nil {
			log.Printf("  WARN: insert damage_claims (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "damage_claims", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateAccountsPayable(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, tripIDs, employeeIDs, truckIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "A50") {
		logStat(Stat{Table: "accounts_payable", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT A50Id, VendorName, Description,
		TxnDate, DueDate, Amount, Active, APType, D20Id FROM A50`)
	if err != nil {
		log.Fatalf("query A50: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d20id sql.NullInt64
		var vendorName, desc, apType sql.NullString
		var txnDate, dueDate sql.NullTime
		var amount sql.NullFloat64
		var active sql.NullInt64
		if err := rows.Scan(&oldID, &vendorName, &desc,
			&txnDate, &dueDate, &amount, &active, &apType, &d20id); err != nil {
			log.Printf("  WARN: scan A50 row: %v", err)
			skipCount++
			continue
		}
		var status *string
		if apType.Valid {
			s := nns(apType)
			if s != "" {
				status = &s
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO accounts_payable (legacy_id, trip_id, vendor_name,
			payable_date, amount, status, description, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			nint(oldID), lookupFK(tripIDs, d20id), ns(vendorName),
			nt(txnDate), nd(amount), status, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert accounts_payable (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "accounts_payable", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}
