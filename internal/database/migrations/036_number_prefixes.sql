-- +goose Up

-- Widen columns to accommodate prefixes
ALTER TABLE orders ALTER COLUMN order_number TYPE VARCHAR(20);
ALTER TABLE invoices ALTER COLUMN order_number TYPE VARCHAR(20);
ALTER TABLE trips ALTER COLUMN load_number TYPE VARCHAR(20);
ALTER TABLE order_vehicles ALTER COLUMN load_number TYPE VARCHAR(20);
-- Add prefixes to existing order numbers (ORD-)
UPDATE orders SET order_number = 'ORD-' || order_number
WHERE order_number NOT LIKE 'ORD-%';

UPDATE order_vehicles SET load_number = 'TRP-' || load_number
WHERE load_number IS NOT NULL AND load_number != '' AND load_number NOT LIKE 'TRP-%';

UPDATE invoices SET order_number = 'ORD-' || order_number
WHERE order_number IS NOT NULL AND order_number != '' AND order_number NOT LIKE 'ORD-%';

-- Add prefixes to existing trip/load numbers (TRP-)
UPDATE trips SET load_number = 'TRP-' || load_number
WHERE load_number NOT LIKE 'TRP-%';

-- Add prefixes to existing invoice numbers (INV-)
UPDATE invoices SET invoice_number = 'INV-' || invoice_number
WHERE invoice_number NOT LIKE 'INV-%';

UPDATE order_vehicles SET invoice_number = 'INV-' || invoice_number
WHERE invoice_number IS NOT NULL AND invoice_number != '' AND invoice_number NOT LIKE 'INV-%';

UPDATE payment_details SET invoice_number = 'INV-' || invoice_number
WHERE invoice_number IS NOT NULL AND invoice_number != '' AND invoice_number NOT LIKE 'INV-%';

UPDATE credit_memos SET invoice_number = 'INV-' || invoice_number
WHERE invoice_number IS NOT NULL AND invoice_number != '' AND invoice_number NOT LIKE 'INV-%';

-- +goose Down

-- Remove prefixes
UPDATE orders SET order_number = REPLACE(order_number, 'ORD-', '')
WHERE order_number LIKE 'ORD-%';

UPDATE order_vehicles SET load_number = REPLACE(load_number, 'TRP-', '')
WHERE load_number IS NOT NULL AND load_number LIKE 'TRP-%';

UPDATE invoices SET order_number = REPLACE(order_number, 'ORD-', '')
WHERE order_number IS NOT NULL AND order_number LIKE 'ORD-%';

UPDATE trips SET load_number = REPLACE(load_number, 'TRP-', '')
WHERE load_number LIKE 'TRP-%';

UPDATE invoices SET invoice_number = REPLACE(invoice_number, 'INV-', '')
WHERE invoice_number LIKE 'INV-%';

UPDATE order_vehicles SET invoice_number = REPLACE(invoice_number, 'INV-', '')
WHERE invoice_number IS NOT NULL AND invoice_number LIKE 'INV-%';

UPDATE payment_details SET invoice_number = REPLACE(invoice_number, 'INV-', '')
WHERE invoice_number IS NOT NULL AND invoice_number LIKE 'INV-%';

UPDATE credit_memos SET invoice_number = REPLACE(invoice_number, 'INV-', '')
WHERE invoice_number IS NOT NULL AND invoice_number LIKE 'INV-%';
