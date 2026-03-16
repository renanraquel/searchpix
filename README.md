# SearchPix

API em Go para consulta de transações **PIX** (Banco do Brasil) e **programa de fidelização** multi-tenant (padarias). Inclui autenticação JWT, banco PostgreSQL ou SQLite e rotas públicas de resgate.

---

## Funcionalidades

### PIX (legado)
- Login com usuário/senha (env) e emissão de JWT
- Consulta PIX por período (data início/fim) via API BB
- mTLS e OAuth2 para comunicação com o Banco do Brasil

### Fidelização (multi-tenant)
- **Multi-tenant:** cada estabelecimento (tenant) tem ID e acessa apenas seus dados
- **Login por estabelecimento:** seleção de tenant + usuário/senha (armazenados no banco)
- **Produtos:** CRUD de produtos resgatáveis (imagem, descrição, pontos)
- **Clientes:** CRUD de clientes (CPF, nome, celular)
- **Pontos:** lançamento de pontos por compra (R$ 1 = 1 ponto), consulta de cliente por CPF
- **Resgate público:** tela sem autenticação acessível por link com tenant + CPF (pontos, itens disponíveis, resgates)

---

## Stack

- **Go 1.25**
- **Banco:** PostgreSQL (produção) ou SQLite (local/teste em memória)
- **Auth:** JWT, bcrypt para senhas
- **PIX:** net/http, mTLS, OAuth2 (BB)

---

## Como rodar

### Pré-requisitos

- Go 1.25+
- (Opcional) Certificado mTLS do BB para uso das rotas PIX
- (Opcional) PostgreSQL para fidelização em produção

### Variáveis de ambiente

Copie `.env.example` para `.env` e ajuste. Principais:

| Variável | Descrição |
|----------|-----------|
| `SERVER_PORT` | Porta do servidor (ex: `8080`) |
| `JWT_SECRET` | Chave para assinar JWT (obrigatório em produção) |
| `DATABASE_DRIVER` | `postgres` ou `sqlite3` |
| `DATABASE_URL` | DSN do banco. Vazio com driver `sqlite3` = memória |

**Exemplo produção (PostgreSQL no Render):**
```env
SERVER_PORT=8080
JWT_SECRET=sua-chave-segura
DATABASE_DRIVER=postgres
DATABASE_URL=postgresql://user:password@host:5432/database
```

**Exemplo local (SQLite em memória):**
```env
SERVER_PORT=8080
JWT_SECRET=dev-secret
# DATABASE_DRIVER e DATABASE_URL vazios = sqlite3 em memória
```

### Executar

```bash
go run ./cmd/api
```

O servidor sobe na porta definida em `SERVER_PORT`. As tabelas são criadas automaticamente na primeira execução (migrations).

### Primeiro uso (fidelização)

1. Crie o primeiro estabelecimento e usuário com um único request (só funciona quando não existe nenhum tenant):

```bash
curl -X POST http://localhost:8080/api/bootstrap \
  -H "Content-Type: application/json" \
  -d '{"tenant_name":"Minha Padaria","tenant_slug":"minha-padaria","username":"admin","password":"senha123"}'
```

2. Faça login na API (ou no frontend):

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"tenant_slug":"minha-padaria","user":"admin","password":"senha123"}'
```

3. Use o `token` retornado no header `Authorization: Bearer <token>` nas rotas protegidas.

---

## API

### Fidelização

| Método | Rota | Auth | Descrição |
|--------|------|------|-----------|
| GET | `/api/tenants` | Não | Lista estabelecimentos (para login) |
| POST | `/api/bootstrap` | Não | Cria primeiro tenant + usuário (só se não houver nenhum) |
| POST | `/api/auth/login` | Não | Login (tenant_slug, user, password) → token + tenant |
| GET | `/api/products` | JWT | Lista produtos do tenant |
| POST | `/api/products/create` | JWT | Cria produto |
| POST | `/api/products/update?id=` | JWT | Atualiza produto |
| POST | `/api/products/delete?id=` | JWT | Exclui produto |
| GET | `/api/customers` | JWT | Lista clientes |
| POST | `/api/customers/create` | JWT | Cria cliente |
| POST | `/api/customers/update?id=` | JWT | Atualiza cliente |
| POST | `/api/customers/delete?id=` | JWT | Exclui cliente |
| GET | `/api/points/customer?cpf=` | JWT | Busca cliente por CPF (para lançar pontos) |
| POST | `/api/points/earn` | JWT | Lança pontos (body: cpf, value_reais) |
| GET | `/api/public/redemption?tenant=slug&cpf=` | Não | Dados do cliente + produtos + resgates (tela pública) |
| POST | `/api/public/redeem` | Não | Resgata produto (body: tenant_slug, cpf, product_id) |

### PIX (quando BB configurado)

| Método | Rota | Auth | Descrição |
|--------|------|------|-----------|
| POST | `/login` | Não | Login legado (user, password do env) |
| GET | `/pix?inicio=&fim=` | JWT | Consulta PIX no período |

---

## Estrutura do projeto

```
searchpix/
├── cmd/api/main.go          # Entrada, rotas, CORS
├── internal/
│   ├── auth/                # Login (legado + fidelização), JWT, context
│   ├── bb/                  # Cliente BB (mTLS, OAuth, PIX)
│   ├── config/              # Config (env)
│   ├── db/                  # Conexão e migrations (Postgres + SQLite)
│   ├── handler/             # Handlers HTTP (PIX + fidelização + bootstrap)
│   ├── model/               # DTOs (PIX, loyalty)
│   ├── repository/          # Acesso a dados (tenants, users, products, customers, points, redemptions)
│   └── service/             # Regras de negócio (PIX, pontos, resgate)
├── go.mod
└── README.md
```

---

## Licença

Uso interno / conforme política do projeto.
