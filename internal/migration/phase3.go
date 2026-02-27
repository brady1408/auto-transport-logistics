package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

func migrateOrders(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, customerIDs IDMap, logStat func(Stat)) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT D00Id, Active, DispatchCode, PONumber, OrderReference,
		BillG00id, BillCustNumber, BillCust,
		BillAddr1, BillAddr2, BillCity, BillState, BillZip,
		LoadG00id, LoadCustNumber, LoadCust, LoadContact, LoadPhone,
		LoadAddr1, LoadAddr2, LoadCity, LoadState, LoadZip,
		DropG00id, DropCustNumber, DropCust, DropContact, DropPhone,
		DropAddr1, DropAddr2, DropCity, DropState, DropZip,
		UnitAmount, FuelSurcharge, FuelCalcType, TotalAmount, TotalOtherCharges,
		UnitCount, D10Waiting, D10Scheduled,
		OrderDate, DOInstructions, PUInstructions, DispatchNotes,
		SalesRep, Terms, TaxCode, Class,
		CreatedBy, CreatedTimeString, UpdatedBy, UpdatedTimeString FROM D00`)
	if err != nil {
		log.Fatalf("query D00: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var active sql.NullInt64
		var dispCode, poNum, orderRef sql.NullString
		var billCustID, loadCustID, dropCustID sql.NullInt64
		var billCustNum, billCust sql.NullString
		var billAddr1, billAddr2, billCity, billState, billZip sql.NullString
		var loadCustNum, loadCust, loadContact, loadPhone sql.NullString
		var loadAddr1, loadAddr2, loadCity, loadState, loadZip sql.NullString
		var dropCustNum, dropCust, dropContact, dropPhone sql.NullString
		var dropAddr1, dropAddr2, dropCity, dropState, dropZip sql.NullString
		var unitAmt, fuelSurcharge, totalAmt, totalOther sql.NullFloat64
		var fuelCalcType sql.NullString
		var unitCount, d10Waiting, d10Sched sql.NullInt64
		var orderDate, createdTime, updatedTime sql.NullTime
		var doInstr, puInstr, dispNotes sql.NullString
		var salesRep, terms, taxCode, class sql.NullString
		var createdBy, updatedBy sql.NullString
		if err := rows.Scan(&oldID, &active, &dispCode, &poNum, &orderRef,
			&billCustID, &billCustNum, &billCust,
			&billAddr1, &billAddr2, &billCity, &billState, &billZip,
			&loadCustID, &loadCustNum, &loadCust, &loadContact, &loadPhone,
			&loadAddr1, &loadAddr2, &loadCity, &loadState, &loadZip,
			&dropCustID, &dropCustNum, &dropCust, &dropContact, &dropPhone,
			&dropAddr1, &dropAddr2, &dropCity, &dropState, &dropZip,
			&unitAmt, &fuelSurcharge, &fuelCalcType, &totalAmt, &totalOther,
			&unitCount, &d10Waiting, &d10Sched,
			&orderDate, &doInstr, &puInstr, &dispNotes,
			&salesRep, &terms, &taxCode, &class,
			&createdBy, &createdTime, &updatedBy, &updatedTime); err != nil {
			log.Printf("  WARN: scan D00 row: %v", err)
			skipCount++
			continue
		}
		orderNumVal := fmt.Sprintf("ORD-%d", oldID.Int64)
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO orders (legacy_id, order_number, active, dispatch_code, po_number, reference_number,
			bill_customer_id, bill_customer_number, bill_customer_name,
			bill_to_address, bill_to_address2, bill_to_city, bill_to_state, bill_to_zip,
			load_customer_id, load_customer_number, load_customer_name, load_contact, load_phone,
			load_address, load_address2, load_city, load_state, load_zip,
			drop_customer_id, drop_customer_number, drop_customer_name, drop_contact, drop_phone,
			drop_address, drop_address2, drop_city, drop_state, drop_zip,
			transport_amt, fuel_surcharge, fuel_calc_type, total_charge, other_charge,
			vehicle_count, waiting_count, scheduled_count,
			create_date, do_instructions, pu_instructions, comments,
			sales_rep1, equipment_type, tax_code, edit_by,
			edit_date, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,
			$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52)
			ON CONFLICT (company_id, order_number) DO NOTHING
			RETURNING id`,
			ni(oldID), orderNumVal, nb(active), ns(dispCode), ns(poNum), ns(orderRef),
			lookupFK(customerIDs, billCustID), ns(billCustNum), ns(billCust),
			ns(billAddr1), ns(billAddr2), ns(billCity), ns(billState), ns(billZip),
			lookupFK(customerIDs, loadCustID), ns(loadCustNum), ns(loadCust), ns(loadContact), ns(loadPhone),
			ns(loadAddr1), ns(loadAddr2), ns(loadCity), ns(loadState), ns(loadZip),
			lookupFK(customerIDs, dropCustID), ns(dropCustNum), ns(dropCust), ns(dropContact), ns(dropPhone),
			ns(dropAddr1), ns(dropAddr2), ns(dropCity), ns(dropState), ns(dropZip),
			nd(unitAmt), nd(fuelSurcharge), ns(fuelCalcType), nd(totalAmt), nd(totalOther),
			nint(unitCount), nint(d10Waiting), nint(d10Sched),
			nt(orderDate), ns(doInstr), ns(puInstr), ns(dispNotes),
			ns(salesRep), ns(class), ns(taxCode), ns(updatedBy),
			nt(updatedTime), companyID).Scan(&newID)
		if err != nil {
			if err.Error() == "no rows in result set" {
				skipCount++
				continue
			}
			log.Printf("  WARN: insert orders (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logStat(Stat{Table: "orders", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
	return ids
}

func migrateTrips(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, truckIDs, employeeIDs IDMap, logStat func(Stat)) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT D20Id, LoadNumber, Active, TruckNumber,
		Driver1, Drv1G10Id, Driver2, Drv2G10Id,
		StartDate, EndDate, EstEndDate,
		TripMiles, TruckRate, TruckCalcType,
		Rate1, Rate1CalcType, AddRate1, AddRate1CalcType,
		Comments, DispatchCode FROM D20`)
	if err != nil {
		log.Fatalf("query D20: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	seen := make(map[string]bool)
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var loadNum, truckNum sql.NullString
		var active sql.NullInt64
		var driver1, driver2 sql.NullString
		var drv1ID, drv2ID sql.NullInt64
		var startDate, endDate, estEndDate sql.NullTime
		var tripMiles sql.NullInt64
		var truckRate, rate1, addRate1 sql.NullFloat64
		var truckCalcType, rate1CalcType, addRate1CalcType sql.NullString
		var comments, dispCode sql.NullString
		if err := rows.Scan(&oldID, &loadNum, &active, &truckNum,
			&driver1, &drv1ID, &driver2, &drv2ID,
			&startDate, &endDate, &estEndDate,
			&tripMiles, &truckRate, &truckCalcType,
			&rate1, &rate1CalcType, &addRate1, &addRate1CalcType,
			&comments, &dispCode); err != nil {
			log.Printf("  WARN: scan D20 row: %v", err)
			skipCount++
			continue
		}
		loadNumVal := nns(loadNum)
		if loadNumVal == "" {
			loadNumVal = fmt.Sprintf("TRP-%d", oldID.Int64)
		}
		if seen[loadNumVal] {
			log.Printf("  WARN: dup load_number '%s' (legacy %d)", loadNumVal, oldID.Int64)
			skipCount++
			continue
		}
		seen[loadNumVal] = true
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO trips (legacy_id, load_number, active, truck_number,
			driver, driver1_id, driver2, driver2_id,
			trip_date, deliver_date, est_deliver_date,
			total_mileage, truck_rate, truck_calc_type,
			driver_rate, driver_calc_type, driver_add_rate, driver_add_calc_type,
			comments, status, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
			RETURNING id`,
			ni(oldID), loadNumVal, nb(active), ns(truckNum),
			ns(driver1), lookupFK(employeeIDs, drv1ID), ns(driver2), lookupFK(employeeIDs, drv2ID),
			nt(startDate), nt(endDate), nt(estEndDate),
			ni(tripMiles), nd(truckRate), ns(truckCalcType),
			nd(rate1), ns(rate1CalcType), nd(addRate1), ns(addRate1CalcType),
			ns(comments), ns(dispCode), companyID).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert trips (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logStat(Stat{Table: "trips", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
	return ids
}

func migrateOrderVehicles(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, orderIDs, tripIDs IDMap, logStat func(Stat)) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT D10Id, D00Id, D20Id, A00Id,
		VIN, Year, Make, Model, Color,
		TransportAmount, Status, InvoiceNumber, FuelSurcharge, FuelCalcType FROM D10`)
	if err != nil {
		log.Fatalf("query D10: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d00id, d20id, a00id sql.NullInt64
		var vin, year, make_, model, color sql.NullString
		var transportAmt, fuelSurcharge sql.NullFloat64
		var status, invNum, fuelCalcType sql.NullString
		if err := rows.Scan(&oldID, &d00id, &d20id, &a00id,
			&vin, &year, &make_, &model, &color,
			&transportAmt, &status, &invNum, &fuelSurcharge, &fuelCalcType); err != nil {
			log.Printf("  WARN: scan D10 row: %v", err)
			skipCount++
			continue
		}
		orderID := lookupFK(orderIDs, d00id)
		if orderID == nil {
			skipCount++
			continue
		}
		statusVal := nns(status)
		if statusVal == "" {
			statusVal = "Waiting"
		}
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO order_vehicles (legacy_id, order_id, trip_id,
			vin, year, make, model, color,
			transport_amt, status, invoice_number, fuel_surcharge, fuel_calc_type, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			RETURNING id`,
			ni(oldID), *orderID, lookupFK(tripIDs, d20id),
			ns(vin), ns(year), ns(make_), ns(model), ns(color),
			nd(transportAmt), statusVal, ns(invNum), nd(fuelSurcharge), ns(fuelCalcType), companyID).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert order_vehicles (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logStat(Stat{Table: "order_vehicles", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
	return ids
}

func migrateLoadDetails(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, tripIDs, orderIDs, vehicleIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	rows, err := src.QueryContext(ctx, `SELECT D30Id, D20Id, D00Id, D10Id,
		Weight, LoadPosition, Status, LoadTimeString, DropTimeString FROM D30`)
	if err != nil {
		log.Fatalf("query D30: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d20id, d00id, d10id sql.NullInt64
		var weight, loadPos sql.NullInt64
		var status sql.NullString
		var loadedDate, delivDate sql.NullTime
		if err := rows.Scan(&oldID, &d20id, &d00id, &d10id,
			&weight, &loadPos, &status, &loadedDate, &delivDate); err != nil {
			skipCount++
			continue
		}
		tripID := lookupFK(tripIDs, d20id)
		if tripID == nil {
			skipCount++
			continue
		}
		bayStr := ""
		if loadPos.Valid && loadPos.Int64 > 0 {
			bayStr = fmt.Sprintf("%d", loadPos.Int64)
		}
		var bayPtr *string
		if bayStr != "" {
			bayPtr = &bayStr
		}
		_, err := tx.Exec(ctx, `INSERT INTO load_details (legacy_id, trip_id, order_id, vehicle_id,
			weight, bay_number, status, loaded_date, delivered_date, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			ni(oldID), *tripID, lookupFK(orderIDs, d00id), lookupFK(vehicleIDs, d10id),
			ni(weight), bayPtr, ns(status), nt(loadedDate), nt(delivDate), companyID)
		if err != nil {
			log.Printf("  WARN: insert load_details (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "load_details", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateOrderCharges(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, orderIDs, vehicleIDs, tripIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	rows, err := src.QueryContext(ctx, `SELECT D13Id, D10Id, D20Id, Description, Amount, ARAmount FROM D13`)
	if err != nil {
		log.Fatalf("query D13: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d10id, d20id sql.NullInt64
		var desc sql.NullString
		var amount, arAmount sql.NullFloat64
		if err := rows.Scan(&oldID, &d10id, &d20id, &desc, &amount, &arAmount); err != nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO order_charges (legacy_id, vehicle_id, trip_id, description, amount, company_id)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			ni(oldID), lookupFK(vehicleIDs, d10id), lookupFK(tripIDs, d20id),
			ns(desc), nd(amount), companyID)
		if err != nil {
			log.Printf("  WARN: insert order_charges (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "order_charges", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateVehicleDamage(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, orderIDs, vehicleIDs, tripIDs IDMap, logStat func(Stat)) IDMap {
	t := time.Now()
	idm := make(IDMap)
	if !tableExists(src, "D33") {
		logStat(Stat{Table: "vehicle_damage", Elapsed: time.Since(t)})
		return idm
	}
	rows, err := src.QueryContext(ctx, `SELECT D33Id, D10Id, D30Id,
		DamageArea, DamageType, DamageSeverity, Comments,
		InspectedBy, InspectedDate, Location FROM D33`)
	if err != nil {
		log.Fatalf("query D33: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d10id, d30id sql.NullInt64
		var area, typ, sev, comments sql.NullString
		var inspBy sql.NullString
		var inspDate sql.NullTime
		var location sql.NullString
		if err := rows.Scan(&oldID, &d10id, &d30id,
			&area, &typ, &sev, &comments,
			&inspBy, &inspDate, &location); err != nil {
			skipCount++
			continue
		}
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO vehicle_damage (legacy_id, vehicle_id,
			damage_area, damage_type, damage_severity, description,
			inspected_by, inspection_date, inspection_point, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id`,
			ni(oldID), lookupFK(vehicleIDs, d10id),
			ns(area), ns(typ), ns(sev), ns(comments),
			ns(inspBy), nt(inspDate), ns(location), companyID).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert vehicle_damage (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			idm[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logStat(Stat{Table: "vehicle_damage", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
	return idm
}

func migrateDamageDetails(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, damageIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "D34") {
		logStat(Stat{Table: "damage_details", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT D34Id, D33Id, DamageArea, DamageType, DamageSeverity FROM D34`)
	if err != nil {
		log.Fatalf("query D34: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d33id sql.NullInt64
		var area, typ, sev sql.NullString
		if err := rows.Scan(&oldID, &d33id, &area, &typ, &sev); err != nil {
			skipCount++
			continue
		}
		dmgID := lookupFK(damageIDs, d33id)
		if dmgID == nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO damage_details (legacy_id, vehicle_damage_id, damage_area, damage_type, damage_severity, company_id)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			ni(oldID), *dmgID, ns(area), ns(typ), ns(sev), companyID)
		if err != nil {
			log.Printf("  WARN: insert damage_details (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "damage_details", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateVehicleNotes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, vehicleIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "D11") {
		logStat(Stat{Table: "vehicle_notes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT D11Id, D10Id, NoteDate, Description, Comment, CreatedBy FROM D11`)
	if err != nil {
		log.Fatalf("query D11: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d10id sql.NullInt64
		var noteDate sql.NullTime
		var desc, comment, createdBy sql.NullString
		if err := rows.Scan(&oldID, &d10id, &noteDate, &desc, &comment, &createdBy); err != nil {
			skipCount++
			continue
		}
		vehID := lookupFK(vehicleIDs, d10id)
		if vehID == nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO vehicle_notes (legacy_id, vehicle_id, note_date, description, comment, created_by, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			ni(oldID), *vehID, nt(noteDate), ns(desc), ns(comment), ns(createdBy), companyID)
		if err != nil {
			log.Printf("  WARN: insert vehicle_notes (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "vehicle_notes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateTripFuel(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, tripIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "D23") {
		logStat(Stat{Table: "trip_fuel", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT D23Id, D20Id, LoadedMiles, TruckNumber, State, Mileage, Gallons FROM D23`)
	if err != nil {
		log.Fatalf("query D23: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d20id, loadedMiles, mileage sql.NullInt64
		var truckNum, state sql.NullString
		var gallons sql.NullFloat64
		if err := rows.Scan(&oldID, &d20id, &loadedMiles, &truckNum, &state, &mileage, &gallons); err != nil {
			skipCount++
			continue
		}
		tripID := lookupFK(tripIDs, d20id)
		if tripID == nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO trip_fuel (legacy_id, trip_id, loaded_miles, truck_number, state, mileage, gallons, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			ni(oldID), *tripID, nb(loadedMiles), ns(truckNum), ns(state), ni(mileage), nd(gallons), companyID)
		if err != nil {
			log.Printf("  WARN: insert trip_fuel (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "trip_fuel", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateTripExpenses(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, tripIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "D24") {
		logStat(Stat{Table: "trip_expenses", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT D24Id, D20Id, Description, Amount, ExpenseDate FROM D24`)
	if err != nil {
		log.Fatalf("query D24: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d20id sql.NullInt64
		var desc sql.NullString
		var amount sql.NullFloat64
		var expDate sql.NullTime
		if err := rows.Scan(&oldID, &d20id, &desc, &amount, &expDate); err != nil {
			skipCount++
			continue
		}
		tripID := lookupFK(tripIDs, d20id)
		if tripID == nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO trip_expenses (legacy_id, trip_id, description, amount, expense_date, company_id)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			ni(oldID), *tripID, ns(desc), nd(amount), nt(expDate), companyID)
		if err != nil {
			log.Printf("  WARN: insert trip_expenses (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "trip_expenses", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateTripRoutes(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, tripIDs, customerIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "D26") {
		logStat(Stat{Table: "trip_routes", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT D26Id, D20Id, SeqNumber, StopCity, StopState,
		StopType, Mileage, StopDateString FROM D26`)
	if err != nil {
		log.Fatalf("query D26: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d20id, seq, miles sql.NullInt64
		var city, state, stopType sql.NullString
		var estArr sql.NullTime
		if err := rows.Scan(&oldID, &d20id, &seq, &city, &state,
			&stopType, &miles, &estArr); err != nil {
			skipCount++
			continue
		}
		tripID := lookupFK(tripIDs, d20id)
		if tripID == nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO trip_routes (legacy_id, trip_id, sequence, city, state,
			stop_type, miles, est_arrival, company_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			ni(oldID), *tripID, ni(seq), ns(city), ns(state),
			ns(stopType), ni(miles), nt(estArr), companyID)
		if err != nil {
			log.Printf("  WARN: insert trip_routes (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "trip_routes", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}

func migrateSplitLoads(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, orderIDs, vehicleIDs, tripIDs IDMap, logStat func(Stat)) {
	t := time.Now()
	if !tableExists(src, "D40") {
		logStat(Stat{Table: "split_loads", Elapsed: time.Since(t)})
		return
	}
	rows, err := src.QueryContext(ctx, `SELECT D40Id, D00Id, D20Id, SplitDate, Comments FROM D40`)
	if err != nil {
		log.Fatalf("query D40: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, d00id, d20id sql.NullInt64
		var splitDate sql.NullTime
		var comments sql.NullString
		if err := rows.Scan(&oldID, &d00id, &d20id, &splitDate, &comments); err != nil {
			skipCount++
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO split_loads (legacy_id, order_id, trip_id, split_date, reason, company_id)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			ni(oldID), lookupFK(orderIDs, d00id), lookupFK(tripIDs, d20id),
			nt(splitDate), ns(comments), companyID)
		if err != nil {
			log.Printf("  WARN: insert split_loads (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		insCount++
	}
	logStat(Stat{Table: "split_loads", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}
