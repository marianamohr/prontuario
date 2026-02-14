# Prontuário Saúde — SaaS Multi-tenant

Sistema multi-tenant para profissionais liberais da área da saúde: prontuário digital, contratos com assinatura eletrônica, controle de acesso granular e auditoria LGPD.

**Stack:** React (Vite) + Go (backend) + PostgreSQL. Deploy preparado para Railway (Postgres + Backend + Frontend + Reminder cron).

Guia completo de deploy: **[DEPLOY_RAILWAY.md](DEPLOY_RAILWAY.md)**.

---

## Subir tudo com Docker Compose

Para rodar a aplicação completa (Postgres, MailHog, Backend e Frontend) em um comando:

1. **Inicie o Docker** (se usar Colima: `colima start`).
2. Na raiz do projeto:

```bash
docker-compose up -d
```

Ou com build forçado na primeira vez:

```bash
docker-compose up --build -d
```

- **Frontend:** http://localhost:80 (ou http://localhost)
- **Backend API:** http://localhost:8080
- **MailHog (e-mails):** http://localhost:8025
- **Postgres:** localhost:5432 (usuário `prontuario`, senha `prontuario_secret`, banco `prontuario`)

Na primeira execução as imagens do backend e frontend são construídas; o backend aplica as migrations automaticamente. Para usar um JWT seguro em local, crie um arquivo `.env` na raiz do projeto com:

```env
JWT_SECRET=seu-secret-com-pelo-menos-32-caracteres
```

Login de teste (após o seed): **Profissional** `profa@clinica-a.local` / `ChangeMe123!` — **Admin** `admin@prontuario.local` / `Admin123!`.

---

## 1. Obter DATABASE_URL na Railway

1. Crie um novo projeto na [Railway](https://railway.app).
2. Adicione o serviço **PostgreSQL** (Add → Database → PostgreSQL).
3. Abra o serviço Postgres e vá em **Variables** ou **Connect**.
4. A variável `DATABASE_URL` já vem preenchida (ex.: `postgresql://postgres:xxx@xxx.railway.app:5432/railway`).
5. Use essa mesma URL no serviço do **Backend**: em Variables do serviço backend, adicione `DATABASE_URL` com o valor copiado (ou use a referência do Railway, ex.: `${{Postgres.DATABASE_URL}}`).

---

## 2. Gerar JWT_SECRET

No terminal:

```bash
openssl rand -base64 32
```

Use o valor gerado como variável de ambiente `JWT_SECRET` no backend (mínimo 32 caracteres).

---

## 3. Gerar DATA_ENCRYPTION_KEYS (AES-256)

Para criptografia em repouso (CPF, conteúdo do prontuário):

```bash
# Gerar uma chave (32 bytes em base64)
openssl rand -base64 32
```

Configure no backend:

- `DATA_ENCRYPTION_KEYS=v1:SEU_BASE64_32_BYTES`
- `CURRENT_DATA_KEY_VERSION=v1`

Para rotação, use múltiplas entradas: `v1:key1,v2:key2` e defina `CURRENT_DATA_KEY_VERSION=v2` quando migrar.

---

## 4. Configurar SMTP (envio de e-mail)

O botão **"Enviar contrato para assinar"**, convites e redefinição de senha dependem do SMTP. Sem configuração válida, o backend tenta enviar mas o e-mail não chega.

- **Local com Docker (MailHog):**  
  O `docker-compose` já sobe o MailHog. Os e-mails **não vão para sua caixa de entrada**; eles ficam no MailHog. Abra **http://localhost:8025** para ver todos os e-mails enviados (incluindo "Contrato para assinatura"). Variáveis: `SMTP_HOST=mailhog` (no Docker) ou `SMTP_HOST=localhost` (backend rodando na máquina), `SMTP_PORT=1025`, deixe `SMTP_USER` e `SMTP_PASS` vazios.

- **Backend na máquina (sem Docker):**  
  Para o envio funcionar, é preciso um servidor SMTP na porta 1025 (ex.: subir só o MailHog: `docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog`) ou configurar um SMTP real (Gmail, SendGrid).

- **Receber no e-mail de verdade (Gmail):**  
  `SMTP_HOST=smtp.gmail.com`, `SMTP_PORT=587`, `SMTP_USER=seu@gmail.com`, `SMTP_PASS=senha-de-app` (crie em Conta Google → Segurança → Senhas de app).

- **SendGrid:** use o host e a chave de API como senha conforme a documentação do SendGrid.

Configure também: `SMTP_FROM_NAME`, `SMTP_FROM_EMAIL`, `APP_PUBLIC_URL` e `BACKEND_PUBLIC_URL`.

---

## 5. APP_PUBLIC_URL e BACKEND_PUBLIC_URL

- **APP_PUBLIC_URL:** URL pública do frontend (ex.: `http://localhost:5173` ou `https://app.seudominio.com`). Usada em links de e-mail (reset de senha, verificação).
- **BACKEND_PUBLIC_URL:** URL pública do backend (ex.: `http://localhost:8080` ou `https://api.seudominio.com`). Usada quando o frontend ou e-mails precisam chamar a API.

---

## 6. VITE_API_URL (Frontend)

No build do frontend, defina a URL da API:

- Local: pode deixar em branco ou `http://localhost:8080` se usar proxy no `vite.config`.
- Produção: ex.: `https://seu-backend.railway.app`.

No Railway, na variável de build do serviço frontend: `VITE_API_URL=https://seu-backend.railway.app`.

---

## 7. Ordem recomendada de setup

1. Subir Postgres (local: `docker-compose up -d postgres`; Railway: criar serviço Postgres).
2. Configurar `.env` (ou variáveis no Railway) com `DATABASE_URL`, `JWT_SECRET`, `DATA_ENCRYPTION_KEYS`, `CURRENT_DATA_KEY_VERSION`, SMTP, URLs.
3. Rodar o backend (migrations rodam no startup).
4. Rodar o frontend com `VITE_API_URL` apontando para o backend.
5. (Opcional) MailHog local: `docker-compose up -d mailhog`.

---

## 8. Testar assinatura e e-mail localmente

1. Suba Postgres e MailHog: `docker-compose up -d`.
2. Backend: `cd backend && go mod tidy && go run .` (com `DATABASE_URL` no `.env`).
3. Frontend: `cd frontend && npm run dev`.
4. Acesse a página de assinatura do contrato (com token válido). Após assinar, o e-mail deve aparecer no MailHog em http://localhost:8025.

---

## 9. Subir os serviços na Railway (passo a passo)

Para um guia completo (Postgres, Backend, Frontend e **Reminder cron**), variáveis e automação por repositório, use:

**[DEPLOY_RAILWAY.md](DEPLOY_RAILWAY.md)**

Resumo rápido:

1. **PostgreSQL** — New Project → Add → Database → PostgreSQL; anote `DATABASE_URL`.
2. **Backend** — Add → GitHub Repo, Root Directory `backend`; configurar Variables (DATABASE_URL, JWT_SECRET, DATA_ENCRYPTION_KEYS, CORS_ORIGINS, APP_PUBLIC_URL, BACKEND_PUBLIC_URL, SMTP, etc.).
3. **Frontend** — Add → mesmo repositório, Root Directory `frontend`; variável de build `VITE_API_URL` = URL do backend.
4. **Reminder (cron)** — Add → mesmo repositório, Root Directory `backend`, Dockerfile Path `Dockerfile.reminder`, Cron Schedule `0 11 * * *` (08:00 BRT); Variables: DATABASE_URL, Twilio (e opcional REMINDER_CRON_TZ).
5. **Health**: Backend expõe `GET /health` e `GET /ready`.

---

## Estrutura do repositório

```
/
  backend/              # API Go (root do serviço backend no Railway)
    cmd/reminder/       # Binário do cron de lembretes WhatsApp
    migrations/         # SQL (aplicadas no startup do backend e do reminder)
    Dockerfile          # Imagem da API
    Dockerfile.reminder # Imagem do job cron (Railway)
  frontend/             # React + Vite (root do serviço frontend)
  docker-compose.yml    # Postgres + MailHog + Backend + Frontend local
  railway.json
  .env.example
  DEPLOY_RAILWAY.md     # Guia de deploy no Railway (4 serviços)
  README.md
```

---

## Roles

- **PROFESSIONAL:** admin do tenant (clínica); acessa apenas dados do seu `clinic_id`.
- **LEGAL_GUARDIAN:** responsável/assinante; acessa apenas dados vinculados e autorizados.
- **SUPER_ADMIN:** backoffice global; ignora tenant; impersonate obrigatório para suporte.

---

## Seed local

Com `DATABASE_URL` apontando para o Postgres local, ao subir o backend é aplicado um seed (se não existir):

- **Super admin (backoffice):** `admin@prontuario.local` / `Admin123!`
- **Profissional A (Clínica A):** `profa@clinica-a.local` / `ChangeMe123!`
- **Profissional B (Clínica B):** `profb@clinica-b.local` / `ChangeMe123!`

Por clínica são criados **4 pacientes** (e responsáveis quando necessário):
- **2 adultos** (paciente = responsável): Maria Silva, João Santos — login responsável: `maria.silva@clinica-a.local` / `joao.santos@clinica-a.local` (senha `Guardian123!`); mesma coisa para Clínica B com sufixo `@clinica-b.local`.
- **2 crianças** (responsável é outro adulto): Ana Silva (pai Carlos Silva), Pedro Santos (mãe Fernanda Santos) — responsáveis: `carlos.silva@...`, `fernanda.santos@...` (senha `Guardian123!`).

Altere as senhas em produção.

---

## Testes

- **Backend (unitários, sem DB):** `cd backend && go test ./internal/auth ./internal/crypto/...`
- **Backend (isolamento multi-tenant):** com Postgres rodando e `DATABASE_URL` definida: `cd backend && go test -v -run TestTenantIsolation ./internal/repo`
- **Frontend:** `cd frontend && npm run build` para validar o build; `npm test -- --run` para testes unitários e integração (MSW).

Detalhes e cobertura: **[TESTING.md](TESTING.md)**.
