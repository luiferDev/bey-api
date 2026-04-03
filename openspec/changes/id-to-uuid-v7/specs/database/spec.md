# Delta for Database Schema — UUIDv7 Migration

## ADDED Requirements

### Requirement: UUID Extension

The system SHALL enable the `pgcrypto` or `gen_random_uuid()` function in PostgreSQL for UUID generation.

The database SHALL support native UUID column type for all primary and foreign key columns.

#### Scenario: Database supports UUID type

- GIVEN the database migration runs
- WHEN inspecting the database schema
- THEN UUID columns exist on all tables
- AND the `gen_random_uuid()` function is available

### Requirement: ID Mapping Table

The system SHALL create an `id_mapping` table to track the transformation from integer IDs to UUID IDs during migration.

Table structure:
```sql
CREATE TABLE id_mapping (
    table_name TEXT NOT NULL,
    old_id BIGINT NOT NULL,
    new_id UUID NOT NULL,
    PRIMARY KEY (table_name, old_id)
);
```

#### Scenario: ID mapping table created during migration

- GIVEN the migration script runs
- WHEN the migration completes
- THEN the `id_mapping` table exists
- AND contains mappings for all existing records across all 14 tables

#### Scenario: ID mapping populated for products table

- GIVEN the products table has 100 existing records with integer IDs
- WHEN the migration script runs
- THEN the `id_mapping` table contains 100 entries for `table_name = 'products'`
- AND each entry maps an old integer ID to a new UUID

### Requirement: UUIDv7 Default for New Records

All tables SHALL use UUIDv7 as the default value for new primary key records.

The default can be implemented via:
- Application-level generation using `github.com/gofrs/uuid` library's `uuid.NewV7()`
- OR database-level default using `gen_random_uuid()` (PostgreSQL 13+)

#### Scenario: New product gets UUIDv7 automatically

- GIVEN the products table has UUID primary key
- WHEN a new product is inserted without specifying ID
- THEN the product receives a UUIDv7 ID
- AND the ID is monotonically sortable by time

### Requirement: Foreign Key Constraints with UUID

All foreign key constraints SHALL reference UUID columns instead of integer columns.

Foreign key constraints MUST be dropped and recreated during migration to reference the new UUID columns.

#### Scenario: Foreign key constraint on product.category_id

- GIVEN the products table has `category_id UUID`
- WHEN a foreign key constraint is created
- THEN the constraint references `categories(id)` where both are UUID columns
- AND inserting a product with non-existent category UUID fails with constraint error

## MODIFIED Requirements

### Requirement: Table Primary Key Migration

All 14 tables SHALL migrate their primary key columns from `integer` (serial) to `uuid` type.

Tables affected:
1. `products` — `id INTEGER` → `id UUID`
2. `categories` — `id INTEGER` → `id UUID`
3. `product_variants` — `id INTEGER` → `id UUID`
4. `product_images` — `id INTEGER` → `id UUID`
5. `product_variant_attributes` — `id INTEGER` → `id UUID`
6. `users` — `id INTEGER` → `id UUID`
7. `addresses` — `id INTEGER` → `id UUID`
8. `orders` — `id INTEGER` → `id UUID`
9. `order_items` — `id INTEGER` → `id UUID`
10. `payments` — `id INTEGER` → `id UUID`
11. `webhook_logs` — `id INTEGER` → `id UUID`
12. `stock_movements` — `id INTEGER` → `id UUID`
13. Any auth-related tables — `id INTEGER` → `id UUID`
14. Any admin-related tables — `id INTEGER` → `id UUID`

(Previously: All tables used `SERIAL` or `BIGSERIAL` integer primary keys)

#### Scenario: Products table migrated to UUID

- GIVEN the products table has `id SERIAL PRIMARY KEY`
- WHEN the migration script runs
- THEN the products table has `id UUID PRIMARY KEY`
- AND all existing products have valid UUID IDs
- AND no data is lost

#### Scenario: Categories table migrated to UUID

- GIVEN the categories table has `id SERIAL PRIMARY KEY` and materialized path `/1/5/12/`
- WHEN the migration script runs
- THEN the categories table has `id UUID PRIMARY KEY`
- AND the materialized path is updated to use UUID short codes (first 8 hex chars)
- AND the category hierarchy is preserved

### Requirement: Category Materialized Path Migration

The category `path` column SHALL be updated from integer-based paths (`/1/5/12/`) to UUID-based paths using the first 8 hexadecimal characters of each UUID.

New path format: `/a1b2c3d4/e5f67890/f1234567/`

(Previously: Path was `/1/5/12/` using integer IDs)

#### Scenario: Category path updated after migration

- GIVEN a category with old path `/1/5/12/` where IDs 1, 5, 12 map to UUIDs starting with `a1b2c3d4`, `e5f67890`, `f1234567`
- WHEN the migration script runs
- THEN the category's path is updated to `/a1b2c3d4/e5f67890/f1234567/`
- AND the `depth` column is preserved

#### Scenario: Category tree queries work with UUID paths

- GIVEN categories with UUID-based paths
- WHEN a query searches for descendants using `path LIKE '/a1b2c3d4/%'`
- THEN all descendant categories are returned correctly
- AND the query performance is comparable to integer-based paths

### Requirement: Index Migration

All indexes on integer ID columns SHALL be rebuilt for UUID columns.

New indexes SHALL be created for:
- Primary key indexes (UUID type)
- Foreign key indexes (UUID type)
- Composite indexes involving ID columns

(Previously: Indexes were on integer columns)

#### Scenario: Primary key index on UUID

- GIVEN the products table is migrated to UUID primary key
- WHEN the migration completes
- THEN a primary key index exists on `products(id UUID)`
- AND the index is used for lookups

#### Scenario: Foreign key index on UUID

- GIVEN the product_variants table has `product_id UUID`
- WHEN the migration completes
- THEN an index exists on `product_variants(product_id)`
- AND the index is used for join queries

### Requirement: GORM AutoMigrate Compatibility

GORM `AutoMigrate` SHALL NOT be used to change column types from integer to UUID.

The migration MUST be performed via manual SQL scripts BEFORE the application code is deployed.

GORM `AutoMigrate` can only handle the new UUID schema after migration is complete.

#### Scenario: AutoMigrate works after manual migration

- GIVEN the manual SQL migration has been executed
- WHEN the application starts and calls `AutoMigrate`
- THEN `AutoMigrate` detects no schema changes needed
- AND the application starts successfully

## REMOVED Requirements

### Requirement: Integer Auto-Increment Primary Keys

(Reason: Replaced by UUIDv7 for security, distribution, and offline generation benefits)

All `SERIAL` and `BIGSERIAL` column definitions are removed from the schema.

### Requirement: Integer-Based Materialized Path

(Reason: UUID-based paths using short codes provide equivalent functionality with UUID primary keys)

The integer-based materialized path format (`/1/5/12/`) is removed. All paths use UUID short codes.
