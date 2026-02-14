# Plano de Migração – Material UI (MUI)

## Status geral

| Fase | Status | Observação |
|------|--------|------------|
| Etapa 1 – Tema + AppShell + Aparência | **Concluída** | ThemeProvider, ThemeSettingsContext, AppShell, PageContainer, Appearance |
| Etapa 2 – Biblioteca de componentes base | **Concluída** | PageContainer, AppDialog; MUI Table, Button, TextField, Select, Paper, Alert em todas as telas |
| Etapa 3 – Telas prioridade 1 | **Concluída** | Patients, PatientContracts, RecordEntries, ContractTemplates, Agenda, Appearance |
| Etapa 4 – Telas prioridade 2 | **Concluída** | Home, Login, LoginAdmin, SignContract, VerifyContract, ScheduleConfig, Backoffice, BackofficeInvite, ForgotPassword, ResetPassword, RegisterProfessional |
| Etapa 5 – Performance e limpeza | **Concluída** | Lazy-load (React.lazy + Suspense), Layout.tsx removido, .table-responsive removido do CSS |
| Etapa 6 – Qualidade | **Pendente** | TypeScript estrito, ESLint/Prettier, a11y, testes opcionais |

**Documentação:** `docs/MUI_README.md` – uso do tema, AppShell, PageContainer e diretrizes de migração.

---

## 1. Inventário do Projeto

### Stack atual
- **Build:** Vite 5
- **Framework:** React 18 + TypeScript
- **Rotas:** React Router v6
- **Estilos:** CSS global (`index.css`) + estilos inline nos componentes
- **Estado:** Context API (AuthContext, BrandingContext, ThemeSettingsContext) – sem Redux/Zustand
- **UI:** MUI v7 + Emotion (ThemeProvider, CssBaseline, AppShell); ícones `@mui/icons-material`

### Páginas / Rotas principais
| Rota | Página | Proteção | Prioridade |
|------|--------|----------|------------|
| `/` | Home | - | 2 |
| `/login`, `/admin/login` | Login, LoginAdmin | - | 2 |
| `/forgot-password`, `/reset-password` | ForgotPassword, ResetPassword | - | 3 |
| `/register` | RegisterProfessional | - | 3 |
| `/sign-contract` | SignContract | - | 2 |
| `verify/:token` | VerifyContract | - | 2 |
| `/patients` | Patients | PROFESSIONAL, SUPER_ADMIN, LEGAL_GUARDIAN | 1 |
| `/patients/:id/contracts` | PatientContracts | PROFESSIONAL, SUPER_ADMIN | 1 |
| `/patients/:id/record-entries` | RecordEntries | PROFESSIONAL, SUPER_ADMIN, LEGAL_GUARDIAN | 1 |
| `/contract-templates` | ContractTemplates | PROFESSIONAL, SUPER_ADMIN | 1 |
| `/appearance` | Appearance | PROFESSIONAL | 1 ✅ migrada |
| `/schedule-config` | ScheduleConfig | PROFESSIONAL, SUPER_ADMIN | 2 |
| `/agenda` | Agenda | PROFESSIONAL, SUPER_ADMIN | 1 |
| `/backoffice`, `/backoffice/invites` | Backoffice, BackofficeInvite | SUPER_ADMIN | 2 |

### Componentes
- **AppShell** (`components/ui/AppShell.tsx`) – em uso; Drawer + AppBar + Outlet
- **PageContainer** (`components/ui/PageContainer.tsx`) – em uso
- **Layout.tsx** – removido (substituído pelo AppShell)
- **ProtectedRoute.tsx**, **ErrorBoundary.tsx**, **ImpersonateBanner.tsx** – em uso

### Estilos globais
- `index.css`: reset, `.verify-contract-body`, `.sign-contract-page` (mantidos para leitura de contrato e assinatura).

---

## 2. Arquitetura de pastas (UI / Theme)

```
src/
├── theme/
│   ├── index.ts              # createAppTheme, ThemeConfig, exportações
│   ├── palette.ts            # PRIMARY_PRESETS, getPalette
│   ├── typography.ts         # fontes e variantes
│   └── components.ts         # MUI component style overrides + densidade
├── contexts/
│   ├── AuthContext.tsx
│   ├── BrandingContext.tsx
│   └── ThemeSettingsContext.tsx   # modo, cor primária, densidade (localStorage)
├── components/
│   ├── ui/
│   │   ├── AppShell.tsx      # ✅ Drawer + AppBar + conteúdo
│   │   ├── PageContainer.tsx # ✅ Container padrão de página
│   │   ├── buttons/          # (a criar)
│   │   ├── inputs/
│   │   ├── dialogs/
│   │   └── ...
│   # Layout.tsx removido
│   ├── ProtectedRoute.tsx
│   ├── ErrorBoundary.tsx
│   └── ImpersonateBanner.tsx
├── pages/
└── lib/
```

---

## 3. Checklist de migração (por etapas)

### Etapa 1 – Tema + AppShell + Configuração de aparência ✅
- [x] Instalar `@mui/material`, `@mui/icons-material`, `@emotion/react`, `@emotion/styled`
- [x] Criar `theme/` (palette, typography, component overrides)
- [x] Criar `ThemeSettingsContext`: mode (light/dark/system), primaryColorKey (presets), density (comfortable/compact), persistência em localStorage
- [x] Integrar ThemeProvider + CssBaseline no `main.tsx` via ThemeSettingsProvider
- [x] Implementar AppShell MUI (Drawer permanente/overlay no mobile, AppBar, área de conteúdo)
- [x] Painel de Aparência: tema do sistema (light/dark/system, cor primária, densidade) + branding da clínica (API)
- [x] Navegação com MUI List, Menu, IconButton; ícones @mui/icons-material
- [x] Responsividade: Drawer overlay no mobile, ícone hamburger no AppBar
- [x] PageContainer criado e usado em Appearance

### Etapa 2 – Biblioteca de componentes base
- [x] PageContainer
- [x] Dialog padronizado: AppDialog (título, conteúdo, ações)
- [x] Uso de MUI Table, Button, TextField, Alert, IconButton nos fluxos migrados
- [ ] Wrappers opcionais: AppButton (variantes), AppTextField (se necessário)
- [ ] Select, Autocomplete (se necessário)
- [ ] Cards, Tabs (quando migrar telas que usam)
- [ ] Breadcrumbs (onde fizer sentido)
- [ ] Skeletons para estados de loading
- [x] Ícones @mui/icons-material na página Patients

### Etapa 3 – Migração das telas (prioridade 1)
- [x] **Patients** – tabela MUI, modais com AppDialog, PageContainer
- [x] **PatientContracts** – PageContainer, AppDialog (enviar + encerrar), Select, TextField, Paper, Alert
- [x] **RecordEntries** – PageContainer, Paper, RichTextToolbar (MUI Button), contentEditable + form
- [x] **ContractTemplates** – PageContainer, AppDialog, List/ListItem, TextField, assinatura (Paper)
- [x] **Appearance** – MUI Card, Stack, TextField, Button, Radio, ThemeSettings
- [x] **Agenda** – PageContainer, grade semanal (Paper), AppDialog (editar + criar), Select, TextField

### Etapa 4 – Migração das telas (prioridade 2)
- [x] **Home** – PageContainer, Typography, Button (Link)
- [x] **Login**, **LoginAdmin** – Paper, TextField, Button, Link
- [x] **SignContract**, **VerifyContract** – Box, Typography, Select, Checkbox, Paper; classes `.sign-contract-page` e `.verify-contract-body` mantidas no CSS
- [x] **ScheduleConfig** – PageContainer, FormControlLabel/Checkbox, TextField, Select, Paper, Button
- [x] **Backoffice**, **BackofficeInvite** – PageContainer, Table MUI, AppDialog (impersonate), TextField, Select
- [x] **ForgotPassword**, **ResetPassword**, **RegisterProfessional** – Paper, TextField, Button, Link

### Etapa 5 – Performance e limpeza
- [x] Lazy-load de rotas (React.lazy + Suspense) em App.tsx; fallback CircularProgress
- [ ] Revisar re-renders (memo, useMemo, useCallback) em listas grandes (opcional)
- [ ] Imports MUI: caminhos diretos para tree-shaking quando necessário
- [x] Remover `Layout.tsx`
- [x] Remover `.table-responsive` de `index.css` (tabelas migradas para MUI TableContainer)
- [x] Documentação atualizada (MUI_MIGRATION_PLAN, MUI_README)

### Etapa 6 – Qualidade
- [ ] TypeScript estrito; corrigir tipos quebrados
- [ ] ESLint + Prettier alinhados ao projeto
- [ ] A11y: aria-labels onde necessário, foco visível, contraste (MUI já contribui)
- [ ] Smoke tests nas telas críticas (opcional)

---

## 4. Diretrizes de uso pós-migração

- **Tema:** Usar `useTheme()` e `sx` com tokens (ex.: `color: 'primary.main'`, `bgcolor: 'background.paper'`).
- **Novos componentes:** Criar em `components/ui/` encapsulando MUI com o design do produto.
- **Ícones:** Apenas `@mui/icons-material`.
- **Evitar:** Misturar outras libs de UI; evitar estilos inline que dupliquem o tema.

---

## 5. Artefatos esperados (entregas)

| Entrega | Conteúdo |
|---------|----------|
| **1) Tema + AppShell** | ThemeProvider, tema centralizado, ThemeSettingsContext, AppShell MUI, persistência de aparência | ✅ |
| **2) Biblioteca base** | PageContainer, AppDialog; MUI usado em todas as telas | ✅ |
| **3) Telas migradas** | Todas as páginas usando MUI + PageContainer ou Paper; fluxos intactos | ✅ |
| **4) Limpeza** | Layout.tsx removido; .table-responsive removido; documentação atualizada | ✅ |
| **Performance** | Lazy load (Suspense + React.lazy) aplicado às rotas | ✅ |

---

## 6. Restrições

- Não quebrar fluxos existentes (login, contratos, agenda, pacientes, etc.).
- Não introduzir novas libs de UI além do MUI sem justificativa.
- Manter compatibilidade com navegadores alvo do projeto.
- Manter branding da clínica (API) funcionando junto ao tema MUI (cor do drawer, home label, etc.).

---

## 7. Notas de performance

- MUI v7 usa Emotion; evitar duplicar com styled-components.
- Imports: `@mui/material/Button` etc. quando quiser garantir tree-shaking.
- Listas muito grandes: considerar MUI DataGrid ou virtualização (react-window).
- Medir bundle antes/depois (ex.: `vite build` e tamanho dos chunks).

---

## 8. Próximos passos (opcional)

1. **Etapa 6:** TypeScript estrito, ESLint/Prettier, a11y (aria-labels, foco), testes.
2. **Opcional:** Wrappers em `components/ui/` (AppButton, AppTextField) se quiser padronizar variantes.
3. **Opcional:** Skeletons para loading, Breadcrumbs onde fizer sentido.
