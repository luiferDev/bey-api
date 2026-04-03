# Delta for Config Module — UUIDv7 Migration

## ADDED Requirements

### Requirement: UUID Migration Feature Flag

The system SHALL support a feature flag `UUID_MIGRATION_COMPLETE` in the configuration to control migration state.

Configuration structure:
```yaml
uuid_migration:
  enabled: true
  migration_complete: false
```

#### Scenario: Feature flag controls migration behavior

- GIVEN `uuid_migration.migration_complete` is `false`
- WHEN the application starts
- THEN the system operates in migration mode
- AND both integer and UUID lookups are supported during the transition window

#### Scenario: Feature flag indicates migration complete

- GIVEN `uuid_migration.migration_complete` is `true`
- WHEN the application starts
- THEN the system operates in UUID-only mode
- AND legacy integer lookups are disabled

### Requirement: UUID Library Dependency

The system SHALL use `github.com/gofrs/uuid` (v5+) for UUID generation and parsing.

The `go.mod` file MUST include the dependency:
```
github.com/gofrs/uuid v5.x.x
```

#### Scenario: UUID library is available

- GIVEN the go.mod includes `github.com/gofrs/uuid`
- WHEN `go mod tidy` is run
- THEN the dependency is resolved successfully
- AND `uuid.NewV7()` is available for generating UUIDv7 identifiers

## MODIFIED Requirements

### Requirement: Application Startup — Migration Check

The application startup sequence SHALL check for UUID migration completion before serving requests.

The startup sequence:
1. Load configuration
2. Connect to database
3. Run GORM AutoMigrate (for non-PK changes only)
4. Check `UUID_MIGRATION_COMPLETE` flag
5. If not complete, log warning and enter migration mode
6. If complete, operate in UUID-only mode

(Previously: No migration check was needed)

#### Scenario: Application starts in migration mode

- GIVEN the database has been migrated to UUID schema
- AND `UUID_MIGRATION_COMPLETE` is `false`
- WHEN the application starts
- THEN the application logs a warning about migration mode
- AND the application serves requests with dual ID support

#### Scenario: Application starts in UUID-only mode

- GIVEN the database has been migrated to UUID schema
- AND `UUID_MIGRATION_COMPLETE` is `true`
- WHEN the application starts
- THEN the application starts normally
- AND only UUID-based lookups are supported

### Requirement: Redis Configuration for Cart (Unchanged)

The Redis configuration for cart storage remains unchanged in structure. Only the key format changes from `cart:%d` to `cart:%s`.

(Previously: Redis configuration was `cart.redis` with host, port, db, etc.)

### Requirement: JWT Secret Validation (Unchanged)

The JWT secret validation at startup remains unchanged. The secret must still be at least 32 characters.

(Previously: JWT secret validation was already in place)
