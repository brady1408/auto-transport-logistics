package migration

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// migrateCompanies maps legacy C00 IDs to the target companyID.
// In the multi-tenant context the company already exists; we do not insert new companies.
func migrateCompanies(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) IDMap {
	t := time.Now()
	ids := make(IDMap)
	if !tableExists(src, "C00") {
		logStat(Stat{Table: "companies", Elapsed: time.Since(t)})
		return ids
	}
	rows, err := src.QueryContext(ctx, `SELECT C00Id FROM C00`)
	if err != nil {
		log.Printf("  WARN: query C00: %v", err)
		logStat(Stat{Table: "companies", Elapsed: time.Since(t)})
		return ids
	}
	defer rows.Close()
	for rows.Next() {
		var oldID sql.NullInt64
		if err := rows.Scan(&oldID); err != nil {
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = companyID
		}
	}
	logStat(Stat{Table: "companies", Source: len(ids), Inserted: 0, Skipped: 0, Elapsed: time.Since(t)})
	return ids
}

func migrateZones(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G30") {
		logStat(Stat{Table: "zones", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G30Id, Zone, Description, Region FROM G30")
	if err != nil {
		log.Fatalf("query G30: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var zone, desc, region sql.NullString
		if err := rows.Scan(&oldID, &zone, &desc, &region); err != nil {
			skipCount++
			continue
		}
		zoneVal := nns(zone)
		if zoneVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO zones (legacy_id, zone, description, region, company_id)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (company_id, zone) DO NOTHING`,
			ni(oldID), zoneVal, ns(desc), ns(region), companyID)
		if err != nil {
			log.Printf("  WARN: insert zones: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "zones", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateRegions(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G35") {
		logStat(Stat{Table: "regions", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G35Id, Region, Description FROM G35")
	if err != nil {
		log.Printf("  WARN: query G35: %v", err)
		logStat(Stat{Table: "regions", Elapsed: time.Since(t)})
		return
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var region, desc sql.NullString
		if err := rows.Scan(&oldID, &region, &desc); err != nil {
			skipCount++
			continue
		}
		regionVal := nns(region)
		if regionVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO regions (legacy_id, region, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, region) DO NOTHING`,
			ni(oldID), regionVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert regions: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "regions", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateDispatchCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G57") {
		logStat(Stat{Table: "dispatch_codes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G57Id, DispatchCode, Description FROM G57")
	if err != nil {
		log.Fatalf("query G57: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO dispatch_codes (legacy_id, code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert dispatch_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "dispatch_codes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateEquipmentTypes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G23") {
		logStat(Stat{Table: "equipment_types", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G23Id, TypeCode, TypeDesc FROM G23")
	if err != nil {
		log.Fatalf("query G23: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO equipment_types (legacy_id, type_code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, type_code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert equipment_types: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "equipment_types", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateItems(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G40") {
		logStat(Stat{Table: "items", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G40Id, OEM, Description FROM G40")
	if err != nil {
		log.Fatalf("query G40: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var item, desc sql.NullString
		if err := rows.Scan(&oldID, &item, &desc); err != nil {
			skipCount++
			continue
		}
		itemVal := nns(item)
		if itemVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO items (legacy_id, item, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, item) DO NOTHING`,
			ni(oldID), itemVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert items: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "items", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateVehicleMakes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G42") {
		logStat(Stat{Table: "vehicle_makes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G42Id, Make, Model, Weight, Category FROM G42")
	if err != nil {
		log.Fatalf("query G42: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, weight sql.NullInt64
		var make_, model, category sql.NullString
		if err := rows.Scan(&oldID, &make_, &model, &weight, &category); err != nil {
			skipCount++
			continue
		}
		makeVal := nns(make_)
		modelVal := nns(model)
		if makeVal == "" && modelVal == "" {
			skipCount++
			continue
		}
		if makeVal == "" {
			makeVal = "UNKNOWN"
		}
		if modelVal == "" {
			modelVal = "UNKNOWN"
		}
		_, err := tx.Exec(ctx, `INSERT INTO vehicle_makes (legacy_id, make, model, weight, category, company_id)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (company_id, make, model) DO NOTHING`,
			ni(oldID), makeVal, modelVal, ni(weight), ns(category), companyID)
		if err != nil {
			log.Printf("  WARN: insert vehicle_makes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "vehicle_makes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateVinDefinitions(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G43") {
		logStat(Stat{Table: "vin_definitions", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT G43Id, P1,P2,P3,P4,P5,P6,P7,P8,P9,P10,P11,P12,P13,P14,P15,P16,P17 FROM G43`)
	if err != nil {
		log.Fatalf("query G43: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var p [17]sql.NullString
		if err := rows.Scan(&oldID, &p[0], &p[1], &p[2], &p[3], &p[4], &p[5], &p[6], &p[7],
			&p[8], &p[9], &p[10], &p[11], &p[12], &p[13], &p[14], &p[15], &p[16]); err != nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO vin_definitions (legacy_id, p1,p2,p3,p4,p5,p6,p7,p8,p9,p10,p11,p12,p13,p14,p15,p16,p17, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
			ni(oldID), ns(p[0]), ns(p[1]), ns(p[2]), ns(p[3]), ns(p[4]),
			ns(p[5]), ns(p[6]), ns(p[7]), ns(p[8]), ns(p[9]),
			ns(p[10]), ns(p[11]), ns(p[12]), ns(p[13]), ns(p[14]),
			ns(p[15]), ns(p[16]), companyID)
		if err != nil {
			log.Printf("  WARN: insert vin_definitions: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "vin_definitions", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateColorCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G45") {
		logStat(Stat{Table: "color_codes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G45Id, MfgCode, ColorCode, ColorDescription FROM G45")
	if err != nil {
		log.Fatalf("query G45: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var mfg, code, desc sql.NullString
		if err := rows.Scan(&oldID, &mfg, &code, &desc); err != nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO color_codes (legacy_id, mfg_code, color_code, description, company_id)
			VALUES ($1,$2,$3,$4,$5)`,
			ni(oldID), ns(mfg), ns(code), ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert color_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "color_codes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateHoldCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G47") {
		logStat(Stat{Table: "hold_codes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G47Id, HoldCode, HoldDescription FROM G47")
	if err != nil {
		log.Fatalf("query G47: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO hold_codes (legacy_id, code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert hold_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "hold_codes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateDeclinationCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G48") {
		logStat(Stat{Table: "declination_codes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G48Id, DeclineCode, DeclineDesc FROM G48")
	if err != nil {
		log.Fatalf("query G48: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO declination_codes (legacy_id, code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert declination_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "declination_codes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateFieldCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, mssqlTable, pgTable string, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, mssqlTable) {
		logStat(Stat{Table: pgTable, Elapsed: time.Since(t)})
		return
	}
	pkCol := mssqlTable + "Id"
	q := "SELECT " + pkCol + ", FieldCode, Description FROM " + mssqlTable
	rows, err := src.QueryContext(ctx, q)
	if err != nil {
		log.Printf("  WARN: query %s: %v", mssqlTable, err)
		logStat(Stat{Table: pgTable, Elapsed: time.Since(t)})
		return
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, "INSERT INTO "+pgTable+" (legacy_id, code, description, company_id) VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING",
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert %s: %v", pgTable, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: pgTable, Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateDamageAreas(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G70") {
		logStat(Stat{Table: "damage_areas", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G70Id, DamageAreaCode, Description FROM G70")
	if err != nil {
		log.Fatalf("query G70: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO damage_areas (legacy_id, code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert damage_areas: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "damage_areas", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateDamageTypes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G71") {
		logStat(Stat{Table: "damage_types", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G71Id, DamageTypeCode, Description FROM G71")
	if err != nil {
		log.Fatalf("query G71: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO damage_types (legacy_id, code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert damage_types: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "damage_types", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateDamageSeverities(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G72") {
		logStat(Stat{Table: "damage_severities", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G72Id, DamageSeverityCode, Description FROM G72")
	if err != nil {
		log.Fatalf("query G72: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		if err := rows.Scan(&oldID, &code, &desc); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO damage_severities (legacy_id, code, description, company_id)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert damage_severities: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "damage_severities", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateChartOfAccounts(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G80") {
		logStat(Stat{Table: "chart_of_accounts", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G80Id, AccountType, AccountName, AccountNum, OpeningBalance, OpeningDate FROM G80")
	if err != nil {
		log.Fatalf("query G80: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var acctType, acctName, acctNum sql.NullString
		var openBal sql.NullFloat64
		var openDate sql.NullTime
		if err := rows.Scan(&oldID, &acctType, &acctName, &acctNum, &openBal, &openDate); err != nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO chart_of_accounts (legacy_id, account_type, account_name, account_num, opening_balance, opening_date, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			ni(oldID), ns(acctType), ns(acctName), ns(acctNum), nd(openBal), nt(openDate), companyID)
		if err != nil {
			log.Printf("  WARN: insert chart_of_accounts: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "chart_of_accounts", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateTerms(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G85") {
		logStat(Stat{Table: "terms", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G85Id, TermName, Description, DaysToPay FROM G85")
	if err != nil {
		log.Fatalf("query G85: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, days sql.NullInt64
		var term, desc sql.NullString
		if err := rows.Scan(&oldID, &term, &desc, &days); err != nil {
			skipCount++
			continue
		}
		termVal := nns(term)
		if termVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO terms (legacy_id, term, description, days, company_id)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (company_id, term) DO NOTHING`,
			ni(oldID), termVal, ns(desc), ni(days), companyID)
		if err != nil {
			log.Printf("  WARN: insert terms: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "terms", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateTaxCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G86") {
		logStat(Stat{Table: "tax_codes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G86Id, TaxCodeName, Description, StateTaxRate FROM G86")
	if err != nil {
		log.Fatalf("query G86: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var code, desc sql.NullString
		var rate sql.NullFloat64
		if err := rows.Scan(&oldID, &code, &desc, &rate); err != nil {
			skipCount++
			continue
		}
		codeVal := nns(code)
		if codeVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO tax_codes (legacy_id, code, description, rate, company_id)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (company_id, code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), nd(rate), companyID)
		if err != nil {
			log.Printf("  WARN: insert tax_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "tax_codes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateVendors(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G50") {
		logStat(Stat{Table: "vendors", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT G50Id, Name, Address1, Address2, City, State, Zip,
		Phone, Fax, Contact, FedSSNumber FROM G50`)
	if err != nil {
		log.Fatalf("query G50: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var name, addr, addr2, city, state, zip, phone, fax, contact, taxID sql.NullString
		if err := rows.Scan(&oldID, &name, &addr, &addr2, &city, &state, &zip,
			&phone, &fax, &contact, &taxID); err != nil {
			skipCount++
			continue
		}
		nameVal := nns(name)
		if nameVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO vendors (legacy_id, name, address, address2, city, state, zip,
			phone, fax, contact, tax_id, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			ni(oldID), nameVal, ns(addr), ns(addr2), ns(city), ns(state), ns(zip),
			ns(phone), ns(fax), ns(contact), ns(taxID), companyID)
		if err != nil {
			log.Printf("  WARN: insert vendors: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "vendors", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateVendorGroups(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G53") {
		logStat(Stat{Table: "vendor_groups", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G53Id, Name, Description FROM G53")
	if err != nil {
		log.Printf("  WARN: query G53: %v", err)
		logStat(Stat{Table: "vendor_groups", Elapsed: time.Since(t)})
		return
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var name, desc sql.NullString
		if err := rows.Scan(&oldID, &name, &desc); err != nil {
			skipCount++
			continue
		}
		nameVal := nns(name)
		if nameVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO vendor_groups (legacy_id, group_name, description, company_id)
			VALUES ($1,$2,$3,$4)`,
			ni(oldID), nameVal, ns(desc), companyID)
		if err != nil {
			log.Printf("  WARN: insert vendor_groups: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "vendor_groups", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateCarriers(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G55") {
		logStat(Stat{Table: "carriers", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT G55Id, LinkId, CarrierName, Address, Address2, City, State, Zip,
		Contact, Phone, Fax FROM G55`)
	if err != nil {
		log.Fatalf("query G55: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var linkID, name, addr, addr2, city, state, zip, contact, phone, fax sql.NullString
		if err := rows.Scan(&oldID, &linkID, &name, &addr, &addr2, &city, &state, &zip,
			&contact, &phone, &fax); err != nil {
			skipCount++
			continue
		}
		nameVal := nns(name)
		if nameVal == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO carriers (legacy_id, link_id, carrier_name, address, city, state, zip,
			contact, phone, fax, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (company_id, carrier_name) DO NOTHING`,
			ni(oldID), ns(linkID), nameVal, ns(addr), ns(city), ns(state), ns(zip),
			ns(contact), ns(phone), ns(fax), companyID)
		if err != nil {
			log.Printf("  WARN: insert carriers: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "carriers", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateZonePricing(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "G32") {
		logStat(Stat{Table: "zone_pricing", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT G32Id, ZoneA, ZoneB, Description, Amount, Miles, TransportDays, ShipTo FROM G32`)
	if err != nil {
		log.Fatalf("query G32: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, miles, days sql.NullInt64
		var zoneA, zoneB, desc, shipTo sql.NullString
		var amount sql.NullFloat64
		if err := rows.Scan(&oldID, &zoneA, &zoneB, &desc, &amount, &miles, &days, &shipTo); err != nil {
			skipCount++
			continue
		}
		za := nns(zoneA)
		zb := nns(zoneB)
		if za == "" || zb == "" {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO zone_pricing (legacy_id, zone_a, zone_b, description, amount, miles, transport_days, ship_to, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (company_id, zone_a, zone_b) DO NOTHING`,
			ni(oldID), za, zb, ns(desc), nd(amount), ni(miles), ni(days), ns(shipTo), companyID)
		if err != nil {
			log.Printf("  WARN: insert zone_pricing: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "zone_pricing", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}
