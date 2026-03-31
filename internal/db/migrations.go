package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// Migrate executa as migrations conforme o driver
func Migrate(db *sql.DB, driver string) error {
	switch driver {
	case "postgres":
		return migratePostgres(db)
	case "sqlite3":
		return migrateSQLite(db)
	default:
		return fmt.Errorf("driver não suportado: %s", driver)
	}
}

func migratePostgres(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			background_image_data BYTEA,
			background_image_content_type VARCHAR(100),
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			email TEXT UNIQUE,
			email_verified BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			image_url TEXT,
			image_data BYTEA,
			image_content_type VARCHAR(100),
			description TEXT NOT NULL,
			points_required INTEGER NOT NULL CHECK (points_required > 0),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS customers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			cpf TEXT NOT NULL,
			name TEXT NOT NULL,
			phone TEXT NOT NULL,
			points_balance INTEGER NOT NULL DEFAULT 0 CHECK (points_balance >= 0),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(tenant_id, cpf)
		)`,
		`CREATE TABLE IF NOT EXISTS points_transactions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			amount INTEGER NOT NULL,
			kind TEXT NOT NULL CHECK (kind IN ('earn', 'redeem')),
			reference TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS redemptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			points_used INTEGER NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_tenant_cpf ON customers(tenant_id, cpf)`,
		`CREATE INDEX IF NOT EXISTS idx_products_tenant ON products(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_redemptions_customer ON redemptions(customer_id)`,
		`CREATE TABLE IF NOT EXISTS nfce_claims (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			access_key VARCHAR(44) NOT NULL,
			customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			value_reais NUMERIC(12,2) NOT NULL,
			points_awarded INTEGER NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(tenant_id, access_key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nfce_claims_tenant ON nfce_claims(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS tenant_nfce_emitters (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			cnpj VARCHAR(14) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(tenant_id, cnpj)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tenant_nfce_emitters_tenant ON tenant_nfce_emitters(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS user_email_verification_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_email_tokens_hash ON user_email_verification_tokens(token_hash)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration postgres: %w", err)
		}
	}
	// Colunas de imagem no banco (para deploys que já tinham a tabela sem elas)
	for _, q := range []string{
		`ALTER TABLE products ADD COLUMN image_data BYTEA`,
		`ALTER TABLE products ADD COLUMN image_content_type VARCHAR(100)`,
		`ALTER TABLE tenants ADD COLUMN background_image_data BYTEA`,
		`ALTER TABLE tenants ADD COLUMN background_image_content_type VARCHAR(100)`,
		`ALTER TABLE tenants ADD COLUMN nfce_emitter_cnpj VARCHAR(14)`,
		`INSERT INTO tenant_nfce_emitters (id, tenant_id, cnpj)
		 SELECT gen_random_uuid(), id, nfce_emitter_cnpj
		 FROM tenants
		 WHERE nfce_emitter_cnpj IS NOT NULL AND nfce_emitter_cnpj <> ''
		 ON CONFLICT (tenant_id, cnpj) DO NOTHING`,
		`ALTER TABLE users ADD COLUMN full_name TEXT`,
		`ALTER TABLE users ADD COLUMN cpf TEXT`,
		`ALTER TABLE users ADD COLUMN phone TEXT`,
		`ALTER TABLE users ADD COLUMN email TEXT`,
		`ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email)`,
	} {
		if _, err := db.Exec(q); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("migration postgres alter: %w", err)
			}
		}
	}
	return nil
}

func migrateSQLite(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tenants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			background_image_data BLOB,
			background_image_content_type TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			email TEXT UNIQUE,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			image_url TEXT,
			image_data BLOB,
			image_content_type TEXT,
			description TEXT NOT NULL,
			points_required INTEGER NOT NULL CHECK (points_required > 0),
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS customers (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			cpf TEXT NOT NULL,
			name TEXT NOT NULL,
			phone TEXT NOT NULL,
			points_balance INTEGER NOT NULL DEFAULT 0 CHECK (points_balance >= 0),
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now')),
			UNIQUE(tenant_id, cpf)
		)`,
		`CREATE TABLE IF NOT EXISTS points_transactions (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			amount INTEGER NOT NULL,
			kind TEXT NOT NULL CHECK (kind IN ('earn', 'redeem')),
			reference TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS redemptions (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			points_used INTEGER NOT NULL,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_tenant_cpf ON customers(tenant_id, cpf)`,
		`CREATE INDEX IF NOT EXISTS idx_products_tenant ON products(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_redemptions_customer ON redemptions(customer_id)`,
		`CREATE TABLE IF NOT EXISTS nfce_claims (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			access_key TEXT NOT NULL,
			customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			value_reais REAL NOT NULL,
			points_awarded INTEGER NOT NULL,
			created_at TEXT DEFAULT (datetime('now')),
			UNIQUE(tenant_id, access_key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nfce_claims_tenant ON nfce_claims(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS tenant_nfce_emitters (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			cnpj TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now')),
			UNIQUE(tenant_id, cnpj)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tenant_nfce_emitters_tenant ON tenant_nfce_emitters(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS user_email_verification_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			used_at TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_email_tokens_hash ON user_email_verification_tokens(token_hash)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration sqlite: %w", err)
		}
	}
	// Colunas de imagem (SQLite: ADD COLUMN suportado)
	for _, q := range []string{
		`ALTER TABLE products ADD COLUMN image_data BLOB`,
		`ALTER TABLE products ADD COLUMN image_content_type TEXT`,
		`ALTER TABLE tenants ADD COLUMN background_image_data BLOB`,
		`ALTER TABLE tenants ADD COLUMN background_image_content_type TEXT`,
		`ALTER TABLE tenants ADD COLUMN nfce_emitter_cnpj TEXT`,
		`INSERT OR IGNORE INTO tenant_nfce_emitters (id, tenant_id, cnpj)
		 SELECT lower(hex(randomblob(16))), id, nfce_emitter_cnpj
		 FROM tenants
		 WHERE nfce_emitter_cnpj IS NOT NULL AND nfce_emitter_cnpj <> ''`,
		`ALTER TABLE users ADD COLUMN full_name TEXT`,
		`ALTER TABLE users ADD COLUMN cpf TEXT`,
		`ALTER TABLE users ADD COLUMN phone TEXT`,
		`ALTER TABLE users ADD COLUMN email TEXT`,
		`ALTER TABLE users ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 0`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email)`,
	} {
		if _, err := db.Exec(q); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("migration sqlite alter: %w", err)
			}
		}
	}
	return nil
}
