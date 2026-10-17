# SearchPix — API RR Solutions

API em Go para uma plataforma de fidelização de clientes por estabelecimento, com consulta PIX do Banco do Brasil, pontuação por NFC-e, mídia para TV e um módulo administrativo de designações.

A interface web fica no repositório [searchpix-ui](https://github.com/renanraquel/searchpix-ui).

## Funcionalidades

- **Fidelização multi-tenant:** estabelecimentos, usuários, clientes por CPF, produtos, saldo de pontos e histórico de resgates.
- **Pontuação:** lançamento manual por valor da compra e envio público de QR Code de NFC-e do Paraná, com consulta do valor na SEFAZ-PR e controle de reutilização da nota.
- **Resgates:** consulta pública de saldo e recompensas, além de operações no painel do estabelecimento.
- **Cadastro de lojistas:** cadastro e confirmação de e-mail via SMTP.
- **Carrossel para TV:** gerenciamento de conteúdo e exibição pública por estabelecimento.
- **PIX:** consulta por período via Banco do Brasil, com OAuth2, mTLS e habilitação por estabelecimento.
- **Designações:** cadastro de pessoas, organização semanal, candidatos, lembretes e geração de mensagens para WhatsApp; acesso restrito ao perfil administrador.

A regra atual de acúmulo é **1 ponto a cada R$ 5, arredondando para cima**: uma compra de R$ 12 gera 3 pontos. A implementação está em `internal/service/loyalty_points.go`.

## Arquitetura

- Go 1.25+, servidor HTTP com `net/http`.
- JWT para autenticação e bcrypt para senhas.
- PostgreSQL para persistência ou SQLite para desenvolvimento local.
- Migrations executadas automaticamente na inicialização.
- Mídia armazenada no banco ou, quando configurado, no Cloudflare R2.

```text
cmd/api/main.go          Entrada da API, composição dos serviços e rotas
internal/auth/           Login, JWT, perfis e acesso ao módulo PIX
internal/config/         Leitura das variáveis de ambiente
internal/db/             Conexão, migrations e suporte aos bancos
internal/handler/        Handlers HTTP
internal/model/          Modelos e DTOs
internal/repository/     Acesso aos dados
internal/service/        Regras de negócio e envio de e-mail
internal/nfcepr/         Leitura e consulta de NFC-e do Paraná
internal/bb/             Cliente Banco do Brasil, OAuth2 e mTLS
internal/r2/             Armazenamento de mídia no R2
internal/seed/           Dados iniciais e migração de mídia
```

A entrada da aplicação é `./cmd/api`. O `main.go` na raiz é um exemplo de IDE e não inicia a API.

## Desenvolvimento local

Requisitos: Go 1.25 ou superior e acesso à rede para baixar módulos. PostgreSQL, R2, SMTP e credenciais BB não são necessários para iniciar o fluxo local de fidelização.

Na raiz deste repositório:

```bash
go mod download
go mod verify
export SERVER_PORT=8080
export DATABASE_DRIVER=sqlite3
export DATABASE_URL='file:searchpix.local.db'
# Defina JWT_SECRET no ambiente antes de iniciar.
go run ./cmd/api
```

O exemplo cria um SQLite persistente no diretório atual. Não versione o banco gerado. Para um banco descartável, use `DATABASE_URL='file::memory:?cache=shared'`; os dados serão perdidos quando o processo terminar.

A aplicação também carrega um arquivo `.env`, se existir. Crie-o manualmente quando necessário; ele é ignorado pelo Git. Não há um `.env.example` versionado neste backend.

Ao iniciar em banco vazio, o seed cria o estabelecimento `ibimassas` e usuários iniciais, incluindo um administrador. Consulte `internal/seed/seed.go` para os dados de desenvolvimento. Revise esses usuários e credenciais antes de disponibilizar a aplicação fora do ambiente local. Como o seed já cria um tenant, o bootstrap não é o caminho normal do primeiro acesso.

Para iniciar a interface, siga o README de `searchpix-ui`, usando `VITE_API_URL=http://localhost:8080`.

## Configuração

### Aplicação e banco

| Variável | Uso |
| --- | --- |
| `SERVER_PORT` | Porta HTTP; padrão `8080`. |
| `JWT_SECRET` | Chave de assinatura JWT. Defina uma chave própria; não use o fallback do código em produção. |
| `DATABASE_DRIVER` | `sqlite3` ou `postgres`. Sem driver, uma URL preenchida seleciona PostgreSQL; sem ambos, usa SQLite em memória. |
| `DATABASE_URL` | DSN do banco. Com `sqlite3` e URL vazia, usa memória. |
| `PUBLIC_UI_ORIGIN` | Origem pública da interface usada nos links de confirmação de e-mail. |

Para PostgreSQL, configure `DATABASE_DRIVER=postgres` e uma `DATABASE_URL` válida no ambiente. As credenciais devem permanecer fora do repositório.

### Integrações opcionais

| Integração | Variáveis |
| --- | --- |
| Banco do Brasil | `BB_OAUTH_URL`, `BB_API_BASE_URL`, `BB_CLIENT_ID`, `BB_CLIENT_SECRET`, `BB_SCOPE`, `BB_GW_DEV_APP_KEY`, `BB_CERT_FILE`, `BB_KEY_FILE` |
| Habilitação PIX | `PIX_MODULE_TENANT_SLUGS`: slugs separados por vírgula; padrão `ibimassas`. Alinhe com `VITE_PIX_MODULE_TENANT_SLUGS` no frontend. |
| Login PIX legado | `AUTH_USER`, `AUTH_PASSWORD` para `/login`; a rota `/pix` exige JWT de fidelização com tenant habilitado. |
| E-mail | `SMTP_PROVIDER`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `EMAIL_FROM`. O provider `gmail` preenche host e porta padrão. |
| Cloudflare R2 | `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `R2_PUBLIC_BASE_URL` e `R2_ACCOUNT_ID` ou `R2_ENDPOINT`. |

As rotas PIX só são registradas quando as URLs BB estão configuradas e o cliente mTLS pode ser inicializado. Sem R2 configurado, a aplicação usa armazenamento de mídia no banco. O fluxo NFC-e depende de acesso à SEFAZ-PR; confirmação de cadastro de lojista depende do envio de e-mail.

## API

O mapa completo de rotas fica em `cmd/api/main.go`. Principais grupos:

| Rotas | Finalidade | Acesso |
| --- | --- | --- |
| `GET /api/tenants` | Listar estabelecimentos | Público |
| `POST /api/auth/login` | Autenticar e obter token, tenant e usuário | Público |
| `POST /api/bootstrap` | Criar primeiro tenant/usuário, apenas se não houver tenant | Público, condicionado ao estado do banco |
| `/api/products`, `/api/customers` | Cadastro e consulta; operações em subrotas `/create`, `/update` e `/delete` | JWT |
| `/api/points/customer`, `/api/points/earn` | Consultar cliente e lançar pontos | JWT |
| `/api/redemptions`, `/api/redemptions/redeem` | Histórico e resgate no balcão | JWT |
| `/api/public/register`, `/api/public/redemption`, `/api/public/redeem` | Cadastro, consulta e resgate do cliente | Público |
| `POST /api/public/nfce-points` | Acumular pontos pela NFC-e | Público |
| `/api/public/merchant-signup`, `/api/public/merchant-signup/verify-email` | Cadastro de lojista e confirmação de e-mail | Público |
| `/api/carousel/*`, `/api/public/carousel` | Gerenciamento e exibição de carrossel | JWT no gerenciamento; exibição pública |
| `/api/desig/*` | Pessoas, semanas, partes, designações e lembretes | JWT e perfil administrador |
| `POST /login`, `GET /pix?inicio=&fim=` | Login legado e consulta PIX | Dependem da configuração BB; consulta exige JWT e tenant habilitado |

Exemplo de login, após definir as credenciais do usuário no shell:

```bash
curl --fail-with-body http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  --data "{\"user\":\"$APP_USER\",\"password\":\"$APP_PASSWORD\"}"
```

Use o campo `token` retornado em `Authorization: Bearer <token>` nas chamadas protegidas. Não publique tokens ou respostas de login em logs.

## Validação

```bash
go test -mod=readonly ./...
go build -mod=readonly -o bin/searchpix ./cmd/api
curl --fail-with-body http://localhost:8080/api/tenants
```

Os testes existentes em `internal/nfcepr` cobrem parsing de CNPJ, chave e valor da nota. Eles não substituem validações de integração com BB, SMTP, R2, SEFAZ ou um fluxo completo de fidelização.

## Licença

Uso interno / conforme política do projeto.
