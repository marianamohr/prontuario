## Checklist de regras de negócio (meta: ≥80% cobertas por testes)

Legenda:
- [ ] não coberto
- [x] coberto por **unit test**
- [X] coberto por **integration test**

### Autenticação e autorização
- [ ] PROFESSIONAL autentica via email/senha (senha hash, nunca plaintext).
- [ ] SUPER_ADMIN autentica via email/senha.
- [ ] LEGAL_GUARDIAN autentica via LOCAL/GOOGLE/HYBRID conforme cadastro.
- [ ] Usuário `status=CANCELLED` não autentica.
- [ ] Rotas protegidas retornam 403 para role incorreta.

### Multi-tenant e isolamento
- [ ] `clinic_id` nunca vem do request; vem do JWT (quando aplicável).
- [ ] PROFESSIONAL só acessa dados do próprio `clinic_id` (pacientes, contratos, agenda, prontuário).
- [ ] SUPER_ADMIN ignora tenant (acessos globais permitidos).
- [X] Isolation: dados criados em clínica A não aparecem em listagens da clínica B.

### Impersonate
- [ ] Iniciar impersonate cria sessão e marca `is_impersonated=true`.
- [ ] Encerrar impersonate restaura sessão do admin.
- [ ] Impersonate expira (TTL) e passa a bloquear ações.
- [ ] Eventos de impersonate registram audit/access logs.

### Pacientes e responsáveis legais
- [ ] Criar paciente sem guardião: `full_name` obrigatório.
- [x] Criar/editar paciente com guardião: valida email via regex.
- [x] Guardião: CPF obrigatório e formato válido (11 dígitos + DV no front).
- [x] Guardião: endereço deve conter Rua/Bairro/Cidade/Estado/País/CEP e CEP tem 8 dígitos.
- [ ] Guardião: birth_date obrigatório quando email preenchido.
- [ ] Paciente: birth_date obrigatório quando fluxo com guardião.
- [X] Paciente: CPF opcional, criptografado em repouso e único por clínica quando preenchido.
- [x] Editar paciente: CPF do guardião aparece mascarado com toggle.

### Soft delete
- [ ] Soft delete de paciente remove de listagens para PROFESSIONAL.
- [ ] Soft delete de guardião remove de listagens para PROFESSIONAL.
- [ ] Soft delete de contrato remove de listagens para PROFESSIONAL.

### Permissões granulares do guardião
- [ ] `can_view_medical_record=false` → acesso a prontuário retorna 403.
- [ ] `can_view_contracts=false` → acesso a contratos retorna 403.

### Prontuário
- [ ] Criar entrada exige permissão.
- [ ] Conteúdo de prontuário é criptografado em repouso (AES-256-GCM) e descriptografado no read.
- [ ] Leitura de prontuário cria access_log (VIEW/READ) sem PII.
- [ ] Editor aceita HTML (WYSIWYG) e renderiza listas corretamente.

### Contratos
- [ ] Enviar contrato gera token/link e envia email.
- [ ] Assinatura exige `accepted_terms=true`.
- [ ] Ao assinar: status=SIGNED, `signed_at`, `pdf_sha256`, `verification_token`.
- [ ] Página `/verify/:token` valida e mostra hash/metadata.
- [ ] Cancelar contrato torna inelegível (status=CANCELLED) e notifica.
- [ ] Encerrar contrato (status=ENDED) respeita data e notifica.
- [ ] Ao assinar, outros contratos pendentes do mesmo paciente/guardião são cancelados.

### Placeholders/templating (contrato)
- [x] `[DATA]` é substituído no template em DD/MM/AAAA.
- [x] `[LOCAL]` não é substituído (local deve estar no template).

### Agenda
- [ ] Criação de agendamentos vinculados a contrato preenche `contract_id`.
- [ ] Encerrar contrato cancela agendamentos **após** a data de encerramento (exclusivo).
- [ ] Listagem de agenda não mostra `CANCELLED` nem `SERIES_ENDED`.

### Auditoria e erros
- [ ] Timeline unificada (`audit_events` + `access_logs`) filtra por `request_id` e severity.
- [ ] Front envia erros de `catch()` para `/api/errors/frontend` sem PII.
- [ ] Backend recover middleware retorna 500 genérico e loga stack com `request_id`.

