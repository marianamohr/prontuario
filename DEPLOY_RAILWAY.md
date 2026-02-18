# Deploy no Railway — Passo a passo

Este guia cobre o deploy do Prontuário no Railway com **4 componentes**:

1. **PostgreSQL** — banco de dados
2. **Backend** — API Go (migrations no startup)
3. **Frontend** — build estático (Vite) servido por Nginx
4. **Reminder** — job cron que envia lembretes de consulta por WhatsApp (roda diariamente e encerra)

---

## Visão geral da automação

A forma mais simples de automatizar é conectar o repositório GitHub ao projeto Railway e configurar **um serviço por componente**. Cada push na branch monitorada (ex.: `main`) dispara o build e deploy apenas dos serviços que usam o mesmo repositório.

- **Um repositório** → vários serviços (cada um com Root Directory e/ou Dockerfile próprios).
- **Variáveis de ambiente** podem ser compartilhadas via **Variables** do projeto ou por serviço.
- O **Reminder** é um Cron Job: não fica rodando 24h; o Railway executa o comando no horário definido e o processo deve terminar.

---

## Pré-requisitos

- Conta no [Railway](https://railway.app)
- Repositório GitHub do projeto conectado ao Railway (ou deploy por CLI)
- Chaves e segredos gerados (ver seção “Variáveis” abaixo)

---

## Onde conseguir cada variável de ambiente

| Variável | Onde conseguir |
|----------|----------------|
| **DATABASE_URL** | **Railway** → serviço PostgreSQL → aba **Variables** ou **Connect**. Copie a URL ou use a referência `${{Postgres.DATABASE_URL}}` em outros serviços do mesmo projeto. |
| **JWT_SECRET** | Gerar no terminal: `openssl rand -base64 32`. Use o resultado (mín. 32 caracteres). Nunca use o valor de exemplo do .env.example em produção. |
| **DATA_ENCRYPTION_KEYS** | Gerar uma chave: `openssl rand -base64 32`. Formato: `v1:RESULTADO_EM_BASE64`. Para rotação futura: `v1:chave1,v2:chave2`. |
| **CURRENT_DATA_KEY_VERSION** | Literal `v1` (deve coincidir com o prefixo usado em DATA_ENCRYPTION_KEYS). |
| **PORT** | Não definir manualmente no Railway; a plataforma injeta automaticamente. |
| **CORS_ORIGINS** | Você define: URL pública do **frontend** (ex.: `https://seu-app.up.railway.app`). Múltiplas origens separadas por vírgula, com `https://`. |
| **APP_PUBLIC_URL** | Você define: URL onde o usuário acessa o **frontend** no navegador (ex.: `https://seu-app.up.railway.app`). Usada em links de e-mail (reset de senha, verificação). |
| **BACKEND_PUBLIC_URL** | Você define: URL pública da **API** (ex.: `https://seu-backend.up.railway.app`). Após o primeiro deploy do backend, copie em **Settings** → **Networking** → **Public URL**. |
| **VITE_API_URL** | Mesmo valor que **BACKEND_PUBLIC_URL**: URL pública do backend. Definida nas **Variables** do serviço **Frontend** (no build). |
| **SMTP_HOST** | Provedor de e-mail: Gmail = `smtp.gmail.com`; SendGrid = `smtp.sendgrid.net`; Mailgun, etc. Ver documentação do provedor. |
| **SMTP_PORT** | Provedor: Gmail/SendGrid geralmente `587` (TLS). Pode ser `465` (SSL) ou `25`. |
| **SMTP_USER** | Sua conta de e-mail ou usuário da API (ex.: SendGrid: `apikey`). Gmail: seu e-mail completo. |
| **SMTP_PASS** | Senha do e-mail ou **Senha de app** (Gmail: Conta Google → Segurança → Senhas de app) ou **API Key** (SendGrid: dashboard → API Keys). |
| **SMTP_FROM_NAME** | Você define: nome que aparece como remetente (ex.: `Prontuário Saúde`). |
| **SMTP_FROM_EMAIL** | E-mail que aparece como remetente (deve ser válido; alguns SMTP exigem que seja da mesma conta). |
| **TWILIO_ACCOUNT_SID** | [Twilio Console](https://console.twilio.com/) → Dashboard. Em **Account Info**: **Account SID**. |
| **TWILIO_AUTH_TOKEN** | Mesma página: **Auth Token** (clique em “Show” para revelar). |
| **TWILIO_WHATSAPP_FROM** | Número Twilio no formato `whatsapp:+5511999999999`. Em [Twilio Messaging](https://console.twilio.com/us1/develop/sms/senders/whatsapp-sandbox) (WhatsApp Sandbox) ou no número provisionado no WhatsApp Business. Ex.: sandbox = `whatsapp:+14155238886`. |
| **REMINDER_CRON_TZ** | Opcional. Nome do fuso para “amanhã” no job (ex.: `America/Sao_Paulo`). Lista: [Wikipedia List of tz database time zones](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). Padrão do código: `America/Sao_Paulo`. |

**Resumo rápido de geração local:**

```bash
# JWT (copie a saída)
openssl rand -base64 32

# Chave de criptografia (copie e monte: v1:SAIDA_AQUI)
openssl rand -base64 32
```

---

## Passo 1: Criar o projeto e o Postgres

1. Acesse [Railway Dashboard](https://railway.app/dashboard) e crie um **New Project**.
2. **Add Service** → **Database** → **PostgreSQL**.
3. Aguarde o Postgres subir. Em **Variables** (ou **Connect**) copie a **DATABASE_URL** (ou use a referência `${{Postgres.DATABASE_URL}}` nos outros serviços).

---

## Passo 2: Serviço Backend (API)

1. **Add Service** → **GitHub Repo** e selecione o repositório do Prontuário.
2. No serviço criado:
   - **Settings** → **Root Directory**: `backend`.
   - **Settings** → **Builder**: Dockerfile (ou deixe Nixpacks; se usar Dockerfile, o Railway detecta `backend/Dockerfile`).
   - **Settings** → **Start Command**: deixe em branco (o Dockerfile já define `CMD ["./backend"]`) ou `./backend` se usar Nixpacks.
3. **Variables** (use referência ao Postgres quando possível):

   | Variável | Valor / Observação |
   |----------|--------------------|
   | `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` ou valor copiado |
   | `JWT_SECRET` | `openssl rand -base64 32` (mín. 32 caracteres) |
   | `DATA_ENCRYPTION_KEYS` | `v1:$(openssl rand -base64 32)` |
   | `CURRENT_DATA_KEY_VERSION` | `v1` |
   | `PORT` | Deixar em branco (Railway injeta) |
   | `CORS_ORIGINS` | URL do frontend (ex.: `https://seu-frontend.up.railway.app`) |
   | `APP_PUBLIC_URL` | URL pública do frontend |
   | `BACKEND_PUBLIC_URL` | URL pública do backend (ex.: `https://seu-backend.up.railway.app`) |
   | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` | SMTP para e-mails |
   | `SMTP_FROM_NAME`, `SMTP_FROM_EMAIL` | Nome e e-mail remetente |

4. **Deploy**: após o primeiro deploy, anote a URL pública do backend (ex.: `https://backend-production-xxxx.up.railway.app`). Ela será usada no frontend e em `BACKEND_PUBLIC_URL`.

---

## Passo 3: Serviço Frontend

1. **Add Service** → **GitHub Repo** (mesmo repositório).
2. No serviço:
   - **Settings** → **Root Directory**: `frontend`.
   - O build usa o `frontend/Dockerfile` (Nginx servindo o build do Vite).
3. **Variables** (importante no **build**):
   - **Build** (ou Variables do serviço):
     - `VITE_API_URL` = URL pública do backend (ex.: `https://seu-backend.up.railway.app`).
   - Não definir `PORT` manualmente; o Railway injeta e o container (nginx) escuta nessa porta.
4. **Deploy**: anote a URL do frontend e use-a em `CORS_ORIGINS` e `APP_PUBLIC_URL` do backend (e atualize o backend se ainda não tiver).

---

## Passo 4: Serviço Reminder (Cron Job)

Este serviço roda **uma vez por dia** (ex.: 08:00 BRT = 11:00 UTC) e encerra após enviar os lembretes por WhatsApp.

1. **Add Service** → **GitHub Repo** (mesmo repositório).
2. No serviço:
   - **Settings** → **Root Directory**: `backend`.
   - **Settings** → **Dockerfile Path**: `Dockerfile.reminder` (ou variável `RAILWAY_DOCKERFILE_PATH=Dockerfile.reminder`). Assim o Railway usa `backend/Dockerfile.reminder`, que builda apenas o binário `./cmd/reminder`.
   - **Settings** → **Cron Schedule**: `0 11 * * *` (todo dia às 11:00 UTC = 08:00 BRT). Ajuste conforme o fuso desejado.
   - O **Start Command** já é `./reminder` (definido no Dockerfile.reminder). O processo deve **terminar** após rodar; não deixe um servidor HTTP rodando aqui.
3. **Variables** (mesmas do backend, exceto PORT):
   - `DATABASE_URL` (referência ao Postgres).
   - `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_WHATSAPP_FROM` (lembretes WhatsApp). Se vazios, o job não envia mensagens.
   - `REMINDER_CRON_TZ` (opcional): ex. `America/Sao_Paulo` para calcular “amanhã” no fuso da clínica.
   - Pode copiar o restante (JWT, encryption keys, etc.) só se o reminder precisar; o reminder usa apenas DB e Twilio.

4. **Importante**: Cron Jobs no Railway executam o comando no horário; o container sobe, roda `./reminder` e deve **sair com código 0**. Não configure health check que espere um servidor.

---

## Resumo dos serviços

| Serviço    | Root Directory | Dockerfile            | Comportamento                          |
|-----------|----------------|------------------------|----------------------------------------|
| Postgres  | —              | —                      | Banco gerenciado pelo Railway          |
| Backend   | `backend`      | `Dockerfile`           | API HTTP; sobe e fica escutando        |
| Frontend  | `frontend`     | `Dockerfile`           | Build Vite + Nginx; sobe e fica escutando |
| Reminder  | `backend`      | `Dockerfile.reminder`  | Roda no cron e encerra                  |

---

## Automatizar deploy (um push = tudo atualizado)

1. **Conectar o repositório**
   - No projeto Railway, cada serviço que foi criado com “GitHub Repo” já está ligado ao mesmo repositório.
   - Em **Settings** do projeto (ou de cada serviço), defina a **branch** (ex.: `main`) e, se quiser, **Watch Paths** (ex.: backend só redeployar quando houver mudança em `backend/`).

2. **Watch Paths (opcional)**
   - Backend: `backend`
   - Frontend: `frontend`
   - Reminder: `backend` (ou `backend/cmd/reminder`, `backend/internal/reminder`, etc., se o Railway suportar).
   - Assim, um push que só mexe no frontend não dispara build do backend.

3. **Ordem de deploy**
   - Não é obrigatório definir ordem; o Backend e o Reminder usam o mesmo Postgres. O Frontend depende apenas do build com `VITE_API_URL` correto.

4. **Variáveis compartilhadas**
   - No Railway, em **Project** → **Variables**, é possível definir variáveis que todos os serviços enxergam. Serviço-specific override nas **Variables** do próprio serviço.

---

## Checklist pós-deploy

- [ ] Backend: `GET https://seu-backend.up.railway.app/health` retorna `{"status":"ok"}`.
- [ ] Backend: `GET https://seu-backend.up.railway.app/ready` retorna `{"status":"ready"}` (com Postgres ok).
- [ ] Frontend abre no navegador e o login chama o backend (checar rede/console).
- [ ] E-mails (SMTP) com URLs de produção em `APP_PUBLIC_URL`.
- [ ] Reminder: após o primeiro horário de cron, conferir logs do serviço Reminder (envios e possíveis erros).

---

## Troubleshooting

- **Backend não sobe**: confira `DATABASE_URL` e se o Postgres está no mesmo projeto. Veja logs do serviço.
- **Frontend em branco ou 404**: confirme `VITE_API_URL` no **build** (não só em runtime). Rebuild após alterar.
- **CORS**: inclua exatamente a URL do frontend (com protocolo) em `CORS_ORIGINS`.
- **Reminder não roda**: confirme Cron Schedule em UTC; confira **Dockerfile Path** = `Dockerfile.reminder` e que o binário termina (não fica em loop). Veja logs do serviço no horário do cron.
- **WhatsApp não envia**: verifique `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN` e `TWILIO_WHATSAPP_FROM` no serviço Reminder; número no formato esperado pelo Twilio (ex.: `whatsapp:+5511999999999`).

---

## Referências

- [Railway – Deploying a Monorepo](https://docs.railway.com/guides/monorepo)
- [Railway – Cron Jobs](https://docs.railway.com/guides/cron-jobs)
- [.env.example](.env.example) no repositório lista todas as variáveis usadas pelo backend e pelo reminder.
