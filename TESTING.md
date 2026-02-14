## Como rodar os testes

### Frontend (unit + integração via MSW)
Dentro de `frontend/`:

- Rodar testes:

```bash
npm test -- --run
```

- Rodar coverage:

```bash
npm run test:coverage
```

Testes adicionados:
- `src/lib/cpf.test.ts` (validação DV + normalização)
- `src/components/ProtectedRoute.test.tsx` (redirect quando não autenticado)
- `src/pages/Patients.test.tsx` (validações de regras no modal “Novo paciente”)
- `src/lib/api.integration.test.ts` (integração leve usando MSW)

### Backend (unit)
Dentro de `backend/`:

```bash
go test ./internal/crypto ./internal/auth ./internal/api
```

Testes adicionados:
- `internal/api/validation_test.go`
- `internal/api/format_date_test.go`
- `internal/api/contract_fill_test.go`

### Backend (integração)
Requer `DATABASE_URL` apontando para um Postgres com as migrations aplicáveis.

- Subir infraestrutura local (na raiz do repo):

```bash
docker-compose up -d postgres
```

- Rodar testes de integração (build tag):

```bash
cd backend
export DATABASE_URL="postgres://prontuario:prontuario_secret@localhost:5432/prontuario?sslmode=disable"
go test -tags=integration ./internal/api -run TestIntegration_
```

Testes de integração adicionados:
- `internal/api/patients_integration_test.go` (isolamento multi-tenant e CPF do paciente único por clínica)
- `internal/api/integration_smoke_test.go` (smoke de /health)

## Critério de 80% (regras de negócio)
A métrica principal é o checklist em `TEST_RULES_CHECKLIST.md`:\n- Marcar **[x]** (unit) ou **[X]** (integration).\n- Meta: **≥80%** das regras-alvo marcadas como cobertas.

