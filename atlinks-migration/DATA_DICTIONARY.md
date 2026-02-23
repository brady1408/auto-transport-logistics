# ATLinks Complete Data Dictionary

Extracted from `SQLATLinks/ATLinks.TXD` - Clarion 6 data dictionary.
All tables use MSSQL driver unless noted. All primary keys are LONG with AUTO increment.

---

## C00 - Company Master (-> `companies`)

**Keys:** K_C00Id PRIMARY (C00Id)

| Column | Type | Notes |
|--------|------|-------|
| C00Id | LONG | PK |
| CompanyName | CSTRING(41) | |
| Address | CSTRING(31) | |
| Address2 | CSTRING(31) | |
| City | CSTRING(26) | |
| State | CSTRING(3) | |
| Zip | CSTRING(11) | |
| Phone | CSTRING(11) | |
| Fax | CSTRING(11) | |
| SCAC | CSTRING(5) | Standard Carrier Alpha Code |
| FederalId | CSTRING(16) | |
| MCNumber | CSTRING(16) | Motor Carrier number |
| DOTNumber | CSTRING(16) | |
| SPLC | CSTRING(11) | Standard Point Location Code |
| InsuranceCarrier | CSTRING(41) | |
| InsurancePolicyNumber | CSTRING(21) | |
| InsuranceAgent | CSTRING(31) | |
| InsurancePhone | CSTRING(11) | |
| InsuranceFax | CSTRING(11) | |
| InsuranceExpDate | DATE | |
| InsuranceCoverageAmt | DECIMAL(9,2) | |

---

## G00 - Customer Master (-> `customers`)

**Keys:**
- K_G00id: PRIMARY (G00id)
- K_State: DUP (State, City, Name)
- K_Name: DUP (Name)
- K_Number: DUP (Number)
- K_Type: DUP (Type, Name)
- K_TypeNumber: UNIQUE (Type, Number)

| Column | Type | Notes |
|--------|------|-------|
| G00id | LONG | PK |
| Number | CSTRING(11) | Customer number |
| COD | BYTE | Boolean |
| Inactive | BYTE | Boolean |
| Name | CSTRING(31) | |
| Address | CSTRING(31) | |
| Address2 | CSTRING(31) | |
| City | CSTRING(26) | |
| State | CSTRING(3) | |
| Zip | CSTRING(11) | |
| Phone | CSTRING(11) | |
| Mobile | CSTRING(11) | |
| Pager | CSTRING(11) | |
| Fax | CSTRING(11) | |
| Data | CSTRING(11) | |
| Contact | CSTRING(21) | |
| Zone | CSTRING(21) | Zone ref |
| Type | CSTRING(11) | Customer type |
| CreditLimit | DECIMAL(7,2) | |
| CreditTerms | CSTRING(11) | COD/Net10/Net30 |
| QBListId | CSTRING(37) | QuickBooks |
| CombineInvDetLine | BYTE | Boolean |
| FuelSurcharge | DECIMAL(7,4) | |
| SPLC | CSTRING(11) | |
| RateClass | CSTRING(11) | |
| RouteCode | CSTRING(21) | |
| PrevSPLC | CSTRING(11) | |
| Comments | CSTRING(501) | |
| DOInstructions | CSTRING(501) | Drop-off instructions |
| FuelCalcType | CSTRING(11) | |
| SalesRep | CSTRING(31) | |
| SalesDate | DATE | |
| RevenueClass | CSTRING(21) | |
| Terms | CSTRING(21) | |
| TaxCode | CSTRING(21) | |
| PUInstructions | CSTRING(501) | Pick-up instructions |
| LocationType | CSTRING(2) | |
| Discount | DECIMAL(7,2) | |
| DiscountCalcType | CSTRING(12) | |

---

## G10 - Employee Master (-> `employees`)

**Keys:**
- K_G10id: PRIMARY (G10id)
- K_NAME: UNIQUE (Name)
- K_ActiveName: DUP (Active, Name)
- K_EmpIdNumber: DUP (EmpIdNumber, G10id)
- K_UserName: UNIQUE (UserName)

| Column | Type | Notes |
|--------|------|-------|
| G10id | LONG | PK |
| G50id | LONG | FK to Vendor |
| QBAccount | CSTRING(7) | |
| Name | CSTRING(31) | |
| Address | CSTRING(31) | |
| Address2 | CSTRING(31) | |
| City | CSTRING(26) | |
| State | CSTRING(3) | UPPER |
| Zip | CSTRING(11) | |
| Phone | CSTRING(11) | |
| Rate | DECIMAL(5,2) | Pay rate |
| Reserve | DECIMAL(5,2) | |
| EmploymentDate | DATE | |
| TerminationDate | DATE | |
| EmergencyContact | CSTRING(31) | |
| EmergencyPhone | CSTRING(11) | |
| ComDataNumber | CSTRING(41) | |
| DriversLicenseNumber | CSTRING(21) | |
| StateDrivingRec | BYTE | Boolean |
| StateDrivingRecExp | DATE | |
| DrivingRecReview | BYTE | Boolean |
| DrivingRecReviewExp | DATE | |
| CopyOfCDL | BYTE | Boolean |
| CDLExp | DATE | |
| CopyOfMedCert | BYTE | Boolean |
| MedCertExp | DATE | |
| DOTApplication | BYTE | Boolean |
| PriorEmpChk | BYTE | Boolean |
| LastServiceHrs | BYTE | Boolean |
| PreEmpDrugTest | BYTE | Boolean |
| PrevEmpInquiries | BYTE | Boolean |
| ReceiptDrugPolicy | BYTE | Boolean |
| W4EmpWithholding | BYTE | Boolean |
| USLegalInfo | BYTE | Boolean |
| Active | BYTE | Boolean |
| SSNumber | CSTRING(12) | |
| DOTApplicationExp | DATE | |
| QBAccountRef | CSTRING(61) | |
| QBClass | CSTRING(41) | |
| CreateAP | BYTE | Boolean |
| RateCalcType | CSTRING(11) | |
| AddRate | DECIMAL(5,2) | |
| AddRateCalcType | CSTRING(11) | |
| IsDriver | BYTE | Boolean |
| IsSales | BYTE | Boolean |
| SalesRate1 | DECIMAL(5,2) | |
| SalesRate1Type | CSTRING(11) | |
| SalesRate1Duration | LONG | |
| SalesRate2 | DECIMAL(5,2) | |
| SalesRate2Type | CSTRING(11) | |
| SalesRate2Duration | LONG | |
| EmpIdNumber | CSTRING(21) | |
| UserName | CSTRING(21) | |
| Password | CSTRING(21) | |
| LastLoginDate | DATE | (via GROUP overlay) |
| LastLoginTime | TIME | (via GROUP overlay) |
| LastLoginUTCMinuteOffset | LONG | |
| BirthDate | DATE | |
| DriversLicenseState | STRING(2) | |

---

## G20 - Truck Master (-> `trucks`)

**Keys:**
- K_G20Id: PRIMARY (G20Id)
- K_Driver1: DUP (Driver1)
- K_TruckNumber: UNIQUE (TruckNumber)
- K_ActiveTruckNumber: DUP (Active, TruckNumber)
- K_LeasedTruck: DUP (LeasedTruck, TruckNumber)
- K_TrailerNumber: DUP (TrailerNumber)

| Column | Type | Notes |
|--------|------|-------|
| G20Id | LONG | PK |
| TruckNumber | CSTRING(11) | UPPER, unique |
| TruckMake | CSTRING(21) | |
| Manufacturer | CSTRING(16) | |
| TruckModel | CSTRING(16) | |
| Model | CSTRING(11) | Legacy |
| TruckSerialNumber | CSTRING(21) | |
| SerialNumber | CSTRING(21) | Legacy |
| TruckManufactureDate | DATE | |
| ManufactureDate | DATE | Legacy |
| TruckLicense | CSTRING(11) | |
| TruckLicenseExp | DATE | |
| TrailerLicense | CSTRING(11) | |
| TrailerLicenseExp | DATE | |
| TruckSafetyInspection | DATE | |
| TrailerSafetyInspection | DATE | |
| TrailerNumber | CSTRING(5) | |
| TrailerMake | CSTRING(21) | |
| TrailerModel | CSTRING(16) | |
| TrailerSerialNumber | CSTRING(21) | |
| TareWeight | LONG | |
| TruckPurchasedFrom | CSTRING(31) | |
| PurchasedFrom | CSTRING(31) | Legacy |
| TruckPurchaseDate | DATE | |
| PurchaseDate | DATE | Legacy |
| TruckCost | DECIMAL(9,2) | |
| Cost | DECIMAL(9,2) | Legacy |
| TrailerPurchasedFrom | CSTRING(31) | |
| TrailerPurchaseDate | DATE | |
| TrailerCost | DECIMAL(9,2) | |
| LicenceNumber | CSTRING(11) | |
| FinancedBy | CSTRING(31) | |
| NoteAmount | DECIMAL(9,2) | |
| OwnedBy | CSTRING(31) | |
| InsuranceExpDate | DATE | |
| InsuranceCoverageAmt | DECIMAL(9,2) | |
| LoanDate | DATE | |
| LoanTerm | BYTE | |
| ContractEndDate | DATE | |
| LoanAccount | CSTRING(16) | |
| TruckRate | DECIMAL(7,2) | |
| TruckCalcType | CSTRING(11) | Percent/Per Unit |
| LeasedTruck | BYTE | Boolean |
| WePayDriver | BYTE | Boolean |
| G50Id | LONG | FK to Vendor |
| A70Id | LONG | FK to GL Account |
| Driver1 | CSTRING(31) | Default driver 1 |
| Driver2 | CSTRING(31) | Default driver 2 |
| FleetNumber | CSTRING(11) | |
| EngineModel | CSTRING(21) | |
| EngineSerialNumber | CSTRING(21) | |
| TransModel | CSTRING(21) | |
| RearEndModel | CSTRING(21) | |
| RearEndRatio | CSTRING(11) | |
| EngineWarrMiles | LONG | |
| EngingWarrYears | LONG | (typo in original) |
| TransWarrMiles | LONG | |
| TransWarrYears | LONG | |
| RearEndWarrMiles | LONG | |
| RearEndWarrYears | LONG | |
| ClimateWarrMiles | LONG | |
| ClimateWarrYears | LONG | |
| ElectricalWarrMiles | LONG | |
| ElectricalWarrYears | LONG | |
| TowingWarrMiles | LONG | |
| TowingWarrYears | LONG | |
| WarrantyNotes | CSTRING(501) | |
| SteerTireModel | CSTRING(21) | |
| SteerTireSize | CSTRING(21) | |
| DriveTireModel | CSTRING(21) | |
| DriveTireSize | CSTRING(21) | |
| TrailerTireModel | CSTRING(21) | |
| TrailerTireSize | CSTRING(21) | |
| TrailerManufactureDate | DATE | |
| TruckYear | CSTRING(5) | |
| TrailerYear | CSTRING(5) | |
| QBTransportItem | CSTRING(16) | |
| QBClassName | CSTRING(41) | |
| Active | BYTE | Boolean |
| LinkId | CSTRING(11) | |
| LinkType | CSTRING(3) | |
| QBAccountRef | CSTRING(61) | |
| QBClass | CSTRING(41) | |
| CreateAP | BYTE | Boolean |
| Straps | BYTE | Boolean |
| Class | CSTRING(2) | Truck class |
| ExcludeFuel | BYTE | Boolean |
| CargoCoverageAmt | DECIMAL(9,2) | |
| W9Date | DATE | |
| WorkersCompDate | DATE | |
| CarrierAgreementDate | DATE | |

---

## G30 - Zone Master (-> `zones`)

**Keys:**
- K_G30id: PRIMARY (G30id)
- K_Zone: UNIQUE (Zone)
- K_Region: DUP (Region, Zone)

| Column | Type | Notes |
|--------|------|-------|
| G30id | LONG | PK |
| Zone | CSTRING(21) | Unique zone ID |
| Description | CSTRING(31) | |
| Region | CSTRING(21) | |

---

## G32 - Zone Pricing (-> `zone_pricing`)

**Keys:**
- K_G32id: PRIMARY (G32id)
- K_ZoneA: UNIQUE (ZoneA, ZoneB)

| Column | Type | Notes |
|--------|------|-------|
| G32id | LONG | PK |
| ZoneA | CSTRING(21) | Origin zone |
| ZoneB | CSTRING(21) | Destination zone |
| Description | CSTRING(201) | |
| Amount | DECIMAL(9,4) | Transport rate |
| Miles | LONG | |
| TransportDays | LONG | |
| ShipTo | CSTRING(21) | |

---

## D00 - Order Master (-> `orders`)

**Keys:**
- K_D00Id: PRIMARY (D00Id)
- K_OrderNumber: UNIQUE (OrderNumber)
- K_BillG00Id: DUP (BillG00Id, OrderNumber)
- K_LoadG00Id: DUP (LoadG00Id, OrderNumber)
- K_DropG00Id: DUP (DropG00Id, OrderNumber)
- K_Active: DUP (Active, OrderNumber)
- K_BOL: DUP (BOLNumber)
- K_CreateDate: DUP (CreateDate, D00Id)
- K_Zone: DUP (Zone, OrderNumber)

| Column | Type | Notes |
|--------|------|-------|
| D00Id | LONG | PK |
| OrderNumber | CSTRING(11) | Unique order number |
| Active | BYTE | Boolean |
| Zone | CSTRING(21) | |
| DispatchCode | CSTRING(11) | |
| BOLNumber | CSTRING(21) | Bill of Lading |
| BillG00Id | LONG | FK to G00 (billing customer) |
| BillCustomerNumber | CSTRING(11) | Denormalized |
| BillCustomerName | CSTRING(31) | Denormalized |
| LoadG00Id | LONG | FK to G00 (load/pickup customer) |
| LoadCustomerNumber | CSTRING(11) | Denormalized |
| LoadCustomerName | CSTRING(31) | Denormalized |
| LoadContact | CSTRING(21) | |
| LoadPhone | CSTRING(11) | |
| DropG00Id | LONG | FK to G00 (drop/delivery customer) |
| DropCustomerNumber | CSTRING(11) | Denormalized |
| DropCustomerName | CSTRING(31) | Denormalized |
| DropContact | CSTRING(21) | |
| DropPhone | CSTRING(11) | |
| ReferenceNumber | CSTRING(21) | |
| PONumber | CSTRING(21) | Purchase Order |
| G10Name1 | CSTRING(31) | Sales rep name |
| Comments | CSTRING(501) | |
| PUInstructions | CSTRING(501) | Pickup instructions |
| DOInstructions | CSTRING(501) | Dropoff instructions |
| TransportAmt | DECIMAL(9,4) | Transport charge |
| TransportCalcType | CSTRING(11) | |
| FuelSurcharge | DECIMAL(7,4) | |
| FuelCalcType | CSTRING(11) | |
| OtherCharge | DECIMAL(7,2) | |
| Discount | DECIMAL(7,2) | |
| DiscountCalcType | CSTRING(12) | |
| TaxRate | DECIMAL(7,4) | |
| Tax | DECIMAL(7,2) | |
| TotalCharge | DECIMAL(9,2) | |
| VehicleCount | LONG | Calculated |
| LoadedCount | LONG | |
| DeliveredCount | LONG | |
| ConfirmedCount | LONG | |
| ScheduledCount | LONG | |
| InvoicedCount | LONG | |
| WaitingCount | LONG | |
| StagingCount | LONG | |
| CreateDate | DATE | |
| OriginalCreateDate | DATE | |
| EditDate | DATE | |
| EditBy | CSTRING(21) | |
| EstPickUpDate | DATE | |
| EstDeliverDate | DATE | |
| EquipmentType | CSTRING(11) | |
| TaxCode | CSTRING(21) | |
| G10Name2 | CSTRING(31) | Second sales rep |
| DimWeight | LONG | |
| BillToAddress | CSTRING(31) | |
| BillToAddress2 | CSTRING(31) | |
| BillToCity | CSTRING(26) | |
| BillToState | CSTRING(3) | |
| BillToZip | CSTRING(11) | |
| LoadAddress | CSTRING(31) | |
| LoadAddress2 | CSTRING(31) | |
| LoadCity | CSTRING(26) | |
| LoadState | CSTRING(3) | |
| LoadZip | CSTRING(11) | |
| DropAddress | CSTRING(31) | |
| DropAddress2 | CSTRING(31) | |
| DropCity | CSTRING(26) | |
| DropState | CSTRING(3) | |
| DropZip | CSTRING(11) | |

---

## D10 - Order Vehicles (-> `order_vehicles`)

**Keys:**
- K_D10Id: PRIMARY (D10Id)
- K_D00Id: DUP (D00Id, D10Id)
- K_VIN: DUP (VIN)
- K_Status: DUP (Status, D00Id, VIN)
- K_Active: DUP (Active, Status, D00Id)
- K_LoadNumber: DUP (LoadNumber)
- K_ConfirmedDate: DUP (ConfirmedDate, D10Id)
- K_DeliveredDate: DUP (DeliveredDate, D10Id)
- K_LoadedDate: DUP (LoadedDate, D10Id)

| Column | Type | Notes |
|--------|------|-------|
| D10Id | LONG | PK |
| D00Id | LONG | FK to D00 (Order) |
| Active | BYTE | Boolean |
| VIN | CSTRING(18) | UPPER |
| Year | CSTRING(5) | |
| Make | CSTRING(21) | |
| Model | CSTRING(31) | |
| Color | CSTRING(21) | |
| Weight | LONG | |
| Category | CSTRING(2) | Size category |
| BodyStyle | CSTRING(21) | |
| Status | CSTRING(11) | Waiting/Scheduled/Loaded/Delivered/Confirmed |
| D20Id | LONG | FK to D20 (Trip) |
| LoadNumber | CSTRING(11) | |
| BayNumber | CSTRING(3) | Position on truck |
| TransportAmt | DECIMAL(9,4) | |
| TransportCalcType | CSTRING(11) | |
| FuelSurcharge | DECIMAL(7,4) | |
| FuelCalcType | CSTRING(11) | |
| OtherCharge | DECIMAL(7,2) | |
| Discount | DECIMAL(7,2) | |
| DiscountCalcType | CSTRING(12) | |
| TaxRate | DECIMAL(7,4) | |
| Tax | DECIMAL(7,2) | |
| TotalCharge | DECIMAL(9,2) | |
| ScheduledDate | DATE | |
| LoadedDate | DATE | |
| DeliveredDate | DATE | |
| ConfirmedDate | DATE | |
| ConfirmedBy | CSTRING(21) | |
| InvoiceNumber | CSTRING(21) | |
| A00Id | LONG | FK to Invoice |
| Lot | CSTRING(11) | |
| DamageCode | CSTRING(7) | |
| PUDamageCode | CSTRING(7) | Pickup damage |
| DODamageCode | CSTRING(7) | Dropoff damage |
| Comments | CSTRING(501) | |
| QBItem | CSTRING(16) | |
| RateClass | CSTRING(11) | |
| DimLength | DECIMAL(5,2) | |
| DimWidth | DECIMAL(5,2) | |
| DimHeight | DECIMAL(5,2) | |
| RunDrive | BYTE | Boolean |
| Operable | BYTE | Boolean |

---

## D20 - Trip/Load Master (-> `trips`)

**Keys:**
- K_D20Id: PRIMARY (D20Id)
- K_LoadNumber: UNIQUE (LoadNumber)
- K_Driver: DUP (Driver, LoadNumber)
- K_TruckNumber: DUP (TruckNumber, LoadNumber)
- K_Active: DUP (Active, LoadNumber)
- K_G20Id: DUP (G20Id, LoadNumber)
- K_TripDate: DUP (TripDate, LoadNumber)

| Column | Type | Notes |
|--------|------|-------|
| D20Id | LONG | PK |
| LoadNumber | CSTRING(11) | Unique load number |
| Active | BYTE | Boolean |
| TruckNumber | CSTRING(11) | |
| G20Id | LONG | FK to Truck |
| TrailerNumber | CSTRING(5) | |
| Driver | CSTRING(31) | Driver 1 name |
| G10Id1 | LONG | FK to Employee (Driver 1) |
| Driver2 | CSTRING(31) | Driver 2 name |
| G10Id2 | LONG | FK to Employee (Driver 2) |
| TripDate | DATE | |
| EstDeliverDate | DATE | |
| DeliverDate | DATE | |
| ArrivalDate | DATE | |
| ReturnDate | DATE | |
| TotalMileage | LONG | |
| TotalFuelGallons | DECIMAL(7,3) | |
| FuelAdvance | DECIMAL(7,2) | |
| TripAdvance | DECIMAL(7,2) | |
| TollsAdvance | DECIMAL(7,2) | |
| DriverRate | DECIMAL(5,2) | |
| DriverCalcType | CSTRING(11) | |
| DriverAddRate | DECIMAL(5,2) | |
| DriverAddCalcType | CSTRING(11) | |
| TruckRate | DECIMAL(7,2) | |
| TruckCalcType | CSTRING(11) | |
| Comments | CSTRING(501) | |
| Status | CSTRING(11) | |
| EquipmentType | CSTRING(11) | |
| Zone | CSTRING(21) | |

---

## D30 - Load Details (-> `load_details`)

**Keys:**
- K_D30Id: PRIMARY (D30Id)
- K_D20Id: DUP (D20Id, D30Id)
- K_D10Id: DUP (D10Id)
- K_D00Id: DUP (D00Id, D20Id)

| Column | Type | Notes |
|--------|------|-------|
| D30Id | LONG | PK |
| D20Id | LONG | FK to Trip |
| D00Id | LONG | FK to Order |
| D10Id | LONG | FK to Vehicle |
| VIN | CSTRING(18) | Denormalized |
| Year | CSTRING(5) | Denormalized |
| Make | CSTRING(21) | Denormalized |
| Model | CSTRING(31) | Denormalized |
| Color | CSTRING(21) | Denormalized |
| Weight | LONG | Denormalized |
| Category | CSTRING(2) | Denormalized |
| BayNumber | CSTRING(3) | Position on truck |
| Status | CSTRING(11) | |
| LoadedDate | DATE | |
| DeliveredDate | DATE | |

---

## D13 - Other Charges (-> `order_charges`)

**Keys:**
- K_D13Id: PRIMARY (D13Id)
- K_D00Id: DUP (D00Id, D13Id)
- K_D10Id: DUP (D10Id, D13Id)
- K_D20Id: DUP (D20Id, D13Id)

| Column | Type | Notes |
|--------|------|-------|
| D13Id | LONG | PK |
| D00Id | LONG | FK to Order |
| D10Id | LONG | FK to Vehicle |
| D20Id | LONG | FK to Trip |
| Description | CSTRING(41) | |
| Amount | DECIMAL(9,2) | |
| G40Item | CSTRING(31) | Item code ref |
| Qty | LONG | |
| Rate | DECIMAL(9,4) | |
| CalcType | CSTRING(11) | |
| Taxable | BYTE | Boolean |
| Billable | BYTE | Boolean |
| APPayable | BYTE | Boolean |
| QBItem | CSTRING(16) | |

---

## D33 - Vehicle Damage (-> `vehicle_damage`)

**Keys:**
- K_D33Id: PRIMARY (D33Id)
- K_D10Id: DUP (D10Id)
- K_D00Id: DUP (D00Id, D33Id)

| Column | Type | Notes |
|--------|------|-------|
| D33Id | LONG | PK |
| D00Id | LONG | FK to Order |
| D10Id | LONG | FK to Vehicle |
| D20Id | LONG | FK to Trip |
| VIN | CSTRING(18) | Denormalized |
| DamageArea | CSTRING(3) | FK to G70 code |
| DamageType | CSTRING(3) | FK to G71 code |
| DamageSeverity | CSTRING(2) | FK to G72 code |
| Description | CSTRING(201) | |
| InspectionPoint | CSTRING(11) | Pickup/Delivery |
| InspectedBy | CSTRING(31) | |
| InspectionDate | DATE | |
| ClaimAmount | DECIMAL(9,2) | |
| ClaimStatus | CSTRING(11) | |

---

## D34 - Damage Detail (-> `damage_details`)

**Keys:**
- K_D34Id: PRIMARY (D34Id)
- K_D33Id: DUP (D33Id, D34Id)

| Column | Type | Notes |
|--------|------|-------|
| D34Id | LONG | PK |
| D33Id | LONG | FK to D33 |
| DamageArea | CSTRING(3) | |
| DamageType | CSTRING(3) | |
| DamageSeverity | CSTRING(2) | |
| Description | CSTRING(201) | |

---

## D23 - Load Fuel & Mileage (-> `trip_fuel`)

**Keys:**
- K_D23: PRIMARY (D23Id)
- K_D20Id: DUP (D20Id, State)
- K_State: DUP (State, TruckNumber)

| Column | Type | Notes |
|--------|------|-------|
| D23Id | LONG | PK |
| D20Id | LONG | FK to Trip |
| LoadedMiles | BYTE | Boolean |
| TruckNumber | CSTRING(11) | |
| State | CSTRING(3) | UPPER |
| Mileage | LONG | |
| Gallons | DECIMAL(7,3) | |

---

## D24 - Load Expenses (-> `trip_expenses`)

**Keys:**
- K_D24Id: PRIMARY (D24Id)
- K_D20Id: DUP (D20Id)

| Column | Type | Notes |
|--------|------|-------|
| D24Id | LONG | PK |
| D20Id | LONG | FK to Trip |
| Description | CSTRING(41) | |
| Amount | DECIMAL(7,2) | |
| ExpenseDate | DATE | |

---

## D26 - Trip Routing (-> `trip_routes`)

**Keys:**
- K_D26Id: PRIMARY (D26Id)
- K_D20Id: DUP (D20Id, Sequence)

| Column | Type | Notes |
|--------|------|-------|
| D26Id | LONG | PK |
| D20Id | LONG | FK to Trip |
| Sequence | LONG | Route stop order |
| G00Id | LONG | FK to Customer |
| CustomerName | CSTRING(31) | Denormalized |
| City | CSTRING(26) | |
| State | CSTRING(3) | |
| StopType | CSTRING(11) | Pickup/Delivery |
| Miles | LONG | |
| EstArrival | DATE | |

---

## D40 - Split Loads (-> `split_loads`)

**Keys:**
- K_D40Id: PRIMARY (D40Id)
- K_D00Id: DUP (D00Id, D40Id)

| Column | Type | Notes |
|--------|------|-------|
| D40Id | LONG | PK |
| D00Id | LONG | FK to Order |
| D10Id | LONG | FK to Vehicle |
| D20Id | LONG | FK to Trip (destination) |
| OrigD20Id | LONG | FK to Trip (original) |
| VIN | CSTRING(18) | |
| SplitDate | DATE | |
| Reason | CSTRING(201) | |

---

## D11 - Vehicle Notes (-> `vehicle_notes`)

**Keys:**
- K_D11Id: PRIMARY (D11Id)
- K_Date: DUP (D10Id, -NoteDate) -- descending

| Column | Type | Notes |
|--------|------|-------|
| D11Id | LONG | PK |
| D10Id | LONG | FK to Vehicle |
| NoteDate | DATE | |
| Description | CSTRING(41) | |
| Comment | CSTRING(501) | |
| CreatedBy | CSTRING(21) | |

---

## A00 - Invoice Header (-> `invoices`)

**Keys:**
- K_A00Id: PRIMARY (A00Id)
- K_InvoiceNumber: UNIQUE (InvoiceNumber)
- K_G00Id: DUP (G00Id, InvoiceNumber)
- K_InvoiceDate: DUP (InvoiceDate, InvoiceNumber)
- K_Active: DUP (Active, InvoiceNumber)

| Column | Type | Notes |
|--------|------|-------|
| A00Id | LONG | PK |
| InvoiceNumber | CSTRING(21) | Unique |
| Active | BYTE | Boolean |
| G00Id | LONG | FK to Customer |
| CustomerNumber | CSTRING(11) | Denormalized |
| CustomerName | CSTRING(31) | Denormalized |
| D00Id | LONG | FK to Order |
| OrderNumber | CSTRING(11) | Denormalized |
| InvoiceDate | DATE | |
| DueDate | DATE | |
| Terms | CSTRING(21) | |
| TaxCode | CSTRING(21) | |
| SubTotal | DECIMAL(9,2) | |
| Tax | DECIMAL(7,2) | |
| TotalAmount | DECIMAL(9,2) | |
| AmountPaid | DECIMAL(9,2) | |
| Balance | DECIMAL(9,2) | |
| Status | CSTRING(11) | Open/Paid/Void |
| QBListId | CSTRING(37) | |
| QBTxnId | CSTRING(37) | |
| Comments | CSTRING(501) | |
| BillToAddress | CSTRING(31) | |
| BillToAddress2 | CSTRING(31) | |
| BillToCity | CSTRING(26) | |
| BillToState | CSTRING(3) | |
| BillToZip | CSTRING(11) | |
| CreatedDate | DATE | |
| CreatedBy | CSTRING(21) | |

---

## A02 - Invoice Detail (-> `invoice_details`)

**Keys:**
- K_A02Id: PRIMARY (A02Id)
- K_A00Id: DUP (A00Id, A02Id)
- K_D10Id: DUP (D10Id)

| Column | Type | Notes |
|--------|------|-------|
| A02Id | LONG | PK |
| A00Id | LONG | FK to Invoice |
| D00Id | LONG | FK to Order |
| D10Id | LONG | FK to Vehicle |
| VIN | CSTRING(18) | Denormalized |
| Year | CSTRING(5) | |
| Make | CSTRING(21) | |
| Model | CSTRING(31) | |
| Description | CSTRING(41) | |
| Qty | LONG | |
| Rate | DECIMAL(9,4) | |
| Amount | DECIMAL(9,2) | |
| QBItem | CSTRING(16) | |
| Taxable | BYTE | Boolean |
| G40Item | CSTRING(31) | |

---

## A10 - Credit Memo (-> `credit_memos`)

**Keys:**
- K_A10Id: PRIMARY (A10Id)
- K_CreditNumber: UNIQUE (CreditNumber)
- K_G00Id: DUP (G00Id)

| Column | Type | Notes |
|--------|------|-------|
| A10Id | LONG | PK |
| CreditNumber | CSTRING(21) | |
| G00Id | LONG | FK to Customer |
| CustomerNumber | CSTRING(11) | |
| CustomerName | CSTRING(31) | |
| A00Id | LONG | FK to Invoice |
| InvoiceNumber | CSTRING(21) | |
| CreditDate | DATE | |
| Amount | DECIMAL(9,2) | |
| Reason | CSTRING(201) | |
| Status | CSTRING(11) | |
| CreatedBy | CSTRING(21) | |
| Comments | CSTRING(501) | |

---

## A20 - Payment Header (-> `payments`)

**Keys:**
- K_A20Id: PRIMARY (A20Id)
- K_CheckNumber: DUP (CheckNumber)
- K_G00Id: DUP (G00Id)
- K_PaymentDate: DUP (PaymentDate)

| Column | Type | Notes |
|--------|------|-------|
| A20Id | LONG | PK |
| G00Id | LONG | FK to Customer |
| CustomerNumber | CSTRING(11) | |
| CustomerName | CSTRING(31) | |
| PaymentDate | DATE | |
| CheckNumber | CSTRING(21) | |
| Amount | DECIMAL(9,2) | |
| AppliedAmount | DECIMAL(9,2) | |
| UnappliedAmount | DECIMAL(9,2) | |
| PaymentMethod | CSTRING(11) | |
| Comments | CSTRING(501) | |
| CreatedBy | CSTRING(21) | |

---

## A30 - Payment Detail (-> `payment_details`)

**Keys:**
- K_A30Id: PRIMARY (A30Id)
- K_A20Id: DUP (A20Id, A30Id)
- K_A00Id: DUP (A00Id)

| Column | Type | Notes |
|--------|------|-------|
| A30Id | LONG | PK |
| A20Id | LONG | FK to Payment |
| A00Id | LONG | FK to Invoice |
| InvoiceNumber | CSTRING(21) | |
| Amount | DECIMAL(9,2) | Amount applied to this invoice |
| DiscountAmount | DECIMAL(7,2) | |

---

## A40 - Damage Claims (-> `damage_claims`)

**Keys:**
- K_A40Id: PRIMARY (A40Id)
- K_ClaimNumber: UNIQUE (ClaimNumber)
- K_D00Id: DUP (D00Id)

| Column | Type | Notes |
|--------|------|-------|
| A40Id | LONG | PK |
| ClaimNumber | CSTRING(21) | |
| D00Id | LONG | FK to Order |
| D10Id | LONG | FK to Vehicle |
| D20Id | LONG | FK to Trip |
| VIN | CSTRING(18) | |
| ClaimDate | DATE | |
| ClaimAmount | DECIMAL(9,2) | |
| PaidAmount | DECIMAL(9,2) | |
| Status | CSTRING(11) | |
| Description | CSTRING(501) | |
| InsuranceClaim | BYTE | Boolean |
| InsuranceClaimNumber | CSTRING(21) | |
| Resolution | CSTRING(501) | |
| ResolvedDate | DATE | |

---

## A50 - Accounts Payable (-> `accounts_payable`)

**Keys:**
- K_A50Id: PRIMARY (A50Id)
- K_D20Id: DUP (D20Id)
- K_G10Id: DUP (G10Id)
- K_G20Id: DUP (G20Id)

| Column | Type | Notes |
|--------|------|-------|
| A50Id | LONG | PK |
| D20Id | LONG | FK to Trip |
| G10Id | LONG | FK to Employee |
| G20Id | LONG | FK to Truck |
| VendorName | CSTRING(31) | |
| PayableDate | DATE | |
| Amount | DECIMAL(9,2) | |
| PaidAmount | DECIMAL(9,2) | |
| Status | CSTRING(11) | |
| Description | CSTRING(201) | |
| CheckNumber | CSTRING(21) | |
| CheckDate | DATE | |
| Comments | CSTRING(501) | |

---

## Lookup Tables

### G40 - Items/Charges (-> `items`)

| Column | Type | Notes |
|--------|------|-------|
| G40Id | LONG | PK |
| Item | CSTRING(31) | Unique item code |
| Description | CSTRING(41) | |
| DefaultAmount | DECIMAL(9,4) | |
| CalcType | CSTRING(11) | |
| QBItem | CSTRING(16) | |

### G42 - Make/Model Association (-> `vehicle_makes`)

| Column | Type | Notes |
|--------|------|-------|
| G42id | LONG | PK |
| Make | CSTRING(31) | UPPER |
| Model | CSTRING(31) | UPPER |
| Weight | LONG | Default weight |
| Category | CSTRING(2) | Size/rate category |

### G43 - VIN Definition (-> `vin_definitions`)

17 position columns (P1-P17) mapping VIN characters to Make/Model.

### G45 - Color Codes (-> `color_codes`)

| Column | Type | Notes |
|--------|------|-------|
| G45Id | LONG | PK |
| MfgCode | CSTRING(11) | Manufacturer |
| ColorCode | CSTRING(11) | |
| ColorDescription | CSTRING(21) | |

### G50 - Vendor File (-> `vendors`)

Standard address fields plus terms, tax info.

### G55 - Associated Carriers (-> `carriers`)

| Column | Type | Notes |
|--------|------|-------|
| G55Id | LONG | PK |
| LinkId | CSTRING(21) | UPPER |
| CarrierName | CSTRING(33) | Unique |
| Address/City/State/Zip | Standard | |
| Contact | CSTRING(31) | |
| Phone/Fax | CSTRING(11) | |

### G70 - Damage Areas (-> `damage_areas`)

| Column | Type | Notes |
|--------|------|-------|
| G70Id | LONG | PK |
| DamageAreaCode | CSTRING(3) | Unique |
| Description | CSTRING(61) | |

### G71 - Damage Types (-> `damage_types`)

| Column | Type | Notes |
|--------|------|-------|
| G71Id | LONG | PK |
| DamageTypeCode | CSTRING(3) | Unique |
| Description | CSTRING(61) | |

### G72 - Damage Severity (-> `damage_severities`)

| Column | Type | Notes |
|--------|------|-------|
| G72Id | LONG | PK |
| DamageSeverityCode | CSTRING(2) | Unique |
| Description | CSTRING(61) | |

### G80 - Chart of Accounts (-> `chart_of_accounts`)

| Column | Type | Notes |
|--------|------|-------|
| G80Id | LONG | PK |
| AccountType | CSTRING(16) | |
| AccountName | CSTRING(36) | |
| AccountNum | CSTRING(16) | |
| OpeningBalance | DECIMAL(9,2) | |
| OpeningDate | DATE | |

---

## Clarion Type Mapping to PostgreSQL

| Clarion Type | PostgreSQL Type | Conversion Notes |
|---|---|---|
| LONG | INTEGER | Direct |
| SHORT | SMALLINT | Direct |
| BYTE (boolean) | BOOLEAN | 0=false, 1=true |
| BYTE (numeric) | SMALLINT | When used as count |
| CSTRING(n) | VARCHAR(n-1) | Trim trailing nulls, n includes null terminator |
| STRING(n) | CHAR(n) or VARCHAR(n) | Fixed-width, trim trailing spaces |
| DECIMAL(p,s) | NUMERIC(p,s) | Direct |
| DATE | DATE | Clarion date = days since 1800-12-28. Convert: pg_date = '1800-12-28'::date + clarion_days |
| TIME | TIME | Clarion time = centiseconds since midnight. Convert: pg_time = (clarion_cs / 100) * interval '1 second' |
| MEMO(n) | TEXT | Large text fields |
| REAL | DOUBLE PRECISION | |

---

## EDI Tables (Post-MVP, documented for reference)

- E00 - COPAC Download
- E01 - COPAC Download Detail
- E10 - COPAC Upload Queue
- ICLI51 - ICL Validation Errors
- ICLI92 - ICL Invoice Audit
- VISTA610 - VISTA Transactions
- DASRA - DAS Remittance
- Various Honda/GM/Ford EDI tables

## Telematics Tables (Post-MVP)

- J00 - Job Master (DriverTech)
- J10 - Job Detail
- Z20 - Dispatch Queue
- Z30 - Customer Queue (DriverTech)

## Activity Log (Local TPS files)

- ActLog - Activity log entries
- ActLogM - Activity log memos
- These are TOPSPEED (local) files, not MSSQL

## Aliases (Views)

A02A, G80A, D20B, D30B, D10C, D13A, A40A, A00A, D40A, D00A, D20A, D23A, D10B, G00A, D30A, D10A, E00A, E01A, A30A - these are Clarion aliases used for self-joins in reports/lookups.
