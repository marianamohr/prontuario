# Material UI – Uso no Prontuário

## Tema e aparência

- O tema é criado em `src/theme/` e aplicado via `ThemeSettingsProvider` em `main.tsx`.
- **Configurações de tema** (modo claro/escuro, cor primária, densidade) ficam em **Aparência** (menu do usuário → Aparência) e são salvas em `localStorage` (`prontuario-theme-settings`).
- **Branding da clínica** (cor do cabeçalho, logo, cores de botões) continua na mesma página e é carregado pela API (useBranding).

### Tokens

- **Paleta:** `src/theme/palette.ts` – presets de cor primária (`PRIMARY_PRESETS`) e `getPalette(mode, primaryKey)`.
- **Tipografia:** `src/theme/typography.ts` – fonte DM Sans, variantes h1–h6, body1/2, button.
- **Componentes:** `src/theme/components.ts` – overrides globais (Button, TextField, Paper, etc.) e densidade (comfortable/compact).

### Uso em componentes

```tsx
import { useTheme } from '@mui/material/styles'
import { Box, Button } from '@mui/material'

function MyComponent() {
  const theme = useTheme()
  return (
    <Box sx={{ color: 'primary.main', p: 2 }}>
      <Button variant="contained">Salvar</Button>
    </Box>
  )
}
```

- Preferir `sx` com tokens do tema (`primary.main`, `text.secondary`, `background.paper`, etc.).
- Espaçamento: usar `theme.spacing(n)` ou no `sx` com números (ex.: `p: 2` = 16px com spacing 8).

## Layout (AppShell)

- **AppShell** (`src/components/ui/AppShell.tsx`): Drawer (menu lateral) + AppBar (topo) + área de conteúdo com `<Outlet />`.
- No mobile o Drawer vira temporário (overlay) e o AppBar ganha ícone de menu.
- O menu lateral usa a cor primária do branding da clínica (quando existir) ou a cor primária do tema MUI.

## Componentes do produto

- **PageContainer** (`src/components/ui/PageContainer.tsx`): wrapper de página com padding e `maxWidth` opcional. Use em todas as telas internas.
- **AppDialog** (`src/components/ui/AppDialog.tsx`): modal padrão com `title`, `children` (conteúdo), `actions` (botões no rodapé), `open`, `onClose`, `maxWidth` (ex.: `"sm"`).
- Novos componentes de produto devem ficar em `src/components/ui/` e usar MUI por baixo. Preferir `@mui/icons-material` para ícones.

## Migração de telas

1. Envolver o conteúdo da página em `<PageContainer>`.
2. Trocar elementos nativos por componentes MUI (Typography, Button, TextField, Card, etc.).
3. Trocar estilos inline por `sx` ou `styled()` quando fizer sentido.
4. Remover classes CSS que foram substituídas pelo tema/componentes.

## Performance

- Importar componentes MUI pelo caminho direto quando quiser reduzir bundle (ex.: `import Button from '@mui/material/Button'`). O Vite já faz tree-shaking do barrel.
- Para rotas pesadas, usar `React.lazy()` e `<Suspense>` no router (ver plano em `docs/MUI_MIGRATION_PLAN.md`).
