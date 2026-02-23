package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

func migrateCustomers(ctx context.Context, src *sql.DB, tx pgx.Tx) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT G00Id, Number, COD, Inactive, Name, Address, Address2,
		City, State, Zip, Phone, Mobile, Fax, Contact, Zone, Type,
		CreditLimit, CreditTerms, CombineInvDetLine, FuelSurcharge, SPLC, RateClass, RouteCode,
		Comments, DOInstructions, PUInstructions, FuelCalcType, SalesRep, SalesDate,
		RevenueClass, Terms, TaxCode, LocationType, Discount, DiscountCalcType FROM G00`)
	if err != nil {
		log.Fatalf("query G00: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	for rows.Next() {
		srcCount++
		var oldID, cod, inactive, combineInv sql.NullInt64
		var number, name, addr, addr2, city, state, zip, phone, mobile, fax sql.NullString
		var contact, zone, typ sql.NullString
		var creditLimit, fuelSurcharge, discount sql.NullFloat64
		var creditTerms, splc, rateClass, routeCode sql.NullString
		var comments, doInstr, puInstr, fuelCalcType, salesRep sql.NullString
		var salesDate sql.NullTime
		var revenueClass, terms, taxCode, locationType, discCalcType sql.NullString
		if err := rows.Scan(&oldID, &number, &cod, &inactive, &name, &addr, &addr2,
			&city, &state, &zip, &phone, &mobile, &fax, &contact, &zone, &typ,
			&creditLimit, &creditTerms, &combineInv, &fuelSurcharge, &splc, &rateClass, &routeCode,
			&comments, &doInstr, &puInstr, &fuelCalcType, &salesRep, &salesDate,
			&revenueClass, &terms, &taxCode, &locationType, &discount, &discCalcType); err != nil {
			log.Printf("  WARN: scan G00 row: %v", err)
			skipCount++
			continue
		}
		nameVal := nns(name)
		if nameVal == "" {
			skipCount++
			continue
		}
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO customers (legacy_id, number, cod, inactive, name, address, address2,
			city, state, zip, phone, mobile, fax, contact, zone, type,
			credit_limit, credit_terms, combine_inv_det_line, fuel_surcharge, splc, rate_class, route_code,
			comments, do_instructions, pu_instructions, fuel_calc_type, sales_rep, sales_date,
			revenue_class, terms, tax_code, location_type, discount, discount_calc_type)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35)
			ON CONFLICT (type, number) DO NOTHING
			RETURNING id`,
			ni(oldID), ns(number), nb(cod), nb(inactive), nameVal, ns(addr), ns(addr2),
			ns(city), ns(state), ns(zip), ns(phone), ns(mobile), ns(fax),
			ns(contact), ns(zone), ns(typ),
			nd(creditLimit), ns(creditTerms), nb(combineInv), nd(fuelSurcharge),
			ns(splc), ns(rateClass), ns(routeCode),
			ns(comments), ns(doInstr), ns(puInstr), ns(fuelCalcType),
			ns(salesRep), nt(salesDate),
			ns(revenueClass), ns(terms), ns(taxCode), ns(locationType),
			nd(discount), ns(discCalcType)).Scan(&newID)
		if err != nil {
			if err.Error() == "no rows in result set" {
				skipCount++
				continue
			}
			log.Printf("  WARN: insert customers (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logTable("customers", srcCount, insCount, skipCount, time.Since(t))
	return ids
}

func migrateEmployees(ctx context.Context, src *sql.DB, tx pgx.Tx) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT G10Id, Name, Address, Address2, City, State, Zip, Phone,
		Rate, Reserve, EmploymentDate, TerminationDate,
		EmergencyContact, EmergencyPhone, ComDataNumber, DriversLicenseNumber, DriversLicenseState,
		StateDrivingRec, StateDrivingRecExp, DrivingRecReview, DrivingRecReviewExp,
		CopyOfCDL, CDLExp, CopyOfMedCert, MedCertExp, DOTApplication, DOTApplicationExp,
		PriorEmpChk, LastServiceHrs, PreEmpDrugTest, PrevEmpInquiries, ReceiptDrugPolicy,
		W4EmpWithholding, USLegalInfo, SSNumber, Active, IsDriver, IsSales,
		RateCalcType, AddRate, AddRateCalcType,
		SalesRate1, SalesRate1Type, SalesRate1Duration,
		SalesRate2, SalesRate2Type, SalesRate2Duration,
		EmpIdNumber, UserName, BirthDate FROM G10`)
	if err != nil {
		log.Fatalf("query G10: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	seen := make(map[string]bool)
	for rows.Next() {
		srcCount++
		var oldID sql.NullInt64
		var name, addr, addr2, city, state, zip, phone sql.NullString
		var rate, reserve, addRate, sr1, sr2 sql.NullFloat64
		var empDate, termDate sql.NullTime
		var emContact, emPhone, comData, dlNum, dlState sql.NullString
		var stateDR, drvReview, cdl, medCert, dotApp sql.NullInt64
		var stateDRExp, drvReviewExp, cdlExp, medCertExp, dotAppExp sql.NullTime
		var priorEmp, lastSvc, preDrug, prevEmp, drugPolicy sql.NullInt64
		var w4, usLegal sql.NullInt64
		var ssn sql.NullString
		var active, isDriver, isSales sql.NullInt64
		var rateCalcType, addRateCalcType sql.NullString
		var sr1Type, sr2Type sql.NullString
		var sr1Dur, sr2Dur sql.NullInt64
		var empIDNum, userName sql.NullString
		var birthDate sql.NullTime
		if err := rows.Scan(&oldID, &name, &addr, &addr2, &city, &state, &zip, &phone,
			&rate, &reserve, &empDate, &termDate,
			&emContact, &emPhone, &comData, &dlNum, &dlState,
			&stateDR, &stateDRExp, &drvReview, &drvReviewExp,
			&cdl, &cdlExp, &medCert, &medCertExp, &dotApp, &dotAppExp,
			&priorEmp, &lastSvc, &preDrug, &prevEmp, &drugPolicy,
			&w4, &usLegal, &ssn, &active, &isDriver, &isSales,
			&rateCalcType, &addRate, &addRateCalcType,
			&sr1, &sr1Type, &sr1Dur,
			&sr2, &sr2Type, &sr2Dur,
			&empIDNum, &userName, &birthDate); err != nil {
			log.Printf("  WARN: scan G10 row: %v", err)
			skipCount++
			continue
		}
		nameVal := nns(name)
		if nameVal == "" {
			skipCount++
			continue
		}
		if seen[nameVal] {
			log.Printf("  WARN: dup employee name '%s' (legacy %d)", nameVal, oldID.Int64)
			skipCount++
			continue
		}
		seen[nameVal] = true
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO employees (legacy_id, name, address, address2, city, state, zip, phone,
			rate, reserve, employment_date, termination_date,
			emergency_contact, emergency_phone, com_data_number, drivers_license_number, drivers_license_state,
			state_driving_rec, state_driving_rec_exp, driving_rec_review, driving_rec_review_exp,
			copy_of_cdl, cdl_exp, copy_of_med_cert, med_cert_exp, dot_application, dot_application_exp,
			prior_emp_chk, last_service_hrs, pre_emp_drug_test, prev_emp_inquiries, receipt_drug_policy,
			w4_emp_withholding, us_legal_info, ssn, active, is_driver, is_sales,
			rate_calc_type, add_rate, add_rate_calc_type,
			sales_rate1, sales_rate1_type, sales_rate1_duration,
			sales_rate2, sales_rate2_type, sales_rate2_duration,
			emp_id_number, username, birth_date)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,
			$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,
			$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50)
			RETURNING id`,
			ni(oldID), nameVal, ns(addr), ns(addr2), ns(city), ns(state), ns(zip), ns(phone),
			nd(rate), nd(reserve), nt(empDate), nt(termDate),
			ns(emContact), ns(emPhone), ns(comData), ns(dlNum), ns(dlState),
			nb(stateDR), nt(stateDRExp), nb(drvReview), nt(drvReviewExp),
			nb(cdl), nt(cdlExp), nb(medCert), nt(medCertExp), nb(dotApp), nt(dotAppExp),
			nb(priorEmp), nb(lastSvc), nb(preDrug), nb(prevEmp), nb(drugPolicy),
			nb(w4), nb(usLegal), ns(ssn), nb(active), nb(isDriver), nb(isSales),
			ns(rateCalcType), nd(addRate), ns(addRateCalcType),
			nd(sr1), ns(sr1Type), ni(sr1Dur),
			nd(sr2), ns(sr2Type), ni(sr2Dur),
			ns(empIDNum), ns(userName), nt(birthDate)).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert employees (legacy %d, name %s): %v", oldID.Int64, nameVal, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logTable("employees", srcCount, insCount, skipCount, time.Since(t))
	return ids
}

func migrateTrucks(ctx context.Context, src *sql.DB, tx pgx.Tx) IDMap {
	t := time.Now()
	ids := make(IDMap)
	rows, err := src.QueryContext(ctx, `SELECT G20Id, TruckNumber, TruckMake, TruckModel, TruckYear,
		TruckSerialNumber, TruckManufactureDate, TruckLicense, TruckLicenseExp,
		TruckSafetyInspection, TrailerNumber, TrailerMake, TrailerModel, TrailerYear,
		TrailerSerialNumber, TrailerManufactureDate, TrailerLicense, TrailerLicenseExp,
		TrailerSafetyInspection, TareWeight,
		TruckPurchasedFrom, TruckPurchaseDate, TruckCost,
		TrailerPurchasedFrom, TrailerPurchaseDate, TrailerCost,
		FinancedBy, NoteAmount, OwnedBy, InsuranceExpDate, InsuranceCoverageAmt,
		LoanDate, LoanTerm, ContractEndDate, LoanAccount,
		TruckRate, TruckCalcType, LeasedTruck, WePayDriver,
		Driver1, Driver2, FleetNumber,
		EngineModel, EngineSerialNumber, TransModel, RearEndModel, RearEndRatio,
		EngineWarrMiles, EngingWarrYears, TransWarrMiles, TransWarrYears,
		RearEndWarrMiles, RearEndWarrYears, ClimateWarrMiles, ClimateWarrYears,
		ElectricalWarrMiles, ElectricalWarrYears, TowingWarrMiles, TowingWarrYears,
		WarrantyNotes,
		SteerTireModel, SteerTireSize, DriveTireModel, DriveTireSize,
		TrailerTireModel, TrailerTireSize,
		Active, Class, Straps, ExcludeFuel, CargoCoverageAmt,
		W9Date, WorkersCompDate, CarrierAgreementDate FROM G20`)
	if err != nil {
		log.Fatalf("query G20: %v", err)
	}
	defer rows.Close()
	srcCount, insCount, skipCount := 0, 0, 0
	seen := make(map[string]bool)
	for rows.Next() {
		srcCount++
		var oldID, tareWeight sql.NullInt64
		var truckNum, truckMake, truckModel, truckYear, truckSerial sql.NullString
		var truckMfgDate, truckLicExp, truckSafety sql.NullTime
		var truckLic sql.NullString
		var trailerNum, trailerMake, trailerModel, trailerYear, trailerSerial sql.NullString
		var trailerMfgDate, trailerLicExp, trailerSafety sql.NullTime
		var trailerLic sql.NullString
		var truckPurchFrom sql.NullString
		var truckPurchDate sql.NullTime
		var truckCost sql.NullFloat64
		var trailerPurchFrom sql.NullString
		var trailerPurchDate sql.NullTime
		var trailerCost sql.NullFloat64
		var financedBy sql.NullString
		var noteAmt, insCovAmt, truckRate, cargoCov sql.NullFloat64
		var ownedBy sql.NullString
		var insExpDate, loanDate, contractEnd sql.NullTime
		var loanTerm sql.NullInt64
		var loanAcct, truckCalcType sql.NullString
		var leased, wePayDriver sql.NullInt64
		var driver1, driver2, fleetNum sql.NullString
		var engModel, engSerial, transModel, rearModel, rearRatio sql.NullString
		var engWarrMi, engWarrYr, transWarrMi, transWarrYr sql.NullInt64
		var rearWarrMi, rearWarrYr, climWarrMi, climWarrYr sql.NullInt64
		var elecWarrMi, elecWarrYr, towWarrMi, towWarrYr sql.NullInt64
		var warrNotes sql.NullString
		var steerTireModel, steerTireSize, driveTireModel, driveTireSize sql.NullString
		var trailerTireModel, trailerTireSize sql.NullString
		var active sql.NullInt64
		var class sql.NullString
		var straps, excludeFuel sql.NullInt64
		var w9Date, wcDate, caDate sql.NullTime
		if err := rows.Scan(&oldID, &truckNum, &truckMake, &truckModel, &truckYear,
			&truckSerial, &truckMfgDate, &truckLic, &truckLicExp,
			&truckSafety, &trailerNum, &trailerMake, &trailerModel, &trailerYear,
			&trailerSerial, &trailerMfgDate, &trailerLic, &trailerLicExp,
			&trailerSafety, &tareWeight,
			&truckPurchFrom, &truckPurchDate, &truckCost,
			&trailerPurchFrom, &trailerPurchDate, &trailerCost,
			&financedBy, &noteAmt, &ownedBy, &insExpDate, &insCovAmt,
			&loanDate, &loanTerm, &contractEnd, &loanAcct,
			&truckRate, &truckCalcType, &leased, &wePayDriver,
			&driver1, &driver2, &fleetNum,
			&engModel, &engSerial, &transModel, &rearModel, &rearRatio,
			&engWarrMi, &engWarrYr, &transWarrMi, &transWarrYr,
			&rearWarrMi, &rearWarrYr, &climWarrMi, &climWarrYr,
			&elecWarrMi, &elecWarrYr, &towWarrMi, &towWarrYr,
			&warrNotes,
			&steerTireModel, &steerTireSize, &driveTireModel, &driveTireSize,
			&trailerTireModel, &trailerTireSize,
			&active, &class, &straps, &excludeFuel, &cargoCov,
			&w9Date, &wcDate, &caDate); err != nil {
			log.Printf("  WARN: scan G20 row: %v", err)
			skipCount++
			continue
		}
		truckNumVal := nns(truckNum)
		if truckNumVal == "" {
			skipCount++
			continue
		}
		if seen[truckNumVal] {
			log.Printf("  WARN: dup truck_number '%s' (legacy %d)", truckNumVal, oldID.Int64)
			skipCount++
			continue
		}
		seen[truckNumVal] = true
		var newID int
		err := tx.QueryRow(ctx, `INSERT INTO trucks (legacy_id, truck_number, truck_make, truck_model, truck_year,
			truck_serial_number, truck_manufacture_date, truck_license, truck_license_exp,
			truck_safety_inspection, trailer_number, trailer_make, trailer_model, trailer_year,
			trailer_serial_number, trailer_manufacture_date, trailer_license, trailer_license_exp,
			trailer_safety_inspection, tare_weight,
			truck_purchased_from, truck_purchase_date, truck_cost,
			trailer_purchased_from, trailer_purchase_date, trailer_cost,
			financed_by, note_amount, owned_by, insurance_exp_date, insurance_coverage_amt,
			loan_date, loan_term, contract_end_date, loan_account,
			truck_rate, truck_calc_type, leased_truck, we_pay_driver,
			driver1, driver2, fleet_number,
			engine_model, engine_serial_number, trans_model, rear_end_model, rear_end_ratio,
			engine_warr_miles, engine_warr_years, trans_warr_miles, trans_warr_years,
			rear_end_warr_miles, rear_end_warr_years, climate_warr_miles, climate_warr_years,
			electrical_warr_miles, electrical_warr_years, towing_warr_miles, towing_warr_years,
			warranty_notes,
			steer_tire_model, steer_tire_size, drive_tire_model, drive_tire_size,
			trailer_tire_model, trailer_tire_size,
			active, class, straps, exclude_fuel, cargo_coverage_amt,
			w9_date, workers_comp_date, carrier_agreement_date)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,
			$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,
			$60,$61,$62,$63,$64,$65,$66,$67,$68,$69,$70,$71,$72,$73,$74)
			RETURNING id`,
			ni(oldID), truckNumVal, ns(truckMake), ns(truckModel), ns(truckYear),
			ns(truckSerial), nt(truckMfgDate), ns(truckLic), nt(truckLicExp),
			nt(truckSafety), ns(trailerNum), ns(trailerMake), ns(trailerModel), ns(trailerYear),
			ns(trailerSerial), nt(trailerMfgDate), ns(trailerLic), nt(trailerLicExp),
			nt(trailerSafety), ni(tareWeight),
			ns(truckPurchFrom), nt(truckPurchDate), nd(truckCost),
			ns(trailerPurchFrom), nt(trailerPurchDate), nd(trailerCost),
			ns(financedBy), nd(noteAmt), ns(ownedBy), nt(insExpDate), nd(insCovAmt),
			nt(loanDate), ni(loanTerm), nt(contractEnd), ns(loanAcct),
			nd(truckRate), ns(truckCalcType), nb(leased), nb(wePayDriver),
			ns(driver1), ns(driver2), ns(fleetNum),
			ns(engModel), ns(engSerial), ns(transModel), ns(rearModel), ns(rearRatio),
			ni(engWarrMi), ni(engWarrYr), ni(transWarrMi), ni(transWarrYr),
			ni(rearWarrMi), ni(rearWarrYr), ni(climWarrMi), ni(climWarrYr),
			ni(elecWarrMi), ni(elecWarrYr), ni(towWarrMi), ni(towWarrYr),
			ns(warrNotes),
			ns(steerTireModel), ns(steerTireSize), ns(driveTireModel), ns(driveTireSize),
			ns(trailerTireModel), ns(trailerTireSize),
			nb(active), ns(class), nb(straps), nb(excludeFuel), nd(cargoCov),
			nt(w9Date), nt(wcDate), nt(caDate)).Scan(&newID)
		if err != nil {
			log.Printf("  WARN: insert trucks (legacy %d): %v", oldID.Int64, err)
			skipCount++
			continue
		}
		if oldID.Valid {
			ids[int(oldID.Int64)] = newID
		}
		insCount++
	}
	logTable("trucks", srcCount, insCount, skipCount, time.Since(t))
	return ids
}
