-- Drop triggers
DROP TRIGGER IF EXISTS update_vehicles_updated_at ON vehicles;
DROP TRIGGER IF EXISTS update_shipments_updated_at ON shipments;
DROP TRIGGER IF EXISTS update_carriers_updated_at ON carriers;
DROP TRIGGER IF EXISTS update_customers_updated_at ON customers;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_vehicles_shipment_id;
DROP INDEX IF EXISTS idx_shipments_carrier_id;
DROP INDEX IF EXISTS idx_shipments_customer_id;
DROP INDEX IF EXISTS idx_shipments_org_created;
DROP INDEX IF EXISTS idx_shipments_org_status;
DROP INDEX IF EXISTS idx_shipments_organization_id;
DROP INDEX IF EXISTS idx_carriers_org_active;
DROP INDEX IF EXISTS idx_carriers_organization_id;
DROP INDEX IF EXISTS idx_customers_org_email;
DROP INDEX IF EXISTS idx_customers_organization_id;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_organization_id;

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS vehicles;
DROP TABLE IF EXISTS shipments;
DROP TABLE IF EXISTS carriers;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

-- Drop enums
DROP TYPE IF EXISTS vehicle_condition;
DROP TYPE IF EXISTS shipment_status;
DROP TYPE IF EXISTS user_role;

-- Note: We don't drop the uuid-ossp extension as it might be used by other databases
