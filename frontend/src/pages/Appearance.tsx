import { useState, useEffect } from 'react'
import {
  Box,
  Typography,
  TextField,
  Button,
  FormControlLabel,
  Radio,
  RadioGroup,
  FormControl,
  FormLabel,
  Card,
  CardContent,
  Stack,
  InputAdornment,
  Alert,
} from '@mui/material'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'
import { useThemeSettings } from '../contexts/ThemeSettingsContext'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'

export function Appearance() {
  const { user } = useAuth()
  const ctx = useBranding()
  const branding = ctx?.branding ?? null
  const refetch = ctx?.refetch ?? (async () => {})
  const { settings, setMode, setPrimaryColorKey, setDensity, primaryPresets } = useThemeSettings()
  const [primaryColor, setPrimaryColor] = useState('')
  const [backgroundColor, setBackgroundColor] = useState('')
  const [homeLabel, setHomeLabel] = useState('')
  const [homeImageUrl, setHomeImageUrl] = useState('')
  const [actionButtonColor, setActionButtonColor] = useState('')
  const [negationButtonColor, setNegationButtonColor] = useState('')
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (branding) {
      setPrimaryColor(branding.primary_color ?? '')
      setBackgroundColor(branding.background_color ?? '')
      setHomeLabel(branding.home_label ?? '')
      setHomeImageUrl(branding.home_image_url ?? '')
      setActionButtonColor(branding.action_button_color ?? '')
      setNegationButtonColor(branding.negation_button_color ?? '')
    }
  }, [branding])

  if (user?.role !== 'PROFESSIONAL') {
    return (
      <PageContainer>
        <Typography>Apenas profissionais podem configurar a aparência da clínica.</Typography>
      </PageContainer>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setMessage('')
    try {
      await api.updateBranding({
        primary_color: primaryColor.trim() || null,
        background_color: backgroundColor.trim() || null,
        home_label: homeLabel.trim() || null,
        home_image_url: homeImageUrl.trim() || null,
        action_button_color: actionButtonColor.trim() || null,
        negation_button_color: negationButtonColor.trim() || null,
      })
      await refetch()
      setMessage('Aparência salva. A página já está com as novas cores.')
    } catch {
      setMessage('Falha ao salvar.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageContainer>
      <Typography variant="h1" sx={{ mb: 2 }}>
        Aparência
      </Typography>

      <Stack spacing={3}>
        <Card>
          <CardContent>
            <Typography variant="h2" gutterBottom sx={{ fontSize: '1.125rem' }}>
              Tema do sistema
            </Typography>
            <Typography color="text.secondary" sx={{ mb: 2 }}>
              Modo claro/escuro, cor primária e densidade da interface. As alterações são salvas automaticamente.
            </Typography>
            <Stack spacing={2}>
              <FormControl component="fieldset">
                <FormLabel component="legend">Modo</FormLabel>
                <RadioGroup row value={settings.mode} onChange={(_, v) => setMode(v as 'light' | 'dark' | 'system')}>
                  <FormControlLabel value="light" control={<Radio />} label="Claro" />
                  <FormControlLabel value="dark" control={<Radio />} label="Escuro" />
                  <FormControlLabel value="system" control={<Radio />} label="Sistema" />
                </RadioGroup>
              </FormControl>
              <FormControl component="fieldset">
                <FormLabel component="legend">Cor primária</FormLabel>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 0.5 }}>
                  {Object.entries(primaryPresets).map(([key, pal]) => (
                    <Button
                      key={key}
                      variant={settings.primaryColorKey === key ? 'contained' : 'outlined'}
                      size="small"
                      onClick={() => setPrimaryColorKey(key)}
                      sx={{
                        minWidth: 0,
                        bgcolor: settings.primaryColorKey === key ? undefined : pal.main,
                        borderColor: pal.main,
                        '&:hover': { borderColor: pal.main, bgcolor: settings.primaryColorKey === key ? undefined : pal.light ?? pal.main },
                      }}
                    >
                      {key}
                    </Button>
                  ))}
                </Box>
              </FormControl>
              <FormControl component="fieldset">
                <FormLabel component="legend">Densidade</FormLabel>
                <RadioGroup row value={settings.density} onChange={(_, v) => setDensity(v as 'comfortable' | 'compact')}>
                  <FormControlLabel value="comfortable" control={<Radio />} label="Confortável" />
                  <FormControlLabel value="compact" control={<Radio />} label="Compacto" />
                </RadioGroup>
              </FormControl>
            </Stack>
          </CardContent>
        </Card>

        <Card>
          <CardContent>
            <Typography variant="h2" gutterBottom sx={{ fontSize: '1.125rem' }}>
              White label (clínica)
            </Typography>
            <Typography color="text.secondary" sx={{ mb: 2 }}>
              Personalize as cores e o botão Home para que o sistema tenha a cara da sua clínica.
            </Typography>
            <form onSubmit={handleSubmit}>
              <Stack spacing={2} sx={{ maxWidth: 420 }}>
                <TextField
                  label="Cor principal (cabeçalho e botões)"
                  value={primaryColor || '#1a1a2e'}
                  onChange={(e) => setPrimaryColor(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <input
                          type="color"
                          value={primaryColor || '#1a1a2e'}
                          onChange={(e) => setPrimaryColor(e.target.value)}
                          style={{ width: 28, height: 28, border: 'none', borderRadius: 4, cursor: 'pointer', padding: 0 }}
                        />
                      </InputAdornment>
                    ),
                  }}
                />
                <TextField
                  label="Cor de fundo da área logada"
                  value={backgroundColor || '#ffffff'}
                  onChange={(e) => setBackgroundColor(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <input
                          type="color"
                          value={backgroundColor || '#ffffff'}
                          onChange={(e) => setBackgroundColor(e.target.value)}
                          style={{ width: 28, height: 28, border: 'none', borderRadius: 4, cursor: 'pointer', padding: 0 }}
                        />
                      </InputAdornment>
                    ),
                  }}
                />
                <TextField label="Texto do botão Home" value={homeLabel} onChange={(e) => setHomeLabel(e.target.value)} placeholder="Ex.: Minha Clínica" fullWidth />
                <TextField label="URL da imagem/logo do botão Home (opcional)" value={homeImageUrl} onChange={(e) => setHomeImageUrl(e.target.value)} placeholder="https://..." fullWidth />
                <TextField
                  label="Cor dos botões de ação"
                  value={actionButtonColor || '#16a34a'}
                  onChange={(e) => setActionButtonColor(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <input type="color" value={actionButtonColor || '#16a34a'} onChange={(e) => setActionButtonColor(e.target.value)} style={{ width: 28, height: 28, border: 'none', borderRadius: 4, cursor: 'pointer', padding: 0 }} />
                      </InputAdornment>
                    ),
                  }}
                />
                <TextField
                  label="Cor dos botões de negação"
                  value={negationButtonColor || '#dc2626'}
                  onChange={(e) => setNegationButtonColor(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <input type="color" value={negationButtonColor || '#dc2626'} onChange={(e) => setNegationButtonColor(e.target.value)} style={{ width: 28, height: 28, border: 'none', borderRadius: 4, cursor: 'pointer', padding: 0 }} />
                      </InputAdornment>
                    ),
                  }}
                />
                {message && (
                  <Alert severity={message.includes('Falha') ? 'error' : 'success'} onClose={() => setMessage('')}>
                    {message}
                  </Alert>
                )}
                <Button type="submit" variant="contained" disabled={saving}>
                  {saving ? 'Salvando...' : 'Salvar aparência da clínica'}
                </Button>
              </Stack>
            </form>
          </CardContent>
        </Card>
      </Stack>
    </PageContainer>
  )
}
