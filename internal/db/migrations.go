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
			background_storage_key TEXT,
			background_image_url TEXT,
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
			image_storage_key TEXT,
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
		`CREATE TABLE IF NOT EXISTS page_visits (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			page_key VARCHAR(80) NOT NULL,
			page_path VARCHAR(255) NOT NULL,
			query_string TEXT,
			tenant_slug VARCHAR(120),
			referrer TEXT,
			user_agent TEXT,
			ip VARCHAR(100),
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_page_visits_key_created_at ON page_visits(page_key, created_at)`,
		`CREATE TABLE IF NOT EXISTS carousel_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			media_type VARCHAR(10) NOT NULL CHECK (media_type IN ('image', 'video')),
			title TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			media_data BYTEA,
			content_type VARCHAR(100) NOT NULL DEFAULT '',
			storage_key TEXT,
			media_url TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_carousel_items_tenant ON carousel_items(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS carousel_settings (
			tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
			image_duration_seconds INTEGER NOT NULL DEFAULT 20 CHECK (image_duration_seconds > 0),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS nfce_claim_duplicate_attempts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			access_key VARCHAR(44) NOT NULL,
			customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
			cpf TEXT,
			qr_payload TEXT,
			source VARCHAR(40) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nfce_dup_attempts_access_key ON nfce_claim_duplicate_attempts(access_key)`,
		`CREATE INDEX IF NOT EXISTS idx_nfce_dup_attempts_tenant_created ON nfce_claim_duplicate_attempts(tenant_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS desig_tipos_parte (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			codigo TEXT NOT NULL UNIQUE,
			categoria TEXT NOT NULL,
			nome TEXT NOT NULL,
			fixa INTEGER NOT NULL DEFAULT 0,
			permite_ajudante INTEGER NOT NULL DEFAULT 0,
			ordem INTEGER NOT NULL DEFAULT 0,
			duracao_padrao_min INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS desig_pessoas (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			nome TEXT NOT NULL,
			tipo TEXT NOT NULL CHECK (tipo IN ('estudante', 'servo', 'anciao')),
			sexo TEXT NOT NULL CHECK (sexo IN ('M', 'F')),
			telefone TEXT,
			ativo INTEGER NOT NULL DEFAULT 1,
			qualificado_tesouros INTEGER NOT NULL DEFAULT 0,
			disponivel_oracao_inicial INTEGER NOT NULL DEFAULT 1,
			qualificado_presidente INTEGER NOT NULL DEFAULT 0,
			capacidade TEXT NOT NULL DEFAULT 'pleno' CHECK (capacidade IN ('pleno', 'limitado')),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_pessoas_tenant ON desig_pessoas(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS desig_semanas (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			data_inicio DATE NOT NULL,
			data_fim DATE NOT NULL,
			data_reuniao DATE NOT NULL,
			rotulo TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(tenant_id, data_inicio)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_semanas_tenant ON desig_semanas(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS desig_partes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			semana_id UUID NOT NULL REFERENCES desig_semanas(id) ON DELETE CASCADE,
			tipo_parte_id UUID NOT NULL REFERENCES desig_tipos_parte(id),
			titulo TEXT NOT NULL,
			tema TEXT NOT NULL DEFAULT '',
			duracao_min INTEGER NOT NULL DEFAULT 0,
			ordem INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_partes_semana ON desig_partes(semana_id)`,
		`CREATE TABLE IF NOT EXISTS desig_designacoes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			parte_id UUID NOT NULL REFERENCES desig_partes(id) ON DELETE CASCADE,
			pessoa_id UUID NOT NULL REFERENCES desig_pessoas(id) ON DELETE CASCADE,
			papel TEXT NOT NULL CHECK (papel IN ('dono', 'ajudante')),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(parte_id, papel)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_designacoes_parte ON desig_designacoes(parte_id)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_designacoes_pessoa ON desig_designacoes(pessoa_id)`,
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
		`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'tenant'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email)`,
		`ALTER TABLE carousel_items ADD COLUMN storage_key TEXT`,
		`ALTER TABLE carousel_items ADD COLUMN media_url TEXT`,
		`ALTER TABLE carousel_items ALTER COLUMN media_data DROP NOT NULL`,
		`ALTER TABLE products ADD COLUMN image_storage_key TEXT`,
		`ALTER TABLE tenants ADD COLUMN background_storage_key TEXT`,
		`ALTER TABLE tenants ADD COLUMN background_image_url TEXT`,
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
			background_storage_key TEXT,
			background_image_url TEXT,
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
			image_storage_key TEXT,
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
		`CREATE TABLE IF NOT EXISTS page_visits (
			id TEXT PRIMARY KEY,
			page_key TEXT NOT NULL,
			page_path TEXT NOT NULL,
			query_string TEXT,
			tenant_slug TEXT,
			referrer TEXT,
			user_agent TEXT,
			ip TEXT,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_page_visits_key_created_at ON page_visits(page_key, created_at)`,
		`CREATE TABLE IF NOT EXISTS carousel_items (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			media_type TEXT NOT NULL CHECK (media_type IN ('image', 'video')),
			title TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			media_data BLOB,
			content_type TEXT NOT NULL DEFAULT '',
			storage_key TEXT,
			media_url TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_carousel_items_tenant ON carousel_items(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS carousel_settings (
			tenant_id TEXT PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
			image_duration_seconds INTEGER NOT NULL DEFAULT 20 CHECK (image_duration_seconds > 0),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS nfce_claim_duplicate_attempts (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			access_key TEXT NOT NULL,
			customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL,
			cpf TEXT,
			qr_payload TEXT,
			source TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nfce_dup_attempts_access_key ON nfce_claim_duplicate_attempts(access_key)`,
		`CREATE INDEX IF NOT EXISTS idx_nfce_dup_attempts_tenant_created ON nfce_claim_duplicate_attempts(tenant_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS desig_tipos_parte (
			id TEXT PRIMARY KEY,
			codigo TEXT NOT NULL UNIQUE,
			categoria TEXT NOT NULL,
			nome TEXT NOT NULL,
			fixa INTEGER NOT NULL DEFAULT 0,
			permite_ajudante INTEGER NOT NULL DEFAULT 0,
			ordem INTEGER NOT NULL DEFAULT 0,
			duracao_padrao_min INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS desig_pessoas (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			nome TEXT NOT NULL,
			tipo TEXT NOT NULL CHECK (tipo IN ('estudante', 'servo', 'anciao')),
			sexo TEXT NOT NULL CHECK (sexo IN ('M', 'F')),
			telefone TEXT,
			ativo INTEGER NOT NULL DEFAULT 1,
			qualificado_tesouros INTEGER NOT NULL DEFAULT 0,
			disponivel_oracao_inicial INTEGER NOT NULL DEFAULT 1,
			qualificado_presidente INTEGER NOT NULL DEFAULT 0,
			capacidade TEXT NOT NULL DEFAULT 'pleno' CHECK (capacidade IN ('pleno', 'limitado')),
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_pessoas_tenant ON desig_pessoas(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS desig_semanas (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			data_inicio TEXT NOT NULL,
			data_fim TEXT NOT NULL,
			data_reuniao TEXT NOT NULL,
			rotulo TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now')),
			UNIQUE(tenant_id, data_inicio)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_semanas_tenant ON desig_semanas(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS desig_partes (
			id TEXT PRIMARY KEY,
			semana_id TEXT NOT NULL REFERENCES desig_semanas(id) ON DELETE CASCADE,
			tipo_parte_id TEXT NOT NULL REFERENCES desig_tipos_parte(id),
			titulo TEXT NOT NULL,
			tema TEXT NOT NULL DEFAULT '',
			duracao_min INTEGER NOT NULL DEFAULT 0,
			ordem INTEGER NOT NULL DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_partes_semana ON desig_partes(semana_id)`,
		`CREATE TABLE IF NOT EXISTS desig_designacoes (
			id TEXT PRIMARY KEY,
			parte_id TEXT NOT NULL REFERENCES desig_partes(id) ON DELETE CASCADE,
			pessoa_id TEXT NOT NULL REFERENCES desig_pessoas(id) ON DELETE CASCADE,
			papel TEXT NOT NULL CHECK (papel IN ('dono', 'ajudante')),
			created_at TEXT DEFAULT (datetime('now')),
			UNIQUE(parte_id, papel)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_designacoes_parte ON desig_designacoes(parte_id)`,
		`CREATE INDEX IF NOT EXISTS idx_desig_designacoes_pessoa ON desig_designacoes(pessoa_id)`,
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
		`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'tenant'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email)`,
		`ALTER TABLE carousel_items ADD COLUMN storage_key TEXT`,
		`ALTER TABLE carousel_items ADD COLUMN media_url TEXT`,
		`ALTER TABLE products ADD COLUMN image_storage_key TEXT`,
		`ALTER TABLE tenants ADD COLUMN background_storage_key TEXT`,
		`ALTER TABLE tenants ADD COLUMN background_image_url TEXT`,
	} {
		if _, err := db.Exec(q); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("migration sqlite alter: %w", err)
			}
		}
	}
	return nil
}
