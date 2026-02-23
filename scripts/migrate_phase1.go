package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

func migrateCompanies(ctx context.Context, src *sql.DB, tx pgx.Tx) IDMap {
	t := time.Now()
	ids := make(IDMap)
	if !tableExists(src, "C00") {
		logTable("companies", 0, 0, 0, time.Since(t))
		return ids
	}
	// C00 in actual DB is mostly config. We extract minimal company identity fields.
	rows, err := src.QueryContext(ctx, `SELECT C00Id, SCAC FROM C00`)
	if err != nil {
		log.Printf("  WARN: query C00: %v", err)
		logTable("companies", 0, 0, 0, time.Since(t))
		return ids
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var scac sql.NullString
		if err := rows.Scan(&oldID, &scac); err != nil {
			skipCount++
			continue
		}
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO companies (legacy_id, company_name, scac)
			VALUES ($1, $2, $3) RETURNING id`,
			ni(oldID), "ATLinks Transport", ns(scac)).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert companies: %v", err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logTable("companies", srcCount, insCount, skipCount, time.Since(t))
	return ids
}

func migrateZones(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G30") {
		logTable("zones", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO zones (legacy_id, zone, description, region)
			VALUES ($1,$2,$3,$4) ON CONFLICT (zone) DO NOTHING`,
			ni(oldID), zoneVal, ns(desc), ns(region))
		if err != nil {
			log.Printf("  WARN: insert zones: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("zones", srcCount, insCount, skipCount, time.Since(t))
}

func migrateRegions(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G35") {
		logTable("regions", 0, 0, 0, time.Since(t))
		return
	}
	rows, err := src.QueryContext(ctx, "SELECT G35Id, Region, Description FROM G35")
	if err != nil {
		log.Printf("  WARN: query G35: %v", err)
		logTable("regions", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO regions (legacy_id, region, description)
			VALUES ($1,$2,$3) ON CONFLICT (region) DO NOTHING`,
			ni(oldID), regionVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert regions: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("regions", srcCount, insCount, skipCount, time.Since(t))
}

func migrateDispatchCodes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G57") {
		logTable("dispatch_codes", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO dispatch_codes (legacy_id, code, description)
			VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert dispatch_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("dispatch_codes", srcCount, insCount, skipCount, time.Since(t))
}

func migrateEquipmentTypes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G23") {
		logTable("equipment_types", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO equipment_types (legacy_id, type_code, description)
			VALUES ($1,$2,$3) ON CONFLICT (type_code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert equipment_types: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("equipment_types", srcCount, insCount, skipCount, time.Since(t))
}

func migrateItems(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G40") {
		logTable("items", 0, 0, 0, time.Since(t))
		return
	}
	// G40 actual columns: G40Id, OEM, Description, RateRequired, HoldRequired
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
		_, err := tx.Exec(ctx, `INSERT INTO items (legacy_id, item, description)
			VALUES ($1,$2,$3) ON CONFLICT (item) DO NOTHING`,
			ni(oldID), itemVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert items: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("items", srcCount, insCount, skipCount, time.Since(t))
}

func migrateVehicleMakes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G42") {
		logTable("vehicle_makes", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO vehicle_makes (legacy_id, make, model, weight, category)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (make, model) DO NOTHING`,
			ni(oldID), makeVal, modelVal, ni(weight), ns(category))
		if err != nil {
			log.Printf("  WARN: insert vehicle_makes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("vehicle_makes", srcCount, insCount, skipCount, time.Since(t))
}

func migrateVinDefinitions(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G43") {
		logTable("vin_definitions", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO vin_definitions (legacy_id, p1,p2,p3,p4,p5,p6,p7,p8,p9,p10,p11,p12,p13,p14,p15,p16,p17)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			ni(oldID), ns(p[0]), ns(p[1]), ns(p[2]), ns(p[3]), ns(p[4]),
			ns(p[5]), ns(p[6]), ns(p[7]), ns(p[8]), ns(p[9]),
			ns(p[10]), ns(p[11]), ns(p[12]), ns(p[13]), ns(p[14]),
			ns(p[15]), ns(p[16]))
		if err != nil {
			log.Printf("  WARN: insert vin_definitions: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("vin_definitions", srcCount, insCount, skipCount, time.Since(t))
}

func migrateColorCodes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G45") {
		logTable("color_codes", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO color_codes (legacy_id, mfg_code, color_code, description)
			VALUES ($1,$2,$3,$4)`,
			ni(oldID), ns(mfg), ns(code), ns(desc))
		if err != nil {
			log.Printf("  WARN: insert color_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("color_codes", srcCount, insCount, skipCount, time.Since(t))
}

func migrateHoldCodes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G47") {
		logTable("hold_codes", 0, 0, 0, time.Since(t))
		return
	}
	// Actual columns: G47Id, MfgCode, HoldCode, HoldDescription, ...
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
		_, err := tx.Exec(ctx, `INSERT INTO hold_codes (legacy_id, code, description)
			VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert hold_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("hold_codes", srcCount, insCount, skipCount, time.Since(t))
}

func migrateDeclinationCodes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G48") {
		logTable("declination_codes", 0, 0, 0, time.Since(t))
		return
	}
	// Actual: G48Id, MfgCode, DeclineCode, DeclineDesc, Comments
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
		_, err := tx.Exec(ctx, `INSERT INTO declination_codes (legacy_id, code, description)
			VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert declination_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("declination_codes", srcCount, insCount, skipCount, time.Since(t))
}

func migrateFieldCodes(ctx context.Context, src *sql.DB, tx pgx.Tx, mssqlTable, pgTable string) {
	t := time.Now()
	if !tableExists(src, mssqlTable) {
		logTable(pgTable, 0, 0, 0, time.Since(t))
		return
	}
	pkCol := mssqlTable + "Id"
	// Actual columns: GxxId, FieldCode, Description
	q := "SELECT " + pkCol + ", FieldCode, Description FROM " + mssqlTable
	rows, err := src.QueryContext(ctx, q)
	if err != nil {
		log.Printf("  WARN: query %s: %v", mssqlTable, err)
		logTable(pgTable, 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, "INSERT INTO "+pgTable+" (legacy_id, code, description) VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING",
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert %s: %v", pgTable, err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable(pgTable, srcCount, insCount, skipCount, time.Since(t))
}

func migrateDamageAreas(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G70") {
		logTable("damage_areas", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO damage_areas (legacy_id, code, description)
			VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert damage_areas: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("damage_areas", srcCount, insCount, skipCount, time.Since(t))
}

func migrateDamageTypes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G71") {
		logTable("damage_types", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO damage_types (legacy_id, code, description)
			VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert damage_types: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("damage_types", srcCount, insCount, skipCount, time.Since(t))
}

func migrateDamageSeverities(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G72") {
		logTable("damage_severities", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO damage_severities (legacy_id, code, description)
			VALUES ($1,$2,$3) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert damage_severities: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("damage_severities", srcCount, insCount, skipCount, time.Since(t))
}

func migrateChartOfAccounts(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G80") {
		logTable("chart_of_accounts", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO chart_of_accounts (legacy_id, account_type, account_name, account_num, opening_balance, opening_date)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			ni(oldID), ns(acctType), ns(acctName), ns(acctNum), nd(openBal), nt(openDate))
		if err != nil {
			log.Printf("  WARN: insert chart_of_accounts: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("chart_of_accounts", srcCount, insCount, skipCount, time.Since(t))
}

func migrateTerms(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G85") {
		logTable("terms", 0, 0, 0, time.Since(t))
		return
	}
	// Actual: G85Id, TermName, Description, DaysToPay, Discount
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
		_, err := tx.Exec(ctx, `INSERT INTO terms (legacy_id, term, description, days)
			VALUES ($1,$2,$3,$4) ON CONFLICT (term) DO NOTHING`,
			ni(oldID), termVal, ns(desc), ni(days))
		if err != nil {
			log.Printf("  WARN: insert terms: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("terms", srcCount, insCount, skipCount, time.Since(t))
}

func migrateTaxCodes(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G86") {
		logTable("tax_codes", 0, 0, 0, time.Since(t))
		return
	}
	// Actual: G86Id, TaxCodeName, Description, StateTaxRate, CountyTaxRate, CityTaxRate
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
		_, err := tx.Exec(ctx, `INSERT INTO tax_codes (legacy_id, code, description, rate)
			VALUES ($1,$2,$3,$4) ON CONFLICT (code) DO NOTHING`,
			ni(oldID), codeVal, ns(desc), nd(rate))
		if err != nil {
			log.Printf("  WARN: insert tax_codes: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("tax_codes", srcCount, insCount, skipCount, time.Since(t))
}

func migrateVendors(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G50") {
		logTable("vendors", 0, 0, 0, time.Since(t))
		return
	}
	// Actual: G50Id, Name, Address1, Address2, City, State, Zip, Phone, Fax, Contact, FedSSNumber
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
			phone, fax, contact, tax_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			ni(oldID), nameVal, ns(addr), ns(addr2), ns(city), ns(state), ns(zip),
			ns(phone), ns(fax), ns(contact), ns(taxID))
		if err != nil {
			log.Printf("  WARN: insert vendors: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("vendors", srcCount, insCount, skipCount, time.Since(t))
}

func migrateVendorGroups(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G53") {
		logTable("vendor_groups", 0, 0, 0, time.Since(t))
		return
	}
	// Actual: G53Id, Name, Description
	rows, err := src.QueryContext(ctx, "SELECT G53Id, Name, Description FROM G53")
	if err != nil {
		log.Printf("  WARN: query G53: %v", err)
		logTable("vendor_groups", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO vendor_groups (legacy_id, group_name, description)
			VALUES ($1,$2,$3)`,
			ni(oldID), nameVal, ns(desc))
		if err != nil {
			log.Printf("  WARN: insert vendor_groups: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("vendor_groups", srcCount, insCount, skipCount, time.Since(t))
}

func migrateCarriers(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G55") {
		logTable("carriers", 0, 0, 0, time.Since(t))
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
			contact, phone, fax)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (carrier_name) DO NOTHING`,
			ni(oldID), ns(linkID), nameVal, ns(addr), ns(city), ns(state), ns(zip),
			ns(contact), ns(phone), ns(fax))
		if err != nil {
			log.Printf("  WARN: insert carriers: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("carriers", srcCount, insCount, skipCount, time.Since(t))
}

func migrateZonePricing(ctx context.Context, src *sql.DB, tx pgx.Tx) {
	t := time.Now()
	if !tableExists(src, "G32") {
		logTable("zone_pricing", 0, 0, 0, time.Since(t))
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
		_, err := tx.Exec(ctx, `INSERT INTO zone_pricing (legacy_id, zone_a, zone_b, description, amount, miles, transport_days, ship_to)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (zone_a, zone_b) DO NOTHING`,
			ni(oldID), za, zb, ns(desc), nd(amount), ni(miles), ni(days), ns(shipTo))
		if err != nil {
			log.Printf("  WARN: insert zone_pricing: %v", err)
			skipCount++
			continue
		}
		insCount++
	}
	logTable("zone_pricing", srcCount, insCount, skipCount, time.Since(t))
}
