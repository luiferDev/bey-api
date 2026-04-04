# Bey API

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![Framework](https://img.shields.io/badge/Gin-00ADD8?logo=go)](https://github.com/gin-gonic/gin)
[![Database](https://img.shields.io/badge/PostgreSQL-17-336791?logo=postgresql)](https://www.postgresql.org/)
[![Cache](https://img.shields.io/badge/Redis-8-DC382D?logo=redis)](https://redis.io/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![CI](https://github.com/luiferdev/bey_api/actions/workflows/ci.yml/badge.svg)](https://github.com/luiferdev/bey_api/actions/workflows/ci.yml)

REST API de e-commerce construida con **Go**, **Gin** y **GORM**. Gestiona productos, categorías, variantes, inventario, carritos de compra, órdenes, pagos (Wompi), autenticación JWT con 2FA y OAuth2.

## 📋 Tabla de Contenidos

- [Características](#-características)
- [Tech Stack](#-tech-stack)
- [Estructura del Proyecto](#-estructura-del-proyecto)
- [Requisitos](#-requisitos)
- [Inicio Rápido](#-inicio-rápido)
- [Configuración](#-configuración)
- [Endpoints de la API](#-endpoints-de-la-api)
- [Base de Datos](#-base-de-datos)
  - [Backups](#-backups)
  - [Restauración](#-restauración)
  - [Migración entre versiones de PostgreSQL](#-migración-entre-versiones-de-postgresql)
- [Docker](#-docker)
  - [Docker Compose](#-docker-compose)
  - [Watch Mode](#-watch-mode)
- [Caché con Redis](#-caché-con-redis)
- [Testing](#-testing)
- [CI/CD](#-cicd)
- [Seguridad](#-seguridad)
- [Swagger](#-swagger)

---

## ✨ Características

- **Autenticación**: JWT con access/refresh tokens, 2FA (TOTP), OAuth2 con Google
- **Productos**: CRUD completo con categorías, variantes (SKU, atributos) e imágenes
- **Carrito**: Shopping cart respaldado por Redis con TTL configurable
- **Órdenes**: Creación, confirmación, cancelación y seguimiento de estado
- **Inventario**: Gestión de stock con reservas y liberaciones
- **Pagos**: Integración con Wompi (Colombia) — pagos y links de pago
- **Caché**: Redis para productos, categorías, variantes, imágenes y búsquedas (TTL 8h)
- **Rate Limiting**: Protección contra brute force y abuso de endpoints
- **Email**: Notificaciones por SMTP (verificación, recuperación de contraseña)
- **Health Check**: Endpoint `/health` con estado de PostgreSQL y Redis
- **Métricas**: Endpoint `/metrics/cache` con hit rate, miss rate y contadores

---

## 🛠 Tech Stack

| Componente | Tecnología |
|------------|------------|
| **Lenguaje** | Go 1.26 |
| **Framework HTTP** | Gin |
| **ORM** | GORM |
| **Base de datos** | PostgreSQL 17 |
| **Caché / Carrito** | Redis 8 |
| **Autenticación** | JWT (HS256), TOTP (2FA), OAuth2 |
| **Pagos** | Wompi (sandbox/production) |
| **Email** | SMTP (go-mail) |
| **Documentación** | Swagger / OpenAPI 3.0 |
| **CI/CD** | GitHub Actions |
| **Contenedores** | Docker + Docker Compose |

---

## 📁 Estructura del Proyecto

```
bey_api/
├── cmd/api/
│   ├── main.go              # Entry point, wiring de dependencias
│   ├── docs/                # Swagger generado (auto-generado)
│   └── static/              # Archivos estáticos (dashboard)
├── internal/
│   ├── config/              # Carga de configuración YAML
│   ├── database/            # Conexión a PostgreSQL
│   ├── concurrency/         # Worker pool, task queue, rate limiter
│   ├── modules/
│   │   ├── auth/            # JWT, OAuth2, 2FA, refresh tokens
│   │   ├── users/           # Gestión de usuarios
│   │   ├── products/        # Productos, categorías, variantes, imágenes
│   │   ├── cart/            # Carrito de compras (Redis)
│   │   ├── orders/          # Órdenes y order items
│   │   ├── payments/        # Wompi: pagos y links de pago
│   │   ├── inventory/       # Stock, reservas, liberaciones
│   │   ├── email/           # Servicio de email SMTP
│   │   └── admin/           # Operaciones administrativas
│   └── shared/
│       ├── cache/           # RedisPool, CacheService, CacheMetrics, CacheWarmer
│       ├── middleware/      # CORS, logging, rate limiting, auth, RBAC, CSRF
│       ├── response/        # Respuestas estandarizadas
│       └── health.go        # Health check endpoint
├── config.yaml              # Configuración de la aplicación
├── docker-compose.yml       # Servicios: API + PostgreSQL + Redis
├── Dockerfile               # Multi-stage build optimizado
├── go.mod / go.sum          # Dependencias
└── .github/workflows/       # CI/CD pipelines
```

---

## 📋 Requisitos

- **Go** 1.26+
- **PostgreSQL** 17+
- **Redis** 8+
- **Docker** + **Docker Compose** (opcional, recomendado)
- **buildx** plugin para Docker (para watch mode)

---

## 🚀 Inicio Rápido

### Con Docker (recomendado)

```bash
# Clonar y levantar
git clone <repo-url>
cd bey_api
docker compose up -d

# La API estará en http://localhost:8080
# Swagger en http://localhost:8080/swagger/index.html
```

### Modo desarrollo con Watch

```bash
# Watch mode: reinicia automáticamente al cambiar archivos .go
docker compose watch

# Los cambios en go.mod, go.sum o Dockerfile triggeran un rebuild automático
```

### Sin Docker

```bash
# Instalar dependencias
go mod download

# Asegurarse de que PostgreSQL y Redis estén corriendo
# Editar config.yaml con las credenciales correctas

# Ejecutar
go run ./cmd/api/

# O compilar
go build -o main ./cmd/api/
./main
```

---

## ⚙️ Configuración

Toda la configuración está en `config.yaml`. Las secciones principales:

| Sección | Descripción |
|---------|-------------|
| `app` | Host, puerto, modo, admin credentials |
| `database` | Conexión PostgreSQL (host, user, password, pool) |
| `security` | JWT secret, expiry, CSRF, CORS origins |
| `rate_limit` | Límites por endpoint (login: 5/min, default: 60/min) |
| `email` | SMTP para verificación y recuperación |
| `oauth` | Google OAuth2 credentials |
| `cart` | Redis para carrito de compras (DB 1) |
| `wompi` | Wompi payment gateway (sandbox/production) |
| `cache` | Redis para caché de lectura (DB 2, TTL 8h) |

> ⚠️ **Nunca commitees secretos reales**. Usa variables de entorno o un gestor de secretos en producción.

---

## 🔌 Endpoints de la API

Todos los endpoints protegidos requieren JWT en el header `Authorization: Bearer <token>`.

### Auth
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `POST` | `/api/v1/auth/login` | Iniciar sesión | ❌ |
| `POST` | `/api/v1/auth/refresh` | Renovar token (cookie) | ❌ |
| `POST` | `/api/v1/auth/logout` | Cerrar sesión | ❌ |
| `POST` | `/api/v1/auth/register` | Registrar usuario | ❌ |
| `POST` | `/api/v1/auth/verify-email` | Verificar email | ❌ |
| `POST` | `/api/v1/auth/forgot-password` | Recuperar contraseña | ❌ |
| `POST` | `/api/v1/auth/reset-password` | Resetear contraseña | ❌ |
| `POST` | `/api/v1/auth/2fa/setup` | Configurar 2FA | ✅ |
| `POST` | `/api/v1/auth/2fa/verify` | Verificar código 2FA | ✅ |
| `GET`  | `/api/v1/auth/google` | OAuth2 Google | ❌ |

### Productos y Categorías
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `GET`  | `/api/v1/products` | Listar productos | ❌ |
| `GET`  | `/api/v1/products/:id` | Obtener producto | ❌ |
| `GET`  | `/api/v1/products/slug/:slug` | Buscar por slug | ❌ |
| `GET`  | `/api/v1/products/:id/variants` | Variantes del producto | ❌ |
| `GET`  | `/api/v1/products/:id/images` | Imágenes del producto | ❌ |
| `GET`  | `/api/v1/variants/:id` | Obtener variante | ❌ |
| `GET`  | `/api/v1/images/:id` | Obtener imagen | ❌ |
| `GET`  | `/api/v1/categories` | Listar categorías | ❌ |
| `POST` | `/api/v1/products` | Crear producto | 🔒 Admin |
| `PUT`  | `/api/v1/products/:id` | Actualizar producto | 🔒 Admin |
| `DELETE` | `/api/v1/products/:id` | Eliminar producto | 🔒 Admin |

### Carrito
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `GET`  | `/api/v1/cart` | Obtener carrito | ✅ |
| `POST` | `/api/v1/cart/items` | Agregar item | ✅ |
| `PUT`  | `/api/v1/cart/items/:variant_id` | Actualizar cantidad | ✅ |
| `DELETE` | `/api/v1/cart/items/:variant_id` | Eliminar item | ✅ |
| `DELETE` | `/api/v1/cart` | Vaciar carrito | ✅ |

### Órdenes
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `POST` | `/api/v1/orders` | Crear orden | ✅ |
| `GET`  | `/api/v1/orders/:id` | Obtener orden | ✅ |
| `PATCH` | `/api/v1/orders/:id/status` | Actualizar estado | ✅ |
| `POST` | `/api/v1/orders/:id/confirm` | Confirmar orden | ✅ |
| `POST` | `/api/v1/orders/:id/cancel` | Cancelar orden | ✅ |
| `GET`  | `/api/v1/orders/tasks/:task_id` | Estado de tarea asíncrona | ✅ |
| `GET`  | `/api/v1/orders` | Listar órdenes | 🔒 Admin |

#### Estados de Orden

| Estado | Descripción | Transiciones válidas |
|--------|-------------|---------------------|
| `pending` | Orden creada, esperando confirmación | → `confirmed`, → `cancelled` |
| `confirmed` | Orden confirmada, stock descontado | → `shipped`, → `cancelled` |
| `shipped` | Orden enviada al cliente | → `delivered` |
| `delivered` | Orden entregada al cliente | (estado final) |
| `cancelled` | Orden cancelada, stock liberado | (estado final) |

> ⚠️ El endpoint `PATCH /api/v1/orders/:id/status` acepta **cualquier string** como estado. No hay validación de transiciones a nivel de API — la validación de estados válidos se hace en los endpoints `confirm` y `cancel`.

#### Estados de Pago

| Estado | Descripción |
|--------|-------------|
| `pending` | Pago pendiente |
| `paid` | Pago aprobado |
| `failed` | Pago rechazado |
| `refunded` | Pago reembolsado |

### Pagos
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `POST` | `/api/v1/payments` | Crear pago | ✅ |
| `GET`  | `/api/v1/payments/:id` | Obtener pago | ✅ |
| `POST` | `/api/v1/payments/links` | Crear link de pago | ✅ |
| `POST` | `/api/v1/payments/webhook` | Webhook Wompi | ❌ |

### Inventario
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `GET`  | `/api/v1/inventory/:product_id` | Ver stock | ❌ |
| `PUT`  | `/api/v1/inventory/:product_id` | Actualizar stock | 🔒 Admin |
| `POST` | `/api/v1/inventory/:product_id/reserve` | Reservar stock | 🔒 Admin |
| `POST` | `/api/v1/inventory/:product_id/release` | Liberar reserva | 🔒 Admin |

### Monitoreo
| Método | Endpoint | Descripción | Auth |
|--------|----------|-------------|------|
| `GET`  | `/health` | Health check (DB + Redis) | ❌ |
| `GET`  | `/metrics/cache` | Métricas de caché | ❌ |
| `GET`  | `/swagger/index.html` | Documentación API | ❌ |

---

## 🗄️ Base de Datos

La aplicación usa **PostgreSQL 17** con **GORM** para ORM. Las migraciones se ejecutan automáticamente al iniciar (`AutoMigrate`).

### Conexión

```yaml
database:
  host: "postgres"
  port: 5432
  user: "bey_user"
  password: "bey_password"
  name: "bey_db"
  sslmode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
```

---

### 📦 Backups

#### Desde Docker Compose

```bash
# Backup completo (estructura + datos) con DROP automático
docker exec bey_postgres pg_dump -U bey_user --clean --if-exists bey_db > backup_completo.sql

# Backup solo de datos (INSERTs)
docker exec bey_postgres pg_dump -U bey_user --data-only --disable-triggers bey_db > backup_datos.sql

# Backup solo de estructura (sin datos)
docker exec bey_postgres pg_dump -U bey_user --schema-only bey_db > backup_estructura.sql

# Backup comprimido (formato custom de PostgreSQL)
docker exec bey_postgres pg_dump -U bey_user -Fc bey_db > backup_custom.dump

# Backup de una sola tabla
docker exec bey_postgres pg_dump -U bey_user --table=products bey_db > backup_products.sql
```

#### Desde PostgreSQL nativo (sin Docker)

```bash
# Backup completo con --clean (recomendado)
pg_dump -h localhost -p 5432 -U bey_user --clean --if-exists bey_db > backup_completo.sql

# Backup con formato comprimido
pg_dump -h localhost -p 5432 -U bey_user -Fc bey_db > backup.dump

# Backup de toda la base (incluye roles y configuraciones)
pg_dumpall -h localhost -p 5432 -U postgres > backup_all.sql

# Backup paralelo (más rápido para bases grandes, requiere formato directory)
pg_dump -h localhost -p 5432 -U bey_user -Fd -j 4 -f backup_dir/ bey_db
```

#### Backup programado (cron)

```bash
# Agregar al crontab: backup diario a las 2:00 AM
0 2 * * * docker exec bey_postgres pg_dump -U bey_user --clean --if-exists bey_db > /backups/bey_db_$(date +\%Y\%m\%d).sql

# Mantener solo los últimos 7 días
find /backups/ -name "bey_db_*.sql" -mtime +7 -delete
```

---

### 🔄 Restauración

#### Método recomendado: `--clean --if-exists`

Este es el método más seguro. El backup incluye `DROP TABLE IF EXISTS` antes de cada `CREATE TABLE`, así que limpia todo antes de restaurar:

```bash
# Restaurar backup completo (con --clean)
docker exec -i bey_postgres psql -U bey_user bey_db < backup_completo.sql

# Sin Docker
psql -h localhost -p 5432 -U bey_user bey_db < backup_completo.sql
```

#### Restaurar backup de solo datos

Si el backup es `--data-only`, primero limpia las tablas existentes:

```bash
# 1. Limpiar tablas (mantiene la estructura)
docker exec -it bey_postgres psql -U bey_user -d bey_db -c "
TRUNCATE TABLE categories, inventories, order_items, orders, payment_links, payments, product_images, product_variants, product_variant_attributes, products, refresh_tokens, users RESTART IDENTITY CASCADE;
"

# 2. Restaurar datos
docker exec -i bey_postgres psql -U bey_user bey_db < backup_datos.sql
```

#### Restaurar formato custom (.dump)

```bash
# Restaurar backup en formato comprimido
docker exec -i bey_postgres pg_restore -U bey_user -d bey_db --clean --if-exists < backup.dump

# Restaurar solo una tabla del dump
docker exec -i bey_postgres pg_restore -U bey_user -d bey_db --table=products < backup.dump
```

#### Restaurar desde backup paralelo

```bash
# Restaurar backup directory (paralelo)
docker cp backup_dir/ bey_postgres:/tmp/backup_dir
docker exec bey_postgres pg_restore -U bey_user -d bey_db -j 4 /tmp/backup_dir
```

#### Restaurar en base de datos nueva (desde cero)

```bash
# 1. Borrar y recrear la base de datos
docker exec -it bey_postgres psql -U bey_user -d bey_db -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO bey_user;"

# 2. Restaurar
docker exec -i bey_postgres psql -U bey_user bey_db < backup_completo.sql
```

---

### 🔄 Migración entre versiones de PostgreSQL

Si actualizas de versión (ej: 15 → 17), los datos del volumen son incompatibles:

```bash
# 1. Hacer backup de la versión anterior
docker exec bey_postgres pg_dump -U bey_user --clean --if-exists bey_db > backup_migracion.sql

# 2. Parar servicios
docker compose down

# 3. Borrar volumen viejo
docker volume rm bey_api_postgres_data

# 4. Actualizar docker-compose.yml a la nueva versión de PostgreSQL
# image: postgres:17-alpine

# 5. Levantar con la nueva versión
docker compose up -d

# 6. Restaurar backup
docker exec -i bey_postgres psql -U bey_user bey_db < backup_migracion.sql
```

---

## 🐳 Docker

### Docker Compose

El `docker-compose.yml` incluye tres servicios:

| Servicio | Imagen | Puerto | Descripción |
|----------|--------|--------|-------------|
| `postgres` | `postgres:17-alpine` | 5432 | Base de datos |
| `redis` | `redis:8-alpine` | 6379 | Caché + carrito + rate limiting |
| `api` | Build local | 8080 | API Go |

```bash
# Levantar todos los servicios
docker compose up -d

# Ver logs
docker compose logs -f api

# Parar todo
docker compose down

# Parar y borrar volúmenes (⚠️ se pierden los datos)
docker compose down -v
```

### Watch Mode

El compose incluye **watch mode** para desarrollo. Al modificar archivos `.go`, el container se sincroniza y reinicia automáticamente:

```bash
# Iniciar watch mode
docker compose watch

# Comportamiento:
# - Archivos .go → sync + restart (rápido, ~1s)
# - go.mod / go.sum → rebuild (reconstruye la imagen)
# - Dockerfile → rebuild
# - Ignora: .git/, vendor/, .idea/, .vscode/
```

> ⚠️ Requiere `docker-buildx-plugin`. Instalar: `sudo apt-get install docker-buildx-plugin`

### Dockerfile

Multi-stage build optimizado:
1. **Builder**: `golang:1.26-alpine` — compila el binario
2. **Final**: `alpine:3.21` — binario + ca-certificates + tzdata

Binario optimizado con `-ldflags="-s -w"` (sin símbolos, sin DWARF).

---

## 🔴 Caché con Redis

La aplicación usa Redis en **3 bases de datos separadas**:

| DB | Uso |
|----|-----|
| `DB 0` | Rate limiting |
| `DB 1` | Carrito de compras |
| `DB 2` | Caché de lectura + refresh tokens |

### Entidades cacheadas

| Entidad | Key Pattern | TTL |
|---------|-------------|-----|
| Producto | `cache:product:{id}` | 8h |
| Producto por slug | `cache:product:slug:{slug}` | 8h |
| Lista de productos | `cache:product:list:{offset}:{limit}` | 8h |
| Categoría | `cache:category:{id}` | 8h |
| Lista de categorías | `cache:category:list` | 8h |
| Variante | `cache:variant:{id}` | 8h |
| Variantes por producto | `cache:variant:product:{productID}` | 8h |
| Imagen | `cache:image:{id}` | 8h |
| Imágenes por producto | `cache:image:product:{productID}` | 8h |
| Refresh token | `auth:refresh:{token_hash}` | Según expiry |

### Invalidación automática

El caché se invalida automáticamente en operaciones de escritura:

| Operación | Qué se invalida |
|-----------|-----------------|
| Crear producto | `cache:product:list:*`, `cache:product:search:*` |
| Actualizar producto | `cache:product:{id}`, listas, búsquedas |
| Eliminar producto | `cache:product:{id}`, listas, búsquedas |
| Crear/actualizar/eliminar variante | `cache:variant:{id}`, `cache:variant:product:*`, producto padre |
| Crear/actualizar/eliminar imagen | `cache:image:{id}`, `cache:image:product:*`, producto padre |
| Crear/actualizar/eliminar categoría | `cache:category:{id}`, `cache:category:list`, `cache:product:list:*` |

### Cache Warming

Al iniciar, la aplicación precarga en caché:
- Todas las categorías activas
- Primeras 100 variantes de productos activos

Se ejecuta de forma asíncrona 2 segundos después del arranque.

### Métricas de caché

```bash
curl http://localhost:8080/metrics/cache
```

Respuesta:
```json
{
  "hits": 1250,
  "misses": 45,
  "hit_rate": 96.5,
  "miss_rate": 3.5,
  "sets": 1295,
  "deletes": 120,
  "errors": 0
}
```

---

## 🧪 Testing

### Ejecutar tests

```bash
# Todos los tests
go test -v ./...

# Tests de un módulo específico
go test -v ./internal/modules/products/...

# Con cobertura
go test -v -cover ./...

# Con race detector (solo desarrollo)
go test -race ./...

# Un test específico
go test -v -run TestHandleLogin ./internal/modules/auth/...
```

### Cobertura por módulo

| Módulo | Tests | Cobertura |
|--------|-------|-----------|
| `auth` | 44+ | Handlers, service, middleware, 2FA, OAuth |
| `users` | 27+ | CRUD, RegisterAdmin, UpdateAvatar |
| `products` | 21+ | Productos, categorías, variantes, imágenes |
| `orders` | 18+ | CRUD, Confirm, Cancel, GetTaskStatus |
| `payments` | 12+ | Pagos, links, webhook |
| `inventory` | 12+ | Stock, reserve, release |
| `cart` | 10+ | CRUD del carrito |
| `shared` | 15+ | Health, middleware, rate limiting |

### Patrón de testing

- **SQLite in-memory** para repositorios y handlers (rápido, sin dependencias externas)
- **Function-based mocks** para servicios (sin code generation)
- **Table-driven tests** para casos múltiples
- **gin.TestMode** + `httptest.NewRecorder` para handlers

---

## 🔄 CI/CD

### GitHub Actions

La pipeline se ejecuta en **push** y **pull_request** a `main` y `develop`:

| Workflow | Trigger | Jobs |
|----------|---------|------|
| `ci.yml` | push/PR a main/develop | Lint → Test → Security → Build |
| `cd.yml` | push a main, tags | Test (race) → Build Docker → Push GHCR → Scan |

### Requisitos para pasar CI

- ✅ `go vet ./...` sin errores
- ✅ `golangci-lint run` sin errores
- ✅ Todos los tests pasan
- ✅ Binario compila correctamente

---

## 🔒 Seguridad

- **JWT**: Access tokens (15min) + Refresh tokens (7 días) con rotación
- **2FA**: TOTP con backup codes
- **OAuth2**: Google Sign-In
- **Rate Limiting**: Por endpoint (login: 5/min, default: 60/min)
- **CSRF**: Protección habilitada
- **CORS**: Orígenes configurables
- **Password hashing**: bcrypt con costo 10
- **SQL Injection**: Prevenir con GORM (queries parametrizadas)
- **XSS**: Sanitización de inputs

---

## 📖 Swagger

La documentación interactiva se genera automáticamente con `swag`:

```bash
# Regenerar docs
swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal

# Acceder
http://localhost:8080/swagger/index.html
```

Para autenticar en Swagger UI:
1. Click en **Authorize** 🔓
2. Ingresar: `Bearer <tu_token_jwt>`
3. Click **Authorize**

---

## 📄 Licencia

Apache 2.0 — ver [LICENSE](LICENSE) para más detalles.
