# Variáveis de ambiente do Backend (Prontuário)

Lista de todas as variáveis usadas pelo backend e passo a passo para obter cada uma.

---

## Resumo rápido

| Variável | Obrigatória | Default (se houver) | Uso |
|----------|-------------|---------------------|-----|
| `DATABASE_URL` | **Sim** | — | Conexão PostgreSQL |
| `JWT_SECRET` | **Sim** (em prod) | valor fraco em dev | Assinatura dos tokens de login |
| `PORT` | Não | `8080` | Porta HTTP do servidor |
| `CORS_ORIGINS` | Não | `http://localhost:5173` | Origens permitidas no CORS |
| `APP_PUBLIC_URL` | Não* | `http://localhost:5173` | URL pública do frontend (links em e-mails e contratos) |
| `BACKEND_PUBLIC_URL` | Não | `http://localhost:8080` | URL pública do backend |
| `DATA_ENCRYPTION_KEYS` | Não | chave de exemplo | Criptografia de CPF/dados sensíveis |
| `CURRENT_DATA_KEY_VERSION` | Não | `v1` | Versão da chave de criptografia em uso |
| `SMTP_*` | Não* | localhost:1025 | Envio de e-mails (convites, reset de senha, contratos) |
| `TWILIO_*` | Não | — | WhatsApp (lembretes de consulta) |

\* Em produção, para enviar e-mails de verdade e links corretos, `APP_PUBLIC_URL` e SMTP costumam ser configurados.

---

## 1. DATABASE_URL (obrigatória)

**O que é:** String de conexão do PostgreSQL (usuário, senha, host, porta, nome do banco e opções).

**Formato:**  
`postgres://USUARIO:SENHA@HOST:PORTA/NOME_DO_BANCO?sslmode=disable`  
(em produção use `sslmode=require` ou o que o provedor indicar.)

### Onde conseguir

**Opção A – Railway**  
1. Acesse [railway.app](https://railway.app) e faça login.  
2. Crie um novo projeto (ou use o existente).  
3. Clique em **"+ New"** → **"Database"** → **"PostgreSQL"**.  
4. Após o deploy, abra o serviço do Postgres.  
5. Aba **"Variables"** ou **"Connect"**: copie a variável `DATABASE_URL` (já vem no formato correto).  
6. Use essa mesma URL nas variáveis do serviço do **backend** (copiar/colar em "Variables").

**Opção B – Supabase**  
1. Acesse [supabase.com](https://supabase.com) → seu projeto.  
2. **Settings** (ícone engrenagem) → **Database**.  
3. Em **Connection string** escolha **URI**.  
4. Copie a URI e substitua `[YOUR-PASSWORD]` pela senha do banco (definida na criação do projeto).  
5. Exemplo: `postgresql://postgres:SUA_SENHA@db.xxxxx.supabase.co:5432/postgres`

**Opção C – Neon, Render, ElephantSQL, etc.**  
No painel do serviço, procure por **Connection string**, **Database URL** ou **PostgreSQL URL** e copie. O formato é sempre `postgres://...` ou `postgresql://...`.

**Opção D – Local (Docker)**  
```bash
# Subir Postgres
docker run -d --name prontuario-db -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=prontuario -p 5432:5432 postgres:15

# DATABASE_URL
DATABASE_URL=postgres://postgres:postgres@localhost:5432/prontuario?sslmode=disable
```

---

## 2. JWT_SECRET (obrigatória em produção)

**O que é:** Chave secreta usada para assinar e validar os tokens JWT (login de profissionais, responsáveis e super admin). Deve ser longa e aleatória.

**Requisito no código:** pelo menos **32 caracteres** (senão o backend usa um default fraco, só para dev).

### Onde conseguir

**Gerar uma chave segura (escolha uma opção):**

**Opção A – OpenSSL (recomendado)**  
No terminal:
```bash
openssl rand -base64 32
```
Use a string gerada como valor de `JWT_SECRET` (ex.: `JWT_SECRET=abc123...`).

**Opção B – Node.js**  
```bash
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

**Opção C – Gerador online**  
Use um gerador de “random string” ou “password” com 32+ caracteres; em produção prefira openssl/node no seu próprio computador.

**Configurar:**  
- Local: no `.env` ou no comando: `export JWT_SECRET="sua_chave_gerada"`.  
- Railway/Render/etc.: em **Variables** do serviço do backend, crie `JWT_SECRET` e cole o valor (sem aspas no valor, a menos que a plataforma exija).

---

## 3. PORT (opcional)

**O que é:** Porta em que o servidor HTTP do backend escuta.

**Default:** `8080`.

**Onde definir:**  
- **Railway / Render / Fly.io:** muitas vezes a plataforma define `PORT` automaticamente; não é obrigatório criar. Se definir, use o valor que a plataforma espera (ex.: Railway já injeta `PORT`).  
- **Local:** pode deixar 8080 ou definir no `.env`: `PORT=8080`.

---

## 4. CORS_ORIGINS (opcional)

**O que é:** Lista de origens permitidas para requisições do browser (frontend). Evita que outros domínios consumam sua API.

**Default:** `http://localhost:5173` (Vite em dev).

**Formato:** várias origens separadas por vírgula, sem espaços extras (ou com espaços que o backend trima).  
Ex.: `https://meuapp.railway.app,https://www.meudominio.com.br`

**Onde definir:**  
- **Produção:** nas variáveis do backend (Railway, etc.), defina `CORS_ORIGINS` com a URL pública do frontend.  
  - Ex.: se o front está em `https://front-production-xxx.up.railway.app`, use:  
    `CORS_ORIGINS=https://front-production-xxx.up.railway.app`  
- **Vários ambientes:** pode incluir mais de uma URL separada por vírgula.

---

## 5. APP_PUBLIC_URL (recomendada em produção)

**O que é:** URL pública do **frontend**. Usada em links enviados por e-mail (convites, reset de senha, link de assinatura de contrato, etc.).

**Default:** `http://localhost:5173`.

**Onde conseguir:**  
- **Produção:** é a URL em que os usuários acessam o front (ex.: `https://front-production-df80.up.railway.app`).  
- Defina nas variáveis do backend, ex.:  
  `APP_PUBLIC_URL=https://front-production-df80.up.railway.app`  
- Não use barra no final.

---

## 6. BACKEND_PUBLIC_URL (opcional)

**O que é:** URL pública do **backend** (API). Usada quando algum link ou recurso precisa apontar explicitamente para a API.

**Default:** `http://localhost:8080`.

**Onde conseguir:**  
- **Produção:** URL do serviço do backend (ex.: `https://back-production-xxx.up.railway.app`).  
- Só é necessário se algum fluxo (ex.: e-mail, integração) precisar dessa URL; caso contrário pode deixar o default em dev.

---

## 7. DATA_ENCRYPTION_KEYS e CURRENT_DATA_KEY_VERSION (opcionais, sensíveis)

**O que é:** Chaves para criptografia de dados sensíveis (ex.: CPF). O formato suportado é por versão, ex.: `v1:CHAVE_BASE64`.

**Default no código:**  
- `DATA_ENCRYPTION_KEYS`: valor de exemplo (não use em produção).  
- `CURRENT_DATA_KEY_VERSION`: `v1`.

### Onde conseguir (gerar chave)

**Gerar uma chave AES-256 (32 bytes) em Base64:**
```bash
openssl rand -base64 32
```

**Configurar:**  
- Ex.: `DATA_ENCRYPTION_KEYS=v1:XXXXXXXXXXXX_BASE64_32_BYTES_XXXXXXXX`  
- `CURRENT_DATA_KEY_VERSION=v1`  
Em produção, use sempre chaves geradas por você e nunca o default do código.

---

## 8. SMTP (envio de e-mails) – opcional

Usado para: convites (profissional e paciente), reset de senha, envio de link de contrato por e-mail.

| Variável | Significado | Default |
|----------|-------------|---------|
| `SMTP_HOST` | Servidor SMTP | `localhost` |
| `SMTP_PORT` | Porta SMTP | `1025` |
| `SMTP_USER` | Usuário (se exigido) | — |
| `SMTP_PASS` | Senha (se exigido) | — |
| `SMTP_FROM_NAME` | Nome do remetente | `Prontuário Saúde` |
| `SMTP_FROM_EMAIL` | E-mail do remetente | `noreply@localhost` |

**Importante:** O backend só habilita o envio de e-mails se `APP_PUBLIC_URL` estiver configurado. Caso contrário, os e-mails ficam desativados.

### Serviços com plano gratuito (para começar)

| Serviço | Plano gratuito | SMTP |
|---------|----------------|------|
| **Mailtrap** | 4.000 e-mails/mês, sem cartão | Sim – [mailtrap.io](https://mailtrap.io) |
| **SMTP2GO** | 1.000 e-mails/mês (200/dia), plano sem expiração | Sim – [smtp2go.com](https://www.smtp2go.com) |
| **Mailgun** | ~100 e-mails/dia (~3.000/mês) | Sim – smtp.mailgun.org |
| **Brevo** (ex-Sendinblue) | 300 e-mails/dia | Sim – smtp-relay.brevo.com |
| **Mailjet** | 200 e-mails/dia (6.000/mês no free tier) | Sim – smtp.mailjet.com |
| **Resend** | 3.000/mês (API; SMTP ver doc.) | Ver documentação |

**Mailtrap** e **SMTP2GO** são boas opções para começar: plano gratuito permanente, sem cartão, e foco em entregabilidade.

**Ainda não tenho domínio?**  
- **Mailtrap:** ao criar a conta, o Mailtrap fornece um **domínio de demonstração** (Demo domain). Use esse domínio em **Sending Domains** e defina `SMTP_FROM_EMAIL` com um endereço @ desse domínio (ex.: `noreply@seu-demo.mailtrap.io`). Assim você envia de verdade sem ter domínio próprio.  
- **Gmail:** use a Opção D abaixo com seu e-mail @gmail.com e uma “Senha de app” — não exige domínio (bom para testes e baixo volume).

### Onde conseguir

**Opção A – Desenvolvimento local (MailHog)**  
1. Subir MailHog (ex.: `docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog`).  
2. Não precisa definir `SMTP_USER`/`SMTP_PASS`.  
3. Default `localhost:1025` já funciona; mensagens aparecem em `http://localhost:8025`.

**Opção B – Resend**  
1. Crie conta em [resend.com](https://resend.com).  
2. Domínio ou “Send with Resend” (domínio de teste).  
3. Em **API Keys**, crie uma chave.  
4. Resend usa API HTTP; se o backend só falar SMTP, use um adaptador SMTP do Resend (ver documentação deles) ou outro provedor SMTP abaixo.

**Opção C – SendGrid**  
1. Conta em [sendgrid.com](https://sendgrid.com).  
2. **Settings** → **Sender Authentication** (verificar remetente ou domínio).  
3. **Settings** → **API Keys** → criar chave.  
4. SendGrid oferece **SMTP Relay**: anote host (ex. `smtp.sendgrid.net`), porta (587), usuário (`apikey`) e senha (a API key).  
5. Defina:  
   `SMTP_HOST=smtp.sendgrid.net`  
   `SMTP_PORT=587`  
   `SMTP_USER=apikey`  
   `SMTP_PASS=SUA_API_KEY`  
   `SMTP_FROM_EMAIL=seu@email.com`  
   `SMTP_FROM_NAME=Nome do App`

**Opção D – Gmail (não recomendado para produção)**  
1. Ativar “Acesso a app menos seguro” ou usar “Senha de app” (conta Google).  
2. Host: `smtp.gmail.com`, porta: `587`, usuário: seu e-mail, senha: senha de app.  
3. Definir `SMTP_FROM_EMAIL` e `SMTP_FROM_NAME`.

**Opção E – Mailtrap (gratuito – 4.000/mês)**  

Passo a passo (bem simples):

| Passo | O que fazer | Dica |
|-------|-------------|------|
| **1** | Abra o navegador e acesse [mailtrap.io](https://mailtrap.io). | — |
| **2** | Clique em **Sign Up** (ou “Criar conta”). Crie a conta com e-mail e senha. **Não pede cartão.** | Use um e-mail que você acesse (vai receber confirmação). |
| **3** | Confirme o e-mail se o Mailtrap mandar um link. Depois faça **login**. | — |
| **4** | No menu da esquerda, na seção **General**, clique em **Sending Domains**. | — |
| **5** | Na lista de domínios, clique no domínio que o Mailtrap já criou (tipo `something.mailtrap.io`). **Anote o nome** (ex.: `abc123.mailtrap.io`) — é o que você usa no `SMTP_FROM_EMAIL`. | É o domínio de demonstração; não precisa criar um. |
| **6** | Dentro da página do domínio, abra a aba **Integrations**. Clique em **Integrate** em **Transactional Stream**. | Não use Bulk Stream; use Transactional. |
| **7** | Mude o seletor para **SMTP** (em vez de API). A tela mostra **Host**, **Port** (ex.: 587), **Username** e **Password**. Anote ou copie esses quatro valores. | Use Show na senha se precisar ver. |
| **8** | Abra o arquivo `.env` do backend do projeto (na pasta `backend`). | Se não existir, crie um arquivo chamado `.env` nessa pasta. |
| **9** | Adicione ou edite estas linhas (troque pelos valores que você anotou): | — |
| | `SMTP_HOST=` o Host do passo 7 (ex.: `live.smtp.mailtrap.io`) | — |
| | `SMTP_PORT=587` | — |
| | `SMTP_USER=` o Username do passo 7 | — |
| | `SMTP_PASS=` a Password do passo 7 | — |
| | `SMTP_FROM_EMAIL=noreply@SEU_DOMINIO_DO_PASSO_5` | Troque `SEU_DOMINIO_DO_PASSO_5` pelo domínio que você anotou (ex.: `noreply@abc123.mailtrap.io`). |
| | `SMTP_FROM_NAME=Prontuário Saúde` | Pode mudar o nome se quiser. |
| **10** | Salve o `.env`. Reinicie o backend se ele estiver rodando. | Pronto: o app vai usar o Mailtrap para enviar e-mails. |

**Resumo:** em **Sending Domains** você clica no domínio → aba **Integrations** → **Integrate** (Transactional Stream) → muda para **SMTP** e copia Host, Port, Username e Password para o `.env`. O "API/SMTP" do menu é para usar a API por token; as credenciais SMTP ficam mesmo em Sending Domains → Integrations.

**Atenção:** use sempre o **Host** que aparece na tela de Integrations do seu domínio de envio. Para envio real (Email Sending) o host é **`live.smtp.mailtrap.io`**. Não use `smtp.mailtrap.io` (sandbox) — com esse host os e-mails não são entregues de verdade.

**Como saber se funcionou?**  
1. **Ao subir o backend:** no terminal deve aparecer algo como `SMTP configurado: live.smtp.mailtrap.io:587`. Se aparecer "SMTP não configurado" ou "APP_PUBLIC_URL vazio", confira `APP_PUBLIC_URL` e as variáveis SMTP no `.env`.  
2. **Teste real:** na tela de login do app, use **"Esqueci a senha"** e informe um e-mail que exista no sistema (profissional, responsável ou super admin). Com domínio demo do Mailtrap, o e-mail só é entregue **para o mesmo e-mail da sua conta Mailtrap** — use esse e-mail no cadastro para testar.  
3. **No Mailtrap:** no menu **Transactional** → **API/SMTP** (ou **Email Logs**), veja se o e-mail enviado aparece na lista. Assim você confirma que o backend conseguiu enviar.

**Opção F – SMTP2GO (gratuito – 1.000/mês, sem expiração)**  
1. Crie conta em [smtp2go.com](https://www.smtp2go.com).  
2. **Settings** → **SMTP Users**: crie um usuário SMTP e anote usuário e senha.  
3. Host: `mail.smtp2go.com`, porta: `2525` (ou 587).  
4. Defina `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM_EMAIL` e `SMTP_FROM_NAME`.

**Opção G – Brevo / Sendinblue (gratuito – 300/dia)**  
1. Conta em [brevo.com](https://www.brevo.com).  
2. **SMTP & API** → credenciais SMTP: host `smtp-relay.brevo.com`, porta 587.  
3. Use o login e a chave SMTP como usuário e senha.

**Opção H – Amazon SES, Mailgun, etc.**  
No painel do serviço, procure “SMTP settings” e use host, porta, usuário e senha fornecidos.

---

## 9. Twilio (WhatsApp – lembretes) – opcional

Usado pelo job de **lembretes de consulta** (WhatsApp). Se não configurar, lembretes por WhatsApp ficam desativados.

| Variável | Significado |
|----------|-------------|
| `TWILIO_ACCOUNT_SID` | Account SID da conta Twilio |
| `TWILIO_AUTH_TOKEN` | Auth Token da conta Twilio |
| `TWILIO_WHATSAPP_FROM` | Número/identificador do WhatsApp (ex.: `whatsapp:+14155238886`) |

### Onde conseguir

1. Criar conta em [twilio.com](https://www.twilio.com).  
2. No **Console** (dashboard): anote **Account SID** e **Auth Token** (revele o token).  
3. **Messaging** → **Try it out** → **Send a WhatsApp message**: siga o passo a passo para ativar o sandbox ou vincular seu número.  
4. O “From” no WhatsApp costuma ser algo como `whatsapp:+14155238886` (número do sandbox ou do seu número Twilio).  
5. Defina as três variáveis no ambiente do backend (ou no `cmd/reminder` se rodar em processo separado).  
6. O job de reminder também usa `DATABASE_URL` e, se aplicável, `REMINDER_CRON_TZ` (timezone do cron).

---

## 10. REMINDER_CRON_TZ (opcional – só para o job de reminder)

**O que é:** Timezone para o agendamento do cron de lembretes (ex.: `America/Sao_Paulo`).

**Onde definir:** No ambiente do processo que roda o `cmd/reminder` (se for um serviço separado). Se não definir, o binário de reminder pode usar o timezone do sistema.

---

## Checklist mínimo para produção

- [ ] `DATABASE_URL` – PostgreSQL de produção  
- [ ] `JWT_SECRET` – 32+ caracteres, gerado com `openssl rand -base64 32`  
- [ ] `PORT` – definido pela plataforma ou 8080  
- [ ] `CORS_ORIGINS` – URL(s) do frontend em produção  
- [ ] `APP_PUBLIC_URL` – URL pública do frontend (para links em e-mails/contratos)  
- [ ] (Recomendado) SMTP configurado para envio de e-mails  
- [ ] (Opcional) `DATA_ENCRYPTION_KEYS` e `CURRENT_DATA_KEY_VERSION` com chave gerada por você  
- [ ] (Opcional) Twilio se for usar lembretes por WhatsApp  

---

## Exemplo de .env local (desenvolvimento)

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/prontuario?sslmode=disable
JWT_SECRET=cole_aqui_uma_chave_de_pelo_menos_32_caracteres_aleatorios
PORT=8080
CORS_ORIGINS=http://localhost:5173
APP_PUBLIC_URL=http://localhost:5173
# SMTP (MailHog) – opcional
SMTP_HOST=localhost
SMTP_PORT=1025
```

Não versionar `.env` no Git (mantenha no `.gitignore`).
