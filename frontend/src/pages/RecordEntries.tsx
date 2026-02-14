import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Alert, Box, Button, FormControl, InputLabel, MenuItem, Paper, Select, Typography } from '@mui/material'
import FormatBoldIcon from '@mui/icons-material/FormatBold'
import FormatItalicIcon from '@mui/icons-material/FormatItalic'
import FormatListBulletedIcon from '@mui/icons-material/FormatListBulleted'
import FormatUnderlinedIcon from '@mui/icons-material/FormatUnderlined'
import FormatStrikethroughIcon from '@mui/icons-material/FormatStrikethrough'
import FormatListNumberedIcon from '@mui/icons-material/FormatListNumbered'
import { useAuth } from '../contexts/AuthContext'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import TextStyle from '@tiptap/extension-text-style'
import Placeholder from '@tiptap/extension-placeholder'
import { Extension, type CommandProps } from '@tiptap/core'

const MESES = ['janeiro', 'fevereiro', 'março', 'abril', 'maio', 'junho', 'julho', 'agosto', 'setembro', 'outubro', 'novembro', 'dezembro']

function formatarAtendimento(createdAt: string): string {
  try {
    const d = new Date(createdAt)
    if (Number.isNaN(d.getTime())) return createdAt
    const dia = d.getDate()
    const mes = MESES[d.getMonth()]
    const ano = d.getFullYear()
    const h = d.getHours().toString().padStart(2, '0')
    const min = d.getMinutes().toString().padStart(2, '0')
    return `Atendimento realizado em ${dia} de ${mes} de ${ano}, às ${h}:${min}`
  } catch {
    return createdAt
  }
}

function isHtml(content: string): boolean {
  return /<[a-z][\s\S]*>/i.test(content)
}

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    fontSize: {
      setFontSize: (fontSize: string) => ReturnType
      unsetFontSize: () => ReturnType
    }
  }
}

const FontSize = Extension.create({
  name: 'fontSize',
  addGlobalAttributes() {
    return [
      {
        types: ['textStyle'],
        attributes: {
          fontSize: {
            default: null,
            parseHTML: (element) => element.style.fontSize || null,
            renderHTML: (attributes) => {
              if (!attributes.fontSize) return {}
              return { style: `font-size: ${attributes.fontSize}` }
            },
          },
        },
      },
    ]
  },
  addCommands() {
    return {
      setFontSize:
        (fontSize: string) =>
        ({ chain }: CommandProps) =>
          chain().setMark('textStyle', { fontSize }).run(),
      unsetFontSize:
        () =>
        ({ chain }: CommandProps) =>
          chain().setMark('textStyle', { fontSize: null }).removeEmptyTextStyle().run(),
    }
  },
})

function RichTextToolbar({
  editor,
  fontSize,
  setFontSize,
}: {
  editor: ReturnType<typeof useEditor>
  fontSize: string
  setFontSize: (v: string) => void
}) {
  if (!editor) return null

  const FONT_SIZES = [
    { value: '', label: 'Tamanho' },
    { value: '12px', label: '12' },
    { value: '14px', label: '14' },
    { value: '16px', label: '16' },
    { value: '18px', label: '18' },
    { value: '20px', label: '20' },
    { value: '24px', label: '24' },
  ]

  return (
    <Box sx={{ display: 'flex', gap: 0.5, mb: 0.75, flexWrap: 'wrap', alignItems: 'center' }}>
      <Button
        size="small"
        variant={editor.isActive('bold') ? 'contained' : 'outlined'}
        onClick={() => editor.chain().focus().toggleBold().run()}
        title="Negrito"
        sx={{ minWidth: 'auto', px: 0.75 }}
      >
        <FormatBoldIcon fontSize="small" />
      </Button>
      <Button
        size="small"
        variant={editor.isActive('italic') ? 'contained' : 'outlined'}
        onClick={() => editor.chain().focus().toggleItalic().run()}
        title="Itálico"
        sx={{ minWidth: 'auto', px: 0.75 }}
      >
        <FormatItalicIcon fontSize="small" />
      </Button>
      <Button
        size="small"
        variant={editor.isActive('underline') ? 'contained' : 'outlined'}
        onClick={() => editor.chain().focus().toggleUnderline().run()}
        title="Sublinhado"
        sx={{ minWidth: 'auto', px: 0.75 }}
      >
        <FormatUnderlinedIcon fontSize="small" />
      </Button>
      <Button
        size="small"
        variant={editor.isActive('strike') ? 'contained' : 'outlined'}
        onClick={() => editor.chain().focus().toggleStrike().run()}
        title="Tachado"
        sx={{ minWidth: 'auto', px: 0.75 }}
      >
        <FormatStrikethroughIcon fontSize="small" />
      </Button>
      <Button
        size="small"
        variant={editor.isActive('bulletList') ? 'contained' : 'outlined'}
        onClick={() => editor.chain().focus().toggleBulletList().run()}
        title="Lista com marcadores"
        sx={{ minWidth: 'auto', px: 0.75 }}
      >
        <FormatListBulletedIcon fontSize="small" />
      </Button>
      <Button
        size="small"
        variant={editor.isActive('orderedList') ? 'contained' : 'outlined'}
        onClick={() => editor.chain().focus().toggleOrderedList().run()}
        title="Lista numerada"
        sx={{ minWidth: 'auto', px: 0.75 }}
      >
        <FormatListNumberedIcon fontSize="small" />
      </Button>

      <FormControl size="small" sx={{ minWidth: 120 }}>
        <InputLabel>Tamanho</InputLabel>
        <Select
          value={fontSize}
          label="Tamanho"
          onChange={(e) => {
            const v = String(e.target.value || '')
            setFontSize(v)
            if (!v) editor.chain().focus().unsetFontSize().run()
            else editor.chain().focus().setFontSize(v).run()
          }}
        >
          {FONT_SIZES.map((o) => (
            <MenuItem key={o.value} value={o.value}>
              {o.label}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
    </Box>
  )
}

export function RecordEntries() {
  const { patientId } = useParams<{ patientId: string }>()
  const { user } = useAuth()
  const canManageContracts = user?.role === 'PROFESSIONAL' || user?.role === 'SUPER_ADMIN'
  const [entries, setEntries] = useState<{ id: string; content: string; entry_date: string; author_type: string; created_at: string }[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [hasContent, setHasContent] = useState(false)
  const [fontSize, setFontSize] = useState('')
  const editorBoxRef = useRef<HTMLDivElement | null>(null)

  const editor = useEditor({
    extensions: [
      StarterKit,
      Underline,
      TextStyle,
      FontSize,
      Placeholder.configure({
        placeholder: 'Conteúdo da anotação... (use a barra acima)',
      }),
    ],
    content: '',
    onUpdate: ({ editor }) => {
      setHasContent(editor.getText().trim() !== '')
    },
    editorProps: {
      attributes: {
        class: 'record-entry-editor',
      },
    },
  })

  const load = useCallback(() => {
    if (!patientId) return
    setLoading(true)
    api
      .listRecordEntries(patientId)
      .then((r) => setEntries(r.entries))
      .catch(() => setError('Sem permissão ou falha ao carregar.'))
      .finally(() => setLoading(false))
  }, [patientId])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    // sem efeitos colaterais
  }, [entries])

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!patientId) return
    const html = editor?.getHTML() ?? ''
    const text = (editor?.getText() ?? '').trim()
    if (!text) return
    setSubmitting(true)
    setError('')
    try {
      await api.createRecordEntry(patientId, html.trim())
      editor?.commands.clearContent()
      setFontSize('')
      load()
    } catch {
      setError('Falha ao criar entrada.')
    } finally {
      setSubmitting(false)
    }
  }

  if (!patientId) return null

  return (
    <PageContainer>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, alignItems: 'center', mb: 2 }}>
        <Typography component={Link} to="/patients" sx={{ color: 'primary.main', textDecoration: 'none' }}>← Pacientes</Typography>
        {canManageContracts && (
          <>
            <Typography component="span" color="text.secondary">·</Typography>
            <Typography component={Link} to={`/patients/${patientId}/contracts`} sx={{ color: 'primary.main', textDecoration: 'none' }}>Gerenciar contratos</Typography>
          </>
        )}
      </Box>
      <Typography variant="h4" sx={{ mb: 2 }}>Prontuário</Typography>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {loading && <Typography color="text.secondary">Carregando...</Typography>}
      {!loading && (
        <>
          <Box component="form" onSubmit={handleAdd} sx={{ mb: 2 }}>
            <Typography variant="subtitle2" sx={{ mb: 0.5 }}>Nova entrada</Typography>
            <RichTextToolbar editor={editor} fontSize={fontSize} setFontSize={setFontSize} />
            <Box
              ref={editorBoxRef}
              sx={{
                width: '100%',
                maxWidth: 560,
                minHeight: 140,
                p: 1,
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 1,
                fontSize: 14,
                '& .record-entry-editor': {
                  minHeight: 110,
                },
                '& .ProseMirror p': {
                  margin: 0,
                },
              }}
            >
              <EditorContent editor={editor} />
            </Box>
            <Button type="submit" variant="contained" disabled={submitting || !hasContent} sx={{ mt: 0.5 }}>
              {submitting ? 'Salvando...' : 'Adicionar'}
            </Button>
          </Box>

          <Box component="ul" sx={{ listStyle: 'none', p: 0, mb: 3 }}>
            {entries.map((e) => (
              <Paper key={e.id} variant="outlined" sx={{ p: 2, mb: 0.75 }}>
                <Typography variant="caption" color="text.secondary" sx={{ mb: 0.25, display: 'block' }}>
                  {formatarAtendimento(e.created_at)}
                </Typography>
                {isHtml(e.content) ? (
                  <Box className="record-entry-content" sx={{ whiteSpace: 'pre-wrap' }} dangerouslySetInnerHTML={{ __html: e.content }} />
                ) : (
                  <Typography className="record-entry-content" component="span" sx={{ whiteSpace: 'pre-wrap' }}>
                    {e.content}
                  </Typography>
                )}
              </Paper>
            ))}
            {entries.length === 0 && <Typography color="text.secondary">Nenhuma entrada ainda.</Typography>}
          </Box>
        </>
      )}
    </PageContainer>
  )
}
