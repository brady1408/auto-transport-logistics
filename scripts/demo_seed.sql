-- ============================================================
-- ATLinks Demo Data Seed
-- Targets company_id = 2 (AutoStrap Transport LLC)
-- ============================================================

BEGIN;

-- ============================================================
-- Additional Customers (company_id=2, existing: C1001-C1012)
-- ============================================================
INSERT INTO customers (number, name, address, city, state, zip, phone, contact, zone, type, company_id) VALUES
('C1013', 'Carmax Louisville',      '4708 Outer Loop',        'Louisville',     'KY', '40219', '5025551234', 'Matt Davis',    'IND', 'Dealer', 2),
('C1014', 'JM Family Enterprises',  '100 Jim Moran Blvd',     'Deerfield Beach','FL', '33442', '9545551010', 'Sarah Hill',    'JAX', 'OEM',    2),
('C1015', 'Subaru of Indiana',      '5500 State Rd 38 E',     'Lafayette',      'IN', '47905', '7655559876', 'Tom Richards',  'IND', 'OEM',    2),
('C1016', 'Group 1 Automotive',     '800 Gessner Rd',         'Houston',        'TX', '77024', '7135558888', 'Amy Foster',    'DAL', 'Dealer', 2),
('C1017', 'Sonic Automotive',       '6415 Idlewild Rd',       'Charlotte',      'NC', '28212', '7045557777', 'Brian Wood',    'CHA', 'Dealer', 2),
('C1018', 'Lithia Motors',          '150 N Bartlett St',      'Medford',        'OR', '97501', '5415556666', 'Diane Torres',  'DAL', 'Dealer', 2),
('C1019', 'Copart Salvage',         '14185 Dallas Pkwy',      'Dallas',         'TX', '75254', '9725554444', 'Kevin Brown',   'DAL', 'Auction',2),
('C1020', 'Manheim Nashville',      '8400 Eastgate Blvd',     'Mt. Juliet',     'TN', '37122', '6155553333', 'Jeff Morgan',   'NAS', 'Auction',2),
('C1021', 'Honda East Liberty',     '11000 Honda Pkwy',       'East Liberty',   'OH', '43319', '9375552222', 'Yuki Tanaka',   'COL', 'OEM',    2),
('C1022', 'Nissan Smyrna',          '983 Nissan Dr',          'Smyrna',         'TN', '37167', '6155551111', 'Kenji Sato',    'NAS', 'OEM',    2),
('C1023', 'Ford Chicago Assembly',  '12600 S Torrence Ave',   'Chicago',        'IL', '60633', '7735550001', 'Robert Clark',  'CHI', 'OEM',    2),
('C1024', 'GM Lansing Delta',       '920 Townsend St',        'Lansing',        'MI', '48906', '5175550002', 'Patricia Reed', 'DET', 'OEM',    2);

-- Helper: function to look up customer ID by number
-- We'll use subqueries: (SELECT id FROM customers WHERE number='C1021' AND company_id=2)

-- ============================================================
-- Additional Employees (company_id=2, existing: 1-8)
-- ============================================================
INSERT INTO employees (name, address, city, state, zip, phone, is_driver, rate, employment_date, drivers_license_number, company_id) VALUES
('Bobby Crawford',  '2310 Meridian St',  'Indianapolis', 'IN', '46225', '3175559001', true,  0.55, '2024-03-15', 'IN-887654321', 2),
('Luis Hernandez',  '445 Washington St', 'Columbus',     'IN', '47201', '8125559002', true,  0.52, '2024-06-01', 'IN-776543210', 2),
('Ray Thompson',    '890 Main St',       'Greenfield',   'IN', '46140', '3175559003', true,  0.58, '2023-11-10', 'IN-665432109', 2),
('Nick Petrov',     '120 Elm Ave',       'Plainfield',   'IN', '46168', '3175559004', true,  0.50, '2025-01-20', 'IN-554321098', 2);

-- ============================================================
-- Additional Trucks (company_id=2, existing: T101-T106)
-- ============================================================
INSERT INTO trucks (truck_number, truck_make, truck_model, truck_year, truck_license, trailer_number, trailer_make, trailer_year, company_id) VALUES
('T107', 'Peterbilt',    '389',      '2023', 'IN-TRK107', 'TR07', 'Cottrell',  '2022', 2),
('T108', 'Kenworth',     'T680',     '2024', 'IN-TRK108', 'TR08', 'Boydstun',  '2023', 2),
('T109', 'International','LT Series','2022', 'IN-TRK109', 'TR09', 'Cottrell',  '2021', 2),
('T110', 'Freightliner', 'Cascadia', '2025', 'IN-TRK110', 'TR10', 'Miller',    '2024', 2);

-- ============================================================
-- ORDERS (30 new orders spread across the last ~8 weeks)
-- All dates are relative to CURRENT_DATE so dashboard charts stay fresh
-- Using subqueries for customer IDs to avoid hardcoded values
-- ============================================================

-- Macro: cid(num) = (SELECT id FROM customers WHERE number=num AND company_id=2)

-- Fully Delivered & Confirmed orders (completed lifecycle)
INSERT INTO orders (order_number, active, origin_zone, dispatch_code,
  bill_customer_id, bill_customer_name, bill_to_city, bill_to_state,
  load_customer_id, load_customer_name, load_city, load_state,
  drop_customer_id, drop_customer_name, drop_city, drop_state,
  create_date, vehicle_count, delivered_count, confirmed_count, loaded_count, invoiced_count, waiting_count, company_id)
VALUES
('ORD-1011', true, 'IND', 'STD',
  (SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'Honda Mfg Indiana', 'Greensburg','IN',
  (SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'Honda Mfg Indiana', 'Greensburg','IN',
  (SELECT id FROM customers WHERE number='C1003' AND company_id=2), 'Tom Wood Honda', 'Indianapolis','IN',
  (CURRENT_DATE - 51), 3, 3, 3, 3, 3, 0, 2),
('ORD-1012', true, 'CHI', 'STD',
  (SELECT id FROM customers WHERE number='C1002' AND company_id=2), 'Toyota Motor Mfg', 'Princeton','IN',
  (SELECT id FROM customers WHERE number='C1002' AND company_id=2), 'Toyota Motor Mfg', 'Princeton','IN',
  (SELECT id FROM customers WHERE number='C1005' AND company_id=2), 'AutoNation Chicago', 'Chicago','IL',
  (CURRENT_DATE - 50), 4, 4, 4, 4, 4, 0, 2),
('ORD-1013', true, 'DET', 'STD',
  (SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'GM Lansing Delta', 'Lansing','MI',
  (SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'GM Lansing Delta', 'Lansing','MI',
  (SELECT id FROM customers WHERE number='C1004' AND company_id=2), 'Penske Automotive', 'Detroit','MI',
  (CURRENT_DATE - 48), 2, 2, 2, 2, 2, 0, 2),
('ORD-1014', true, 'NAS', 'STD',
  (SELECT id FROM customers WHERE number='C1022' AND company_id=2), 'Nissan Smyrna', 'Smyrna','TN',
  (SELECT id FROM customers WHERE number='C1022' AND company_id=2), 'Nissan Smyrna', 'Smyrna','TN',
  (SELECT id FROM customers WHERE number='C1010' AND company_id=2), 'Carvana Nashville', 'Nashville','TN',
  (CURRENT_DATE - 46), 5, 5, 5, 5, 5, 0, 2),
('ORD-1015', true, 'COL', 'EXP',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'Honda East Liberty', 'East Liberty','OH',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'Honda East Liberty', 'East Liberty','OH',
  (SELECT id FROM customers WHERE number='C1008' AND company_id=2), 'Germain Motor Co', 'Columbus','OH',
  (CURRENT_DATE - 44), 2, 2, 2, 2, 2, 0, 2),
('ORD-1016', true, 'ATL', 'STD',
  (SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'Hendrick Automotive', 'Charlotte','NC',
  (SELECT id FROM customers WHERE number='C1007' AND company_id=2), 'SE Auto Auction', 'Atlanta','GA',
  (SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'Hendrick Automotive', 'Charlotte','NC',
  (CURRENT_DATE - 42), 3, 3, 3, 3, 0, 0, 2),
('ORD-1017', true, 'JAX', 'STD',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'JM Family Enterprises', 'Deerfield Beach','FL',
  (SELECT id FROM customers WHERE number='C1012' AND company_id=2), 'Brumos Motor Cars', 'Jacksonville','FL',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'JM Family Enterprises', 'Deerfield Beach','FL',
  (CURRENT_DATE - 41), 4, 4, 4, 4, 0, 0, 2),
('ORD-1018', true, 'DAL', 'HOT',
  (SELECT id FROM customers WHERE number='C1011' AND company_id=2), 'Park Place Dealers', 'Dallas','TX',
  (SELECT id FROM customers WHERE number='C1019' AND company_id=2), 'Copart Salvage', 'Dallas','TX',
  (SELECT id FROM customers WHERE number='C1011' AND company_id=2), 'Park Place Dealers', 'Dallas','TX',
  (CURRENT_DATE - 38), 2, 2, 2, 2, 0, 0, 2);

-- Delivered but not yet invoiced
INSERT INTO orders (order_number, active, origin_zone, dispatch_code,
  bill_customer_id, bill_customer_name, bill_to_city, bill_to_state,
  load_customer_id, load_customer_name, load_city, load_state,
  drop_customer_id, drop_customer_name, drop_city, drop_state,
  create_date, vehicle_count, delivered_count, confirmed_count, loaded_count, invoiced_count, waiting_count, company_id)
VALUES
('ORD-1019', true, 'STL', 'STD',
  (SELECT id FROM customers WHERE number='C1009' AND company_id=2), 'Lou Fusz Auto', 'St. Louis','MO',
  (SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'Ford Chicago Assembly', 'Chicago','IL',
  (SELECT id FROM customers WHERE number='C1009' AND company_id=2), 'Lou Fusz Auto', 'St. Louis','MO',
  (CURRENT_DATE - 34), 3, 3, 0, 3, 0, 0, 2),
('ORD-1020', true, 'CHA', 'STD',
  (SELECT id FROM customers WHERE number='C1017' AND company_id=2), 'Sonic Automotive', 'Charlotte','NC',
  (SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'Hendrick Automotive', 'Charlotte','NC',
  (SELECT id FROM customers WHERE number='C1017' AND company_id=2), 'Sonic Automotive', 'Charlotte','NC',
  (CURRENT_DATE - 31), 2, 2, 0, 2, 0, 0, 2),
('ORD-1021', true, 'IND', 'DLR',
  (SELECT id FROM customers WHERE number='C1015' AND company_id=2), 'Subaru of Indiana', 'Lafayette','IN',
  (SELECT id FROM customers WHERE number='C1015' AND company_id=2), 'Subaru of Indiana', 'Lafayette','IN',
  (SELECT id FROM customers WHERE number='C1003' AND company_id=2), 'Tom Wood Honda', 'Indianapolis','IN',
  (CURRENT_DATE - 28), 3, 3, 0, 3, 0, 0, 2);

-- In Transit (loaded, not delivered)
INSERT INTO orders (order_number, active, origin_zone, dispatch_code,
  bill_customer_id, bill_customer_name, bill_to_city, bill_to_state,
  load_customer_id, load_customer_name, load_city, load_state,
  drop_customer_id, drop_customer_name, drop_city, drop_state,
  create_date, vehicle_count, delivered_count, loaded_count, waiting_count, company_id)
VALUES
('ORD-1022', true, 'NAS', 'STD',
  (SELECT id FROM customers WHERE number='C1020' AND company_id=2), 'Manheim Nashville', 'Mt. Juliet','TN',
  (SELECT id FROM customers WHERE number='C1022' AND company_id=2), 'Nissan Smyrna', 'Smyrna','TN',
  (SELECT id FROM customers WHERE number='C1020' AND company_id=2), 'Manheim Nashville', 'Mt. Juliet','TN',
  (CURRENT_DATE - 24), 4, 0, 4, 0, 2),
('ORD-1023', true, 'DAL', 'EXP',
  (SELECT id FROM customers WHERE number='C1016' AND company_id=2), 'Group 1 Automotive', 'Houston','TX',
  (SELECT id FROM customers WHERE number='C1019' AND company_id=2), 'Copart Salvage', 'Dallas','TX',
  (SELECT id FROM customers WHERE number='C1016' AND company_id=2), 'Group 1 Automotive', 'Houston','TX',
  (CURRENT_DATE - 22), 3, 0, 3, 0, 2),
('ORD-1024', true, 'CHI', 'STD',
  (SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'Ford Chicago Assembly', 'Chicago','IL',
  (SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'Ford Chicago Assembly', 'Chicago','IL',
  (SELECT id FROM customers WHERE number='C1005' AND company_id=2), 'AutoNation Chicago', 'Chicago','IL',
  (CURRENT_DATE - 20), 5, 0, 5, 0, 2),
('ORD-1025', true, 'DET', 'STD',
  (SELECT id FROM customers WHERE number='C1004' AND company_id=2), 'Penske Automotive', 'Detroit','MI',
  (SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'GM Lansing Delta', 'Lansing','MI',
  (SELECT id FROM customers WHERE number='C1004' AND company_id=2), 'Penske Automotive', 'Detroit','MI',
  (CURRENT_DATE - 18), 3, 0, 3, 0, 2);

-- Scheduled (assigned to trip, not yet loaded)
INSERT INTO orders (order_number, active, origin_zone, dispatch_code,
  bill_customer_id, bill_customer_name, bill_to_city, bill_to_state,
  load_customer_id, load_customer_name, load_city, load_state,
  drop_customer_id, drop_customer_name, drop_city, drop_state,
  create_date, vehicle_count, scheduled_count, waiting_count, company_id)
VALUES
('ORD-1026', true, 'ATL', 'STD',
  (SELECT id FROM customers WHERE number='C1007' AND company_id=2), 'SE Auto Auction', 'Atlanta','GA',
  (SELECT id FROM customers WHERE number='C1007' AND company_id=2), 'SE Auto Auction', 'Atlanta','GA',
  (SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'Hendrick Automotive', 'Charlotte','NC',
  (CURRENT_DATE - 15), 3, 3, 0, 2),
('ORD-1027', true, 'JAX', 'HOT',
  (SELECT id FROM customers WHERE number='C1012' AND company_id=2), 'Brumos Motor Cars', 'Jacksonville','FL',
  (SELECT id FROM customers WHERE number='C1012' AND company_id=2), 'Brumos Motor Cars', 'Jacksonville','FL',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'JM Family Enterprises', 'Deerfield Beach','FL',
  (CURRENT_DATE - 13), 2, 2, 0, 2),
('ORD-1028', true, 'COL', 'STD',
  (SELECT id FROM customers WHERE number='C1008' AND company_id=2), 'Germain Motor Co', 'Columbus','OH',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'Honda East Liberty', 'East Liberty','OH',
  (SELECT id FROM customers WHERE number='C1008' AND company_id=2), 'Germain Motor Co', 'Columbus','OH',
  (CURRENT_DATE - 11), 4, 4, 0, 2);

-- Waiting (new orders, no assignment)
INSERT INTO orders (order_number, active, origin_zone, dispatch_code,
  bill_customer_id, bill_customer_name, bill_to_city, bill_to_state,
  load_customer_id, load_customer_name, load_city, load_state,
  drop_customer_id, drop_customer_name, drop_city, drop_state,
  create_date, vehicle_count, waiting_count, company_id)
VALUES
('ORD-1029', true, 'IND', 'STD',
  (SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'Honda Mfg Indiana', 'Greensburg','IN',
  (SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'Honda Mfg Indiana', 'Greensburg','IN',
  (SELECT id FROM customers WHERE number='C1013' AND company_id=2), 'Carmax Louisville', 'Louisville','KY',
  (CURRENT_DATE - 9), 3, 3, 2),
('ORD-1030', true, 'DAL', 'STD',
  (SELECT id FROM customers WHERE number='C1018' AND company_id=2), 'Lithia Motors', 'Medford','OR',
  (SELECT id FROM customers WHERE number='C1019' AND company_id=2), 'Copart Salvage', 'Dallas','TX',
  (SELECT id FROM customers WHERE number='C1018' AND company_id=2), 'Lithia Motors', 'Medford','OR',
  (CURRENT_DATE - 8), 2, 2, 2),
('ORD-1031', true, 'NAS', 'DLR',
  (SELECT id FROM customers WHERE number='C1010' AND company_id=2), 'Carvana Nashville', 'Nashville','TN',
  (SELECT id FROM customers WHERE number='C1020' AND company_id=2), 'Manheim Nashville', 'Mt. Juliet','TN',
  (SELECT id FROM customers WHERE number='C1010' AND company_id=2), 'Carvana Nashville', 'Nashville','TN',
  (CURRENT_DATE - 7), 4, 4, 2),
('ORD-1032', true, 'CHA', 'EXP',
  (SELECT id FROM customers WHERE number='C1017' AND company_id=2), 'Sonic Automotive', 'Charlotte','NC',
  (SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'Hendrick Automotive', 'Charlotte','NC',
  (SELECT id FROM customers WHERE number='C1017' AND company_id=2), 'Sonic Automotive', 'Charlotte','NC',
  (CURRENT_DATE - 6), 2, 2, 2),
('ORD-1033', true, 'CHI', 'STD',
  (SELECT id FROM customers WHERE number='C1005' AND company_id=2), 'AutoNation Chicago', 'Chicago','IL',
  (SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'Ford Chicago Assembly', 'Chicago','IL',
  (SELECT id FROM customers WHERE number='C1005' AND company_id=2), 'AutoNation Chicago', 'Chicago','IL',
  (CURRENT_DATE - 5), 5, 5, 2),
('ORD-1034', true, 'IND', 'AUC',
  (SELECT id FROM customers WHERE number='C1007' AND company_id=2), 'SE Auto Auction', 'Atlanta','GA',
  (SELECT id FROM customers WHERE number='C1007' AND company_id=2), 'SE Auto Auction', 'Atlanta','GA',
  (SELECT id FROM customers WHERE number='C1013' AND company_id=2), 'Carmax Louisville', 'Louisville','KY',
  (CURRENT_DATE - 4), 3, 3, 2),
('ORD-1035', true, 'STL', 'REL',
  (SELECT id FROM customers WHERE number='C1009' AND company_id=2), 'Lou Fusz Auto', 'St. Louis','MO',
  (SELECT id FROM customers WHERE number='C1009' AND company_id=2), 'Lou Fusz Auto', 'St. Louis','MO',
  (SELECT id FROM customers WHERE number='C1013' AND company_id=2), 'Carmax Louisville', 'Louisville','KY',
  (CURRENT_DATE - 3), 2, 2, 2),
('ORD-1036', true, 'DET', 'SHW',
  (SELECT id FROM customers WHERE number='C1004' AND company_id=2), 'Penske Automotive', 'Detroit','MI',
  (SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'GM Lansing Delta', 'Lansing','MI',
  (SELECT id FROM customers WHERE number='C1004' AND company_id=2), 'Penske Automotive', 'Detroit','MI',
  (CURRENT_DATE - 2), 3, 3, 2),
('ORD-1037', true, 'COL', 'STD',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'Honda East Liberty', 'East Liberty','OH',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'Honda East Liberty', 'East Liberty','OH',
  (SELECT id FROM customers WHERE number='C1008' AND company_id=2), 'Germain Motor Co', 'Columbus','OH',
  (CURRENT_DATE - 1), 4, 4, 2),
('ORD-1038', true, 'JAX', 'STD',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'JM Family Enterprises', 'Deerfield Beach','FL',
  (SELECT id FROM customers WHERE number='C1012' AND company_id=2), 'Brumos Motor Cars', 'Jacksonville','FL',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'JM Family Enterprises', 'Deerfield Beach','FL',
  CURRENT_DATE, 3, 3, 2);

-- ============================================================
-- TRIPS (19 new trips)
-- ============================================================

-- Completed trips
INSERT INTO trips (load_number, truck_number, truck_id, driver, driver1_id, trip_date, deliver_date, total_mileage, fuel_advance, trip_advance, tolls_advance, driver_rate, driver_calc_type, status, zone, company_id) VALUES
('LD-2007', 'T101', 1, 'Mike Johnson', 1, (CURRENT_DATE - 50), (CURRENT_DATE - 49), 120, 75.00, 0, 8.50, 0.55, 'Per Mile', 'Completed', 'IND', 2),
('LD-2008', 'T102', 2, 'Carlos Ramirez', 2, (CURRENT_DATE - 49), (CURRENT_DATE - 47), 285, 125.00, 50.00, 15.00, 0.52, 'Per Mile', 'Completed', 'CHI', 2),
('LD-2009', 'T103', 3, 'Steve Williams', 3, (CURRENT_DATE - 47), (CURRENT_DATE - 46), 240, 100.00, 0, 12.00, 0.58, 'Per Mile', 'Completed', 'DET', 2),
('LD-2010', 'T104', 4, 'Tommy Nguyen', 4, (CURRENT_DATE - 45), (CURRENT_DATE - 43), 340, 150.00, 75.00, 18.00, 0.55, 'Per Mile', 'Completed', 'NAS', 2),
('LD-2011', 'T105', 5, 'Dave Marshall', 5, (CURRENT_DATE - 43), (CURRENT_DATE - 42), 175, 80.00, 0, 10.00, 0.52, 'Per Mile', 'Completed', 'COL', 2),
('LD-2012', 'T107', (SELECT id FROM trucks WHERE truck_number='T107' AND company_id=2), 'Bobby Crawford', (SELECT id FROM employees WHERE name='Bobby Crawford' AND company_id=2), (CURRENT_DATE - 41), (CURRENT_DATE - 39), 620, 275.00, 100.00, 32.00, 0.55, 'Per Mile', 'Completed', 'ATL', 2),
('LD-2013', 'T108', (SELECT id FROM trucks WHERE truck_number='T108' AND company_id=2), 'Luis Hernandez', (SELECT id FROM employees WHERE name='Luis Hernandez' AND company_id=2), (CURRENT_DATE - 40), (CURRENT_DATE - 37), 850, 350.00, 125.00, 42.00, 0.58, 'Per Mile', 'Completed', 'JAX', 2),
('LD-2014', 'T106', 6, 'James Walker', 6, (CURRENT_DATE - 37), (CURRENT_DATE - 35), 920, 400.00, 150.00, 48.00, 0.55, 'Per Mile', 'Completed', 'DAL', 2),
('LD-2015', 'T109', (SELECT id FROM trucks WHERE truck_number='T109' AND company_id=2), 'Ray Thompson', (SELECT id FROM employees WHERE name='Ray Thompson' AND company_id=2), (CURRENT_DATE - 33), (CURRENT_DATE - 31), 310, 135.00, 50.00, 16.00, 0.55, 'Per Mile', 'Completed', 'STL', 2),
('LD-2016', 'T110', (SELECT id FROM trucks WHERE truck_number='T110' AND company_id=2), 'Nick Petrov', (SELECT id FROM employees WHERE name='Nick Petrov' AND company_id=2), (CURRENT_DATE - 30), (CURRENT_DATE - 29), 180, 85.00, 0, 9.00, 0.50, 'Per Mile', 'Completed', 'CHA', 2),
('LD-2017', 'T101', 1, 'Mike Johnson', 1, (CURRENT_DATE - 27), (CURRENT_DATE - 26), 195, 90.00, 0, 10.50, 0.55, 'Per Mile', 'Completed', 'IND', 2);

-- In Transit trips (started in the last few days, delivering now)
INSERT INTO trips (load_number, truck_number, truck_id, driver, driver1_id, trip_date, est_deliver_date, total_mileage, fuel_advance, trip_advance, driver_rate, driver_calc_type, status, zone, company_id) VALUES
('LD-2018', 'T102', 2, 'Carlos Ramirez', 2, (CURRENT_DATE - 3), (CURRENT_DATE + 1), 340, 150.00, 75.00, 0.52, 'Per Mile', 'In Transit', 'NAS', 2),
('LD-2019', 'T103', 3, 'Steve Williams', 3, (CURRENT_DATE - 2), (CURRENT_DATE + 1), 920, 400.00, 125.00, 0.58, 'Per Mile', 'In Transit', 'DAL', 2),
('LD-2020', 'T107', (SELECT id FROM trucks WHERE truck_number='T107' AND company_id=2), 'Bobby Crawford', (SELECT id FROM employees WHERE name='Bobby Crawford' AND company_id=2), (CURRENT_DATE - 1), (CURRENT_DATE + 1), 285, 125.00, 50.00, 0.55, 'Per Mile', 'In Transit', 'CHI', 2),
('LD-2021', 'T104', 4, 'Tommy Nguyen', 4, (CURRENT_DATE - 1), CURRENT_DATE, 240, 100.00, 0, 0.55, 'Per Mile', 'In Transit', 'DET', 2);

-- Scheduled trips
INSERT INTO trips (load_number, truck_number, truck_id, driver, driver1_id, trip_date, est_deliver_date, driver_rate, driver_calc_type, status, zone, company_id) VALUES
('LD-2022', 'T108', (SELECT id FROM trucks WHERE truck_number='T108' AND company_id=2), 'Luis Hernandez', (SELECT id FROM employees WHERE name='Luis Hernandez' AND company_id=2), (CURRENT_DATE + 1), (CURRENT_DATE + 3), 0.58, 'Per Mile', 'Scheduled', 'ATL', 2),
('LD-2023', 'T105', 5, 'Dave Marshall', 5, (CURRENT_DATE + 2), (CURRENT_DATE + 4), 0.52, 'Per Mile', 'Scheduled', 'JAX', 2),
('LD-2024', 'T109', (SELECT id FROM trucks WHERE truck_number='T109' AND company_id=2), 'Ray Thompson', (SELECT id FROM employees WHERE name='Ray Thompson' AND company_id=2), (CURRENT_DATE + 3), (CURRENT_DATE + 4), 0.55, 'Per Mile', 'Scheduled', 'COL', 2);

-- ============================================================
-- ORDER VEHICLES — Completed orders
-- ============================================================

-- ORD-1011: 3 Honda Civics, Confirmed
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 51),(CURRENT_DATE - 50),(CURRENT_DATE - 49),(CURRENT_DATE - 48),
  (SELECT id FROM trips WHERE load_number='LD-2007' AND company_id=2), 'LD-2007', 2
FROM orders o, (VALUES
  ('1HGCV1F34RA000101','2026','Honda','Civic','Silver',210.00),
  ('1HGCV1F34RA000102','2026','Honda','Civic','Black', 210.00),
  ('1HGCV1F34RA000103','2026','Honda','Civic','White', 210.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1011' AND o.company_id=2;

-- ORD-1012: 4 Toyotas, Confirmed
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 50),(CURRENT_DATE - 49),(CURRENT_DATE - 47),(CURRENT_DATE - 46),
  (SELECT id FROM trips WHERE load_number='LD-2008' AND company_id=2), 'LD-2008', 2
FROM orders o, (VALUES
  ('4T1BF1FK0RU100201','2026','Toyota','Camry','Red',     260.00),
  ('4T1BF1FK0RU100202','2026','Toyota','Camry','Blue',    260.00),
  ('JTDKN3DU0R0100203','2026','Toyota','Corolla','White', 260.00),
  ('JTDKN3DU0R0100204','2026','Toyota','Corolla','Gray',  260.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1012' AND o.company_id=2;

-- ORD-1013: 2 Corvettes, Confirmed
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 48),(CURRENT_DATE - 47),(CURRENT_DATE - 46),(CURRENT_DATE - 45),
  (SELECT id FROM trips WHERE load_number='LD-2009' AND company_id=2), 'LD-2009', 2
FROM orders o, (VALUES
  ('1G1YY22G065100301','2026','Chevrolet','Corvette','Torch Red', 450.00),
  ('1G1YY22G065100302','2026','Chevrolet','Corvette','Arctic Wht',450.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1013' AND o.company_id=2;

-- ORD-1014: 5 Nissans, Confirmed
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 46),(CURRENT_DATE - 45),(CURRENT_DATE - 43),(CURRENT_DATE - 42),
  (SELECT id FROM trips WHERE load_number='LD-2010' AND company_id=2), 'LD-2010', 2
FROM orders o, (VALUES
  ('3N1AB8CV0RY100401','2026','Nissan','Sentra','Pearl White', 185.00),
  ('3N1AB8CV0RY100402','2026','Nissan','Sentra','Gun Metal',   185.00),
  ('5N1AT3BB0RC100403','2026','Nissan','Rogue','Scarlet',      195.00),
  ('5N1AT3BB0RC100404','2026','Nissan','Rogue','Magnetic Blk', 195.00),
  ('JN8AZ2NE0R9100405','2026','Nissan','Murano','Brilliant Sv',210.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1014' AND o.company_id=2;

-- ORD-1015: 2 Hondas, Confirmed
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 44),(CURRENT_DATE - 43),(CURRENT_DATE - 42),(CURRENT_DATE - 41),
  (SELECT id FROM trips WHERE load_number='LD-2011' AND company_id=2), 'LD-2011', 2
FROM orders o, (VALUES
  ('2HGFE2F59RH100501','2026','Honda','Civic','Rallye Red',  225.00),
  ('5J6RW2H85RL100502','2026','Honda','CR-V','Crystal Black', 245.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1015' AND o.company_id=2;

-- ORD-1016: 3 BMWs, Confirmed (no invoice yet)
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 42),(CURRENT_DATE - 41),(CURRENT_DATE - 39),(CURRENT_DATE - 38),
  (SELECT id FROM trips WHERE load_number='LD-2012' AND company_id=2), 'LD-2012', 2
FROM orders o, (VALUES
  ('WBAPH5C55BA100601','2024','BMW','5 Series','Alpine White', 320.00),
  ('WBA8E9C50GK100602','2023','BMW','3 Series','Melbourne Red',295.00),
  ('5UXCR6C09R9100603','2025','BMW','X5','Phytonic Blue',     340.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1016' AND o.company_id=2;

-- ORD-1017: 4 Porsches, Confirmed (no invoice yet)
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 41),(CURRENT_DATE - 40),(CURRENT_DATE - 37),(CURRENT_DATE - 36),
  (SELECT id FROM trips WHERE load_number='LD-2013' AND company_id=2), 'LD-2013', 2
FROM orders o, (VALUES
  ('WP0AA2A73RS100701','2024','Porsche','911','Guards Red',     475.00),
  ('WP0AB2A78RS100702','2025','Porsche','911 Turbo','GT Silver',525.00),
  ('WP1AA2AY0RP100703','2024','Porsche','Cayenne','Moonlight Bl',395.00),
  ('WP0CA2A15RK100704','2025','Porsche','718 Cayman','Racing Ylw',440.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1017' AND o.company_id=2;

-- ORD-1018: 2 Mercedes, Confirmed (no invoice yet)
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, confirmed_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Confirmed', v.amt, v.amt, (CURRENT_DATE - 38),(CURRENT_DATE - 37),(CURRENT_DATE - 35),(CURRENT_DATE - 34),
  (SELECT id FROM trips WHERE load_number='LD-2014' AND company_id=2), 'LD-2014', 2
FROM orders o, (VALUES
  ('WDDGF4HB8DA100801','2022','Mercedes','C-Class','Obsidian Blk', 380.00),
  ('WDDGF8AB0CR100802','2021','Mercedes','E-Class','Selenite Grey', 410.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1018' AND o.company_id=2;

-- ORD-1019: 3 Fords, Delivered (not confirmed)
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Delivered', v.amt, v.amt, (CURRENT_DATE - 34),(CURRENT_DATE - 33),(CURRENT_DATE - 31),
  (SELECT id FROM trips WHERE load_number='LD-2015' AND company_id=2), 'LD-2015', 2
FROM orders o, (VALUES
  ('1FA6P8TH9R5100901','2026','Ford','Mustang','Grabber Blue',  310.00),
  ('1FTEW1E59RK100902','2026','Ford','F-150','Oxford White',    275.00),
  ('3FMCR9B69RR100903','2026','Ford','Bronco Sport','Area 51',  285.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1019' AND o.company_id=2;

-- ORD-1020: 2 Stellantis, Delivered
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Delivered', v.amt, v.amt, (CURRENT_DATE - 31),(CURRENT_DATE - 30),(CURRENT_DATE - 29),
  (SELECT id FROM trips WHERE load_number='LD-2016' AND company_id=2), 'LD-2016', 2
FROM orders o, (VALUES
  ('1C4RJFBG0RC101001','2025','Jeep','Grand Cherokee','Diamond Blk', 290.00),
  ('2C3CDXCT5RH101002','2025','Dodge','Challenger','TorRed',         310.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1020' AND o.company_id=2;

-- ORD-1021: 3 Subarus, Delivered
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, delivered_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Delivered', v.amt, v.amt, (CURRENT_DATE - 28),(CURRENT_DATE - 27),(CURRENT_DATE - 26),
  (SELECT id FROM trips WHERE load_number='LD-2017' AND company_id=2), 'LD-2017', 2
FROM orders o, (VALUES
  ('JF2SKAEC0RH101101','2026','Subaru','Forester','Ice Silver',    235.00),
  ('4S3BTAAC0R3101102','2026','Subaru','Legacy','Magnetite Gray',  220.00),
  ('4S4WMAND0R3101103','2026','Subaru','Outback','Autumn Green',   245.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1021' AND o.company_id=2;

-- ORD-1022: 4 Nissans, Loaded (in transit)
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Loaded', v.amt, v.amt, (CURRENT_DATE - 4),(CURRENT_DATE - 3),
  (SELECT id FROM trips WHERE load_number='LD-2018' AND company_id=2), 'LD-2018', 2
FROM orders o, (VALUES
  ('3N1AB8CV0RY101201','2026','Nissan','Altima','Brilliant Silver', 195.00),
  ('5N1DR3CC0RC101202','2026','Nissan','Pathfinder','Pearl White',  225.00),
  ('JN1TANT32R0101203','2026','Nissan','Frontier','Tactical Grn',  215.00),
  ('5N1AT3BB0RC101204','2026','Nissan','Rogue','Scarlet Ember',    205.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1022' AND o.company_id=2;

-- ORD-1023: 3 Chevys, Loaded
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Loaded', v.amt, v.amt, (CURRENT_DATE - 3),(CURRENT_DATE - 2),
  (SELECT id FROM trips WHERE load_number='LD-2019' AND company_id=2), 'LD-2019', 2
FROM orders o, (VALUES
  ('1G1FE6S07R0101301','2026','Chevrolet','Camaro','Rally Green',  325.00),
  ('3GNKBKRS0RS101302','2026','Chevrolet','Equinox','Summit White', 245.00),
  ('1GNSKCKD0RR101303','2026','Chevrolet','Tahoe','Black',          365.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1023' AND o.company_id=2;

-- ORD-1024: 5 Fords, Loaded
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Loaded', v.amt, v.amt, (CURRENT_DATE - 2),(CURRENT_DATE - 1),
  (SELECT id FROM trips WHERE load_number='LD-2020' AND company_id=2), 'LD-2020', 2
FROM orders o, (VALUES
  ('1FA6P8CF0R5101401','2026','Ford','Mustang','Race Red',        310.00),
  ('1FMCU9J96RU101402','2026','Ford','Escape','Star White',       245.00),
  ('1FM5K8GC0RG101403','2026','Ford','Explorer','Forged Green',   275.00),
  ('1FTEW1E59RK101404','2026','Ford','F-150','Antimatter Blue',   290.00),
  ('3FTTW8E99RR101405','2026','Ford','F-250','Iconic Silver',     315.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1024' AND o.company_id=2;

-- ORD-1025: 3 GM, Loaded
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, loaded_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Loaded', v.amt, v.amt, (CURRENT_DATE - 2),(CURRENT_DATE - 1),
  (SELECT id FROM trips WHERE load_number='LD-2021' AND company_id=2), 'LD-2021', 2
FROM orders o, (VALUES
  ('1G1YY22G065101501','2026','Chevrolet','Corvette','Rapid Blue',  465.00),
  ('1G1FE1S33R0101502','2026','Chevrolet','Camaro','Vivid Orange',  335.00),
  ('1GNSKBKD0RR101503','2026','GMC','Yukon','Onyx Black',           355.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1025' AND o.company_id=2;

-- ORD-1026: 3 BMWs, Scheduled
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Scheduled', v.amt, v.amt, (CURRENT_DATE + 1),
  (SELECT id FROM trips WHERE load_number='LD-2022' AND company_id=2), 'LD-2022', 2
FROM orders o, (VALUES
  ('WBAPH5C55BA101601','2024','BMW','5 Series','Tanzanite Blue', 335.00),
  ('WBA8E9G50HK101602','2023','BMW','3 Series','Mineral White',  305.00),
  ('5UXCR6C05R9101603','2025','BMW','X5','Black Sapphire',       355.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1026' AND o.company_id=2;

-- ORD-1027: 2 Porsches, Scheduled
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Scheduled', v.amt, v.amt, (CURRENT_DATE + 2),
  (SELECT id FROM trips WHERE load_number='LD-2023' AND company_id=2), 'LD-2023', 2
FROM orders o, (VALUES
  ('WP0AA2A70RS101701','2025','Porsche','911','Crayon',            490.00),
  ('WP1AB2AY1RP101702','2024','Porsche','Cayenne','Mahogany Met',  415.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1027' AND o.company_id=2;

-- ORD-1028: 4 Honda/Acura, Scheduled
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, scheduled_date, trip_id, load_number, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Scheduled', v.amt, v.amt, (CURRENT_DATE + 3),
  (SELECT id FROM trips WHERE load_number='LD-2024' AND company_id=2), 'LD-2024', 2
FROM orders o, (VALUES
  ('2HGFE2F59RH101801','2026','Honda','Civic','Sonic Gray',    225.00),
  ('5J6RW2H85RL101802','2026','Honda','CR-V','Radiant Red',    250.00),
  ('YH4RT4870R3101803','2026','Acura','MDX','Majestic Black',  310.00),
  ('19UDE2F30RA101804','2026','Acura','ILX','Platinum White',  265.00)
) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1028' AND o.company_id=2;

-- ============================================================
-- WAITING vehicles (no trip, no dates)
-- ============================================================
INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('1HGCV1F34RA102901','2026','Honda','Accord','Platinum Wht',225.00),('1HGCV1F34RA102902','2026','Honda','Accord','Crystal Black',225.00),('1HGCV1F34RA102903','2026','Honda','HR-V','Urban Gray',215.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1029' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('JTDKN3DU0R0103001','2025','Toyota','Corolla','Celestite',240.00),('4T1G11AK0RU103002','2025','Toyota','Camry','Midnight Blk',255.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1030' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('3N1AB8CV0RY103101','2026','Nissan','Altima','Glacier Wht',195.00),('5N1AT3BB0RC103102','2026','Nissan','Rogue','Scarlet',205.00),('JN8AZ2NE0R9103103','2026','Nissan','Murano','Deep Blue',215.00),('5N1DR3CC0RC103104','2026','Nissan','Pathfinder','Silver',225.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1031' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('1C4RJFBG0RC103201','2025','Jeep','Wrangler','Sarge Green',285.00),('2C3CDXCT5RH103202','2025','Dodge','Charger','Pitch Black',300.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1032' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('1FA6P8TH9R5103301','2026','Ford','Mustang','Vapor Blue',310.00),('1FMCU9J96RU103302','2026','Ford','Escape','Carbon Gray',245.00),('1FM5K8GC0RG103303','2026','Ford','Explorer','Star White',275.00),('1FTEW1E59RK103304','2026','Ford','F-150','Atlas Blue',290.00),('3FTTW8E99RR103305','2026','Ford','F-250','Avalanche',315.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1033' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('WBAPH5C55BA103401','2023','BMW','3 Series','Sunset Org',295.00),('5UXCR6C05R9103402','2024','BMW','X3','Phytonic Blue',315.00),('WBAJA5C50RB103403','2025','BMW','5 Series','Aventurine',335.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1034' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('1G1FE1S33R0103501','2025','Chevrolet','Malibu','Mosaic Blk',230.00),('1GNKVHKD0RJ103502','2025','Chevrolet','Blazer','Summit Wht',260.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1035' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('1G1YY22G065103601','2026','Chevrolet','Corvette','Hypersonic',465.00),('1GNSKBKD0RR103602','2026','GMC','Sierra','Downpour',345.00),('1GKS2CKDXRR103603','2026','GMC','Terrain','Summit Wht',255.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1036' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('2HGFE2F59RH103701','2026','Honda','Civic','Boost Blue',225.00),('5J6RW2H85RL103702','2026','Honda','CR-V','Still Night',250.00),('YH4RT4870R3103703','2026','Acura','MDX','Liquid Carbon',310.00),('19UDE2F30RA103704','2026','Acura','Integra','Indy Yellow',275.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1037' AND o.company_id=2;

INSERT INTO order_vehicles (order_id, vin, year, make, model, color, status, transport_amt, total_charge, company_id)
SELECT o.id, v.vin, v.yr, v.mk, v.mdl, v.clr, 'Waiting', v.amt, v.amt, 2
FROM orders o, (VALUES ('WP0AA2A73RS103801','2025','Porsche','911','Python Green',510.00),('WP1AB2AY0RP103802','2024','Porsche','Cayenne','Carrara Wht',405.00),('WP0CA2A15RK103803','2025','Porsche','718 Boxster','Shark Bl',455.00)) AS v(vin,yr,mk,mdl,clr,amt) WHERE o.order_number='ORD-1038' AND o.company_id=2;

-- ============================================================
-- LOAD DETAILS (link vehicles to trips)
-- ============================================================
INSERT INTO load_details (trip_id, order_id, vehicle_id, vin, year, make, model, color, status, loaded_date, delivered_date, company_id)
SELECT ov.trip_id, ov.order_id, ov.id, ov.vin, ov.year, ov.make, ov.model, ov.color, ov.status, ov.loaded_date, ov.delivered_date, 2
FROM order_vehicles ov
JOIN orders o ON ov.order_id = o.id
WHERE o.order_number IN ('ORD-1011','ORD-1012','ORD-1013','ORD-1014','ORD-1015','ORD-1016','ORD-1017','ORD-1018',
  'ORD-1019','ORD-1020','ORD-1021','ORD-1022','ORD-1023','ORD-1024','ORD-1025')
AND o.company_id = 2 AND ov.trip_id IS NOT NULL;

-- ============================================================
-- INVOICES
-- Valid statuses are Open/Paid/Void only. "Posted" is not a status;
-- posted-ness is posted_at/posted_by, which locks invoice lines.
-- ============================================================
INSERT INTO invoices (invoice_number, customer_id, customer_number, customer_name, order_id, order_number, invoice_date, due_date, terms,
  subtotal, tax, total_amount, amount_paid, balance, status, bill_to_city, bill_to_state, created_date, created_by, posted_at, posted_by, company_id)
VALUES
('INV-3004',
  (SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'C1001', 'Honda Mfg Indiana',
  (SELECT id FROM orders WHERE order_number='ORD-1011' AND company_id=2), 'ORD-1011',
  (CURRENT_DATE - 46), (CURRENT_DATE - 16), 'Net 30', 630.00, 0, 630.00, 630.00, 0, 'Paid', 'Greensburg', 'IN', (CURRENT_DATE - 46), 'admin', (CURRENT_DATE - 45), 'admin', 2),
('INV-3005',
  (SELECT id FROM customers WHERE number='C1002' AND company_id=2), 'C1002', 'Toyota Motor Mfg',
  (SELECT id FROM orders WHERE order_number='ORD-1012' AND company_id=2), 'ORD-1012',
  (CURRENT_DATE - 44), (CURRENT_DATE - 14), 'Net 30', 1040.00, 0, 1040.00, 1040.00, 0, 'Paid', 'Princeton', 'IN', (CURRENT_DATE - 44), 'admin', (CURRENT_DATE - 43), 'admin', 2),
('INV-3006',
  (SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'C1024', 'GM Lansing Delta',
  (SELECT id FROM orders WHERE order_number='ORD-1013' AND company_id=2), 'ORD-1013',
  (CURRENT_DATE - 42), (CURRENT_DATE - 12), 'Net 30', 900.00, 0, 900.00, 900.00, 0, 'Paid', 'Lansing', 'MI', (CURRENT_DATE - 42), 'admin', (CURRENT_DATE - 41), 'admin', 2),
('INV-3007',
  (SELECT id FROM customers WHERE number='C1022' AND company_id=2), 'C1022', 'Nissan Smyrna',
  (SELECT id FROM orders WHERE order_number='ORD-1014' AND company_id=2), 'ORD-1014',
  (CURRENT_DATE - 40), (CURRENT_DATE - 10), 'Net 30', 970.00, 0, 970.00, 500.00, 470.00, 'Open', 'Smyrna', 'TN', (CURRENT_DATE - 40), 'admin', (CURRENT_DATE - 39), 'admin', 2),
('INV-3008',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'C1021', 'Honda East Liberty',
  (SELECT id FROM orders WHERE order_number='ORD-1015' AND company_id=2), 'ORD-1015',
  (CURRENT_DATE - 38), (CURRENT_DATE - 8), 'Net 30', 470.00, 0, 470.00, 470.00, 0, 'Paid', 'East Liberty', 'OH', (CURRENT_DATE - 38), 'admin', (CURRENT_DATE - 37), 'admin', 2),
('INV-3009',
  (SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'C1006', 'Hendrick Automotive',
  (SELECT id FROM orders WHERE order_number='ORD-1016' AND company_id=2), 'ORD-1016',
  (CURRENT_DATE - 36), (CURRENT_DATE - 6), 'Net 30', 955.00, 0, 955.00, 0, 955.00, 'Open', 'Charlotte', 'NC', (CURRENT_DATE - 36), 'admin', (CURRENT_DATE - 35), 'admin', 2),
('INV-3010',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'C1014', 'JM Family Enterprises',
  (SELECT id FROM orders WHERE order_number='ORD-1017' AND company_id=2), 'ORD-1017',
  (CURRENT_DATE - 34), (CURRENT_DATE - 4), 'Net 30', 1835.00, 0, 1835.00, 0, 1835.00, 'Open', 'Deerfield Beach', 'FL', (CURRENT_DATE - 34), 'admin', NULL, NULL, 2),
('INV-3011',
  (SELECT id FROM customers WHERE number='C1011' AND company_id=2), 'C1011', 'Park Place Dealers',
  (SELECT id FROM orders WHERE order_number='ORD-1018' AND company_id=2), 'ORD-1018',
  (CURRENT_DATE - 32), (CURRENT_DATE - 2), 'Net 30', 790.00, 0, 790.00, 0, 790.00, 'Open', 'Dallas', 'TX', (CURRENT_DATE - 32), 'admin', NULL, NULL, 2);

-- Recent manual invoices (no order linkage) so the current month of the
-- revenue-per-month chart is never empty.
INSERT INTO invoices (invoice_number, customer_id, customer_number, customer_name, invoice_date, due_date, terms,
  subtotal, tax, total_amount, amount_paid, balance, status, bill_to_city, bill_to_state, created_date, created_by, posted_at, posted_by, company_id)
VALUES
('INV-3012',
  (SELECT id FROM customers WHERE number='C1019' AND company_id=2), 'C1019', 'Copart Salvage',
  (CURRENT_DATE - 6), (CURRENT_DATE + 24), 'Net 30', 1980.00, 0, 1980.00, 0, 1980.00, 'Open', 'Dallas', 'TX', (CURRENT_DATE - 6), 'admin', (CURRENT_DATE - 5), 'admin', 2),
('INV-3013',
  (SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'C1023', 'Ford Chicago Assembly',
  CURRENT_DATE, (CURRENT_DATE + 30), 'Net 30', 2410.00, 0, 2410.00, 0, 2410.00, 'Open', 'Chicago', 'IL', CURRENT_DATE, 'admin', NULL, NULL, 2);

INSERT INTO invoice_details (invoice_id, description, qty, rate, amount, taxable, company_id)
SELECT id, 'Vehicle transport services', 1, total_amount, total_amount, false, 2
FROM invoices
WHERE invoice_number IN ('INV-3012','INV-3013') AND company_id = 2;

-- ============================================================
-- HISTORICAL INVOICES (months 2-6 back)
-- Fill the 6-month revenue chart and the AR aging buckets.
-- INV-2905 + INV-2906 are posted, unpaid, due >90 days past:
-- together they put $2,340.00 in the 90+ aging bucket.
-- INV-2907 lands in the 61-90 bucket, INV-2908 in 31-60.
-- ============================================================
INSERT INTO invoices (invoice_number, customer_id, customer_number, customer_name, invoice_date, due_date, terms,
  subtotal, tax, total_amount, amount_paid, balance, status, bill_to_city, bill_to_state, created_date, created_by, posted_at, posted_by, company_id)
VALUES
('INV-2901',
  (SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'C1023', 'Ford Chicago Assembly',
  (CURRENT_DATE - 170), (CURRENT_DATE - 140), 'Net 30', 3150.00, 0, 3150.00, 3150.00, 0, 'Paid', 'Chicago', 'IL', (CURRENT_DATE - 170), 'admin', (CURRENT_DATE - 169), 'admin', 2),
('INV-2902',
  (SELECT id FROM customers WHERE number='C1015' AND company_id=2), 'C1015', 'Subaru of Indiana',
  (CURRENT_DATE - 160), (CURRENT_DATE - 130), 'Net 30', 1880.00, 0, 1880.00, 1880.00, 0, 'Paid', 'Lafayette', 'IN', (CURRENT_DATE - 160), 'admin', (CURRENT_DATE - 159), 'admin', 2),
('INV-2903',
  (SELECT id FROM customers WHERE number='C1019' AND company_id=2), 'C1019', 'Copart Salvage',
  (CURRENT_DATE - 145), (CURRENT_DATE - 115), 'Net 30', 2760.00, 0, 2760.00, 2760.00, 0, 'Paid', 'Dallas', 'TX', (CURRENT_DATE - 145), 'admin', (CURRENT_DATE - 144), 'admin', 2),
('INV-2904',
  (SELECT id FROM customers WHERE number='C1016' AND company_id=2), 'C1016', 'Group 1 Automotive',
  (CURRENT_DATE - 115), (CURRENT_DATE - 85), 'Net 30', 2050.00, 0, 2050.00, 2050.00, 0, 'Paid', 'Houston', 'TX', (CURRENT_DATE - 115), 'admin', (CURRENT_DATE - 114), 'admin', 2),
('INV-2905',
  (SELECT id FROM customers WHERE number='C1018' AND company_id=2), 'C1018', 'Lithia Motors',
  (CURRENT_DATE - 150), (CURRENT_DATE - 120), 'Net 30', 1450.00, 0, 1450.00, 0, 1450.00, 'Open', 'Medford', 'OR', (CURRENT_DATE - 150), 'admin', (CURRENT_DATE - 149), 'admin', 2),
('INV-2906',
  (SELECT id FROM customers WHERE number='C1013' AND company_id=2), 'C1013', 'Carmax Louisville',
  (CURRENT_DATE - 128), (CURRENT_DATE - 98), 'Net 30', 890.00, 0, 890.00, 0, 890.00, 'Open', 'Louisville', 'KY', (CURRENT_DATE - 128), 'admin', (CURRENT_DATE - 127), 'admin', 2),
('INV-2907',
  (SELECT id FROM customers WHERE number='C1017' AND company_id=2), 'C1017', 'Sonic Automotive',
  (CURRENT_DATE - 100), (CURRENT_DATE - 70), 'Net 30', 1240.00, 0, 1240.00, 600.00, 640.00, 'Open', 'Charlotte', 'NC', (CURRENT_DATE - 100), 'admin', (CURRENT_DATE - 99), 'admin', 2),
('INV-2908',
  (SELECT id FROM customers WHERE number='C1020' AND company_id=2), 'C1020', 'Manheim Nashville',
  (CURRENT_DATE - 75), (CURRENT_DATE - 45), 'Net 30', 980.00, 0, 980.00, 0, 980.00, 'Open', 'Mt. Juliet', 'TN', (CURRENT_DATE - 75), 'admin', (CURRENT_DATE - 74), 'admin', 2),
('INV-2909',
  (SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'C1021', 'Honda East Liberty',
  (CURRENT_DATE - 85), (CURRENT_DATE - 55), 'Net 30', 3420.00, 0, 3420.00, 3420.00, 0, 'Paid', 'East Liberty', 'OH', (CURRENT_DATE - 85), 'admin', (CURRENT_DATE - 84), 'admin', 2),
('INV-2910',
  (SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'C1024', 'GM Lansing Delta',
  (CURRENT_DATE - 60), (CURRENT_DATE - 30), 'Net 30', 2680.00, 0, 2680.00, 2680.00, 0, 'Paid', 'Lansing', 'MI', (CURRENT_DATE - 60), 'admin', (CURRENT_DATE - 59), 'admin', 2);

-- Line items for historical invoices (no vehicle rows; summary lines only)
INSERT INTO invoice_details (invoice_id, description, qty, rate, amount, taxable, company_id)
SELECT id, 'Vehicle transport services', 1, total_amount, total_amount, false, 2
FROM invoices
WHERE invoice_number IN ('INV-2901','INV-2902','INV-2903','INV-2904','INV-2905','INV-2906','INV-2907','INV-2908','INV-2909','INV-2910')
AND company_id = 2;

-- Update invoiced_count on orders
UPDATE orders SET invoiced_count = vehicle_count
WHERE order_number IN ('ORD-1011','ORD-1012','ORD-1013','ORD-1014','ORD-1015','ORD-1016','ORD-1017','ORD-1018') AND company_id = 2;

-- Link vehicles to their invoices
UPDATE order_vehicles SET
  invoice_number = inv.invoice_number,
  invoice_id = inv.id
FROM invoices inv
WHERE inv.order_id = order_vehicles.order_id
AND inv.invoice_number IN ('INV-3004','INV-3005','INV-3006','INV-3007','INV-3008','INV-3009','INV-3010','INV-3011')
AND inv.company_id = 2;

-- ============================================================
-- INVOICE DETAILS
-- ============================================================
INSERT INTO invoice_details (invoice_id, order_id, vehicle_id, vin, year, make, model, description, qty, rate, amount, taxable, company_id)
SELECT inv.id, ov.order_id, ov.id, ov.vin, ov.year, ov.make, ov.model, 'Vehicle Transport', 1, ov.transport_amt, ov.total_charge, false, 2
FROM invoices inv
JOIN order_vehicles ov ON ov.order_id = inv.order_id AND ov.company_id = 2
WHERE inv.invoice_number IN ('INV-3004','INV-3005','INV-3006','INV-3007','INV-3008','INV-3009','INV-3010','INV-3011')
AND inv.company_id = 2;

-- ============================================================
-- PAYMENTS
-- ============================================================
INSERT INTO payments (customer_id, customer_number, customer_name, payment_date, check_number, amount, applied_amount, unapplied_amount, payment_method, created_by, company_id) VALUES
((SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'C1001', 'Honda Mfg Indiana',    (CURRENT_DATE - 41), 'CHK-44201', 630.00,  630.00,  0,      'Check', 'admin', 2),
((SELECT id FROM customers WHERE number='C1002' AND company_id=2), 'C1002', 'Toyota Motor Mfg',     (CURRENT_DATE - 36), 'ACH-88301', 1040.00, 1040.00, 0,      'ACH',   'admin', 2),
((SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'C1024', 'GM Lansing Delta',     (CURRENT_DATE - 31), 'WIR-99401', 900.00,  900.00,  0,      'Wire',  'admin', 2),
((SELECT id FROM customers WHERE number='C1022' AND company_id=2), 'C1022', 'Nissan Smyrna',        (CURRENT_DATE - 24), 'CHK-55501', 500.00,  500.00,  0,      'Check', 'admin', 2),
((SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'C1021', 'Honda East Liberty',   (CURRENT_DATE - 20), 'ACH-66601', 470.00,  470.00,  0,      'ACH',   'admin', 2),
((SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'C1001', 'Honda Mfg Indiana',    (CURRENT_DATE - 15), 'CHK-44301', 1500.00, 1040.00, 460.00, 'Check', 'admin', 2),
((SELECT id FROM customers WHERE number='C1006' AND company_id=2), 'C1006', 'Hendrick Automotive',  (CURRENT_DATE - 10), 'ACH-77701', 2000.00, 0,       2000.00,'ACH',   'admin', 2);

-- ============================================================
-- PAYMENT DETAILS
-- ============================================================
INSERT INTO payment_details (payment_id, invoice_id, invoice_number, amount, company_id)
SELECT p.id, inv.id, inv.invoice_number, pd.amt, 2
FROM (VALUES
  ('CHK-44201', 'INV-3004', 630.00),
  ('ACH-88301', 'INV-3005', 1040.00),
  ('WIR-99401', 'INV-3006', 900.00),
  ('CHK-55501', 'INV-3007', 500.00),
  ('ACH-66601', 'INV-3008', 470.00),
  ('CHK-44301', 'INV-3002', 1040.00)
) AS pd(chk, inv_num, amt)
JOIN payments p ON p.check_number = pd.chk AND p.company_id = 2
JOIN invoices inv ON inv.invoice_number = pd.inv_num AND inv.company_id = 2;

-- Update INV-3002 to Paid
UPDATE invoices SET amount_paid = 1040.00, balance = 0, status = 'Paid'
WHERE invoice_number = 'INV-3002' AND company_id = 2;

-- ============================================================
-- HISTORICAL PAYMENTS
-- Back the Paid historical invoices (INV-2901..2904, 2909, 2910)
-- so payment reports agree with the invoice list in those months.
-- ============================================================
INSERT INTO payments (customer_id, customer_number, customer_name, payment_date, check_number, amount, applied_amount, unapplied_amount, payment_method, created_by, company_id) VALUES
((SELECT id FROM customers WHERE number='C1023' AND company_id=2), 'C1023', 'Ford Chicago Assembly', (CURRENT_DATE - 148), 'CHK-22101', 3150.00, 3150.00, 0, 'Check', 'admin', 2),
((SELECT id FROM customers WHERE number='C1015' AND company_id=2), 'C1015', 'Subaru of Indiana',     (CURRENT_DATE - 138), 'ACH-22201', 1880.00, 1880.00, 0, 'ACH',   'admin', 2),
((SELECT id FROM customers WHERE number='C1019' AND company_id=2), 'C1019', 'Copart Salvage',        (CURRENT_DATE - 122), 'WIR-22301', 2760.00, 2760.00, 0, 'Wire',  'admin', 2),
((SELECT id FROM customers WHERE number='C1016' AND company_id=2), 'C1016', 'Group 1 Automotive',    (CURRENT_DATE - 92),  'CHK-22401', 2050.00, 2050.00, 0, 'Check', 'admin', 2),
((SELECT id FROM customers WHERE number='C1021' AND company_id=2), 'C1021', 'Honda East Liberty',    (CURRENT_DATE - 62),  'ACH-22501', 3420.00, 3420.00, 0, 'ACH',   'admin', 2),
((SELECT id FROM customers WHERE number='C1024' AND company_id=2), 'C1024', 'GM Lansing Delta',      (CURRENT_DATE - 37),  'WIR-22601', 2680.00, 2680.00, 0, 'Wire',  'admin', 2);

INSERT INTO payment_details (payment_id, invoice_id, invoice_number, amount, company_id)
SELECT p.id, inv.id, inv.invoice_number, pd.amt, 2
FROM (VALUES
  ('CHK-22101', 'INV-2901', 3150.00),
  ('ACH-22201', 'INV-2902', 1880.00),
  ('WIR-22301', 'INV-2903', 2760.00),
  ('CHK-22401', 'INV-2904', 2050.00),
  ('ACH-22501', 'INV-2909', 3420.00),
  ('WIR-22601', 'INV-2910', 2680.00)
) AS pd(chk, inv_num, amt)
JOIN payments p ON p.check_number = pd.chk AND p.company_id = 2
JOIN invoices inv ON inv.invoice_number = pd.inv_num AND inv.company_id = 2;

-- ============================================================
-- DAMAGE CLAIMS
-- ============================================================
INSERT INTO damage_claims (claim_number, order_id, vehicle_id, trip_id, vin, claim_date, claim_amount, paid_amount, status, description, insurance_claim, company_id)
SELECT dc.claim_num,
  (SELECT id FROM orders WHERE order_number=dc.ord_num AND company_id=2),
  (SELECT ov.id FROM order_vehicles ov JOIN orders o ON ov.order_id=o.id WHERE ov.vin=dc.vin AND o.company_id=2 LIMIT 1),
  (SELECT ov.trip_id FROM order_vehicles ov JOIN orders o ON ov.order_id=o.id WHERE ov.vin=dc.vin AND o.company_id=2 LIMIT 1),
  dc.vin, dc.claim_date::date, dc.claim_amt, dc.paid_amt, dc.st, dc.descr, dc.ins, 2
FROM (VALUES
  ('DMG-001', 'ORD-1012', '4T1BF1FK0RU100201', (CURRENT_DATE - 46), 1200.00, 1200.00, 'Resolved', 'Scratched front bumper during loading', false),
  ('DMG-002', 'ORD-1014', '5N1AT3BB0RC100403', (CURRENT_DATE - 42), 2500.00, 0.00,    'Open',     'Dent on rear quarter panel at delivery', true),
  ('DMG-003', 'ORD-1017', 'WP0AA2A73RS100701', (CURRENT_DATE - 36), 4800.00, 3200.00, 'Partial',  'Paint chip on hood and cracked mirror', true),
  ('DMG-004', 'ORD-1016', 'WBA8E9C50GK100602', (CURRENT_DATE - 38), 850.00,  850.00,  'Resolved', 'Minor scuff on driver side door', false),
  ('DMG-005', 'ORD-1013', '1G1YY22G065100302', (CURRENT_DATE - 45), 6500.00, 0.00,    'Open',     'Windshield crack found at delivery', true)
) AS dc(claim_num, ord_num, vin, claim_date, claim_amt, paid_amt, st, descr, ins);

-- ============================================================
-- CREDIT MEMOS
-- ============================================================
INSERT INTO credit_memos (credit_number, customer_id, customer_number, customer_name, invoice_id, invoice_number, credit_date, amount, reason, status, created_by, company_id)
VALUES
('CM-001',
  (SELECT id FROM customers WHERE number='C1001' AND company_id=2), 'C1001', 'Honda Mfg Indiana',
  (SELECT id FROM invoices WHERE invoice_number='INV-3004' AND company_id=2), 'INV-3004',
  (CURRENT_DATE - 38), 50.00, 'Late delivery fee credit', 'Applied', 'admin', 2),
('CM-002',
  (SELECT id FROM customers WHERE number='C1022' AND company_id=2), 'C1022', 'Nissan Smyrna',
  (SELECT id FROM invoices WHERE invoice_number='INV-3007' AND company_id=2), 'INV-3007',
  (CURRENT_DATE - 20), 120.00, 'Overcharge adjustment - wrong rate', 'Applied', 'admin', 2),
('CM-003',
  (SELECT id FROM customers WHERE number='C1014' AND company_id=2), 'C1014', 'JM Family Enterprises',
  (SELECT id FROM invoices WHERE invoice_number='INV-3010' AND company_id=2), 'INV-3010',
  (CURRENT_DATE - 15), 75.00, 'Duplicate fuel surcharge billed', 'Pending', 'admin', 2);

-- ============================================================
-- ACCOUNTS PAYABLE
-- ============================================================
INSERT INTO accounts_payable (trip_id, employee_id, truck_id, vendor_name, payable_date, amount, paid_amount, status, description, company_id)
SELECT t.id, t.driver1_id, t.truck_id, ap.vendor, ap.pay_date::date, ap.amt, ap.paid, ap.st, ap.descr, 2
FROM trips t, (VALUES
  ('LD-2007', 'Driver Settlement',  (CURRENT_DATE - 46), 66.00,   66.00,  'Paid',   'Mike Johnson - 120mi @ $0.55/mi'),
  ('LD-2008', 'Driver Settlement',  (CURRENT_DATE - 44), 148.20,  148.20, 'Paid',   'Carlos Ramirez - 285mi @ $0.52/mi'),
  ('LD-2009', 'Driver Settlement',  (CURRENT_DATE - 42), 139.20,  139.20, 'Paid',   'Steve Williams - 240mi @ $0.58/mi'),
  ('LD-2010', 'Driver Settlement',  (CURRENT_DATE - 40), 187.00,  187.00, 'Paid',   'Tommy Nguyen - 340mi @ $0.55/mi'),
  ('LD-2011', 'Driver Settlement',  (CURRENT_DATE - 38), 91.00,   91.00,  'Paid',   'Dave Marshall - 175mi @ $0.52/mi'),
  ('LD-2012', 'Driver Settlement',  (CURRENT_DATE - 36), 341.00,  341.00, 'Paid',   'Bobby Crawford - 620mi @ $0.55/mi'),
  ('LD-2013', 'Driver Settlement',  (CURRENT_DATE - 34), 493.00,  0,      'Open',   'Luis Hernandez - 850mi @ $0.58/mi'),
  ('LD-2014', 'Driver Settlement',  (CURRENT_DATE - 31), 506.00,  0,      'Open',   'James Walker - 920mi @ $0.55/mi'),
  ('LD-2015', 'Driver Settlement',  (CURRENT_DATE - 28), 170.50,  0,      'Open',   'Ray Thompson - 310mi @ $0.55/mi'),
  ('LD-2016', 'Driver Settlement',  (CURRENT_DATE - 26), 90.00,   0,      'Open',   'Nick Petrov - 180mi @ $0.50/mi'),
  ('LD-2017', 'Driver Settlement',  (CURRENT_DATE - 23), 107.25,  0,      'Open',   'Mike Johnson - 195mi @ $0.55/mi'),
  ('LD-2007', 'Midwest Fuel Co',    (CURRENT_DATE - 48), 75.00,   75.00,  'Paid',   'Fuel advance reimbursement'),
  ('LD-2008', 'Midwest Fuel Co',    (CURRENT_DATE - 46), 125.00,  125.00, 'Paid',   'Fuel advance reimbursement'),
  ('LD-2012', 'Quick Tire Service', (CURRENT_DATE - 39), 450.00,  0,      'Open',   'Emergency tire replacement - T107'),
  ('LD-2014', 'TX Toll Authority',  (CURRENT_DATE - 34), 48.00,   48.00,  'Paid',   'Tolls - Dallas to Indianapolis')
) AS ap(ld_num, vendor, pay_date, amt, paid, st, descr)
WHERE t.load_number = ap.ld_num AND t.company_id = 2;

COMMIT;
