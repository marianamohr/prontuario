import { useEffect, useState } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import {
  Box,
  Drawer,
  AppBar,
  Toolbar,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  useTheme,
  useMediaQuery,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import HomeIcon from '@mui/icons-material/Home'
import PeopleIcon from '@mui/icons-material/People'
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth'
import SettingsIcon from '@mui/icons-material/Settings'
import MenuIcon from '@mui/icons-material/Menu'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import DescriptionIcon from '@mui/icons-material/Description'
import PaletteIcon from '@mui/icons-material/Palette'
import ScheduleIcon from '@mui/icons-material/Schedule'
import PersonIcon from '@mui/icons-material/Person'
import TimelineIcon from '@mui/icons-material/Timeline'
import BugReportIcon from '@mui/icons-material/BugReport'
import { ImpersonateBanner } from '../ImpersonateBanner'
import { useAuth } from '../../contexts/AuthContext'
import { useBranding } from '../../contexts/BrandingContext'

const DRAWER_WIDTH = 260
const DRAWER_WIDTH_COLLAPSED = 64

const navItems: { to: string; end?: boolean; label: string; icon: React.ReactNode; roles?: string[] }[] = [
  { to: '/home', end: true, label: 'Página inicial', icon: <HomeIcon /> },
  { to: '/patients', label: 'Pacientes', icon: <PeopleIcon />, roles: ['PROFESSIONAL', 'LEGAL_GUARDIAN'] },
  { to: '/agenda', label: 'Agenda', icon: <CalendarMonthIcon />, roles: ['PROFESSIONAL', 'SUPER_ADMIN'] },
  // `end: true` evita que /backoffice fique ativo em /backoffice/*
  { to: '/backoffice', end: true, label: 'Usuários', icon: <SettingsIcon />, roles: ['SUPER_ADMIN'] },
  { to: '/backoffice/audit', end: true, label: 'Auditoria', icon: <TimelineIcon />, roles: ['SUPER_ADMIN'] },
  { to: '/backoffice/errors', end: true, label: 'Erros', icon: <BugReportIcon />, roles: ['SUPER_ADMIN'] },
]

const guestNavItems = [
  { to: '/login', label: 'Entrar', icon: <PeopleIcon /> },
]

export function AppShell() {
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))
  const { user, logout, isImpersonated } = useAuth()
  const branding = useBranding()?.branding ?? null
  const navigate = useNavigate()
  const location = useLocation()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(false)
  const [userMenuAnchor, setUserMenuAnchor] = useState<null | HTMLElement>(null)

  const isProfessional = user?.role === 'PROFESSIONAL'
  const drawerBg = isProfessional && branding?.primary_color ? branding.primary_color : theme.palette.primary.main
  const homeLabel = isProfessional && branding?.home_label ? branding.home_label : 'CamiHealth'
  const homeImageUrl = isProfessional ? branding?.home_image_url : null

  useEffect(() => {
    if (!isMobile) setDrawerOpen(false)
  }, [isMobile])

  useEffect(() => {
    setUserMenuAnchor(null)
  }, [location.pathname])

  const handleLogout = () => {
    setUserMenuAnchor(null)
    logout()
    if (isImpersonated) {
      localStorage.removeItem('impersonating')
    }
    navigate('/login', { replace: true })
  }

  const drawerWidth = isMobile ? DRAWER_WIDTH : (collapsed ? DRAWER_WIDTH_COLLAPSED : DRAWER_WIDTH)
  const isSuperAdminWithoutImpersonate = user?.role === 'SUPER_ADMIN' && !isImpersonated
  const showNavItems = user
    ? navItems.filter((item) => {
        if (isSuperAdminWithoutImpersonate && (item.to === '/' || item.to === '/agenda')) return false
        return !item.roles || item.roles.includes(user.role)
      })
    : guestNavItems

  const drawerContent = (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', bgcolor: drawerBg, color: '#fff' }}>
      <Toolbar
        disableGutters
        sx={{
          px: 1.5,
          minHeight: 56,
          justifyContent: isMobile ? 'space-between' : (collapsed ? 'center' : 'space-between'),
        }}
      >
        {(!collapsed || isMobile) && (
          <Link
            to="/home"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              color: 'inherit',
              textDecoration: 'none',
              fontWeight: 600,
              fontSize: '1rem',
              minWidth: 0,
            }}
          >
            {homeImageUrl ? (
              <img src={homeImageUrl} alt="" loading="lazy" style={{ height: 28, maxWidth: 100, objectFit: 'contain' }} />
            ) : (
              <HomeIcon />
            )}
            {!collapsed && <span style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{homeLabel}</span>}
          </Link>
        )}
        {!isMobile && (
          <IconButton size="small" onClick={() => setCollapsed(!collapsed)} sx={{ color: 'inherit' }} aria-label={collapsed ? 'Expandir menu' : 'Recolher menu'}>
            {collapsed ? <ChevronRightIcon /> : <ChevronLeftIcon />}
          </IconButton>
        )}
      </Toolbar>
      <List sx={{ flex: 1, px: 1, py: 0 }}>
        {showNavItems.map((item) => (
          <ListItemButton
            key={item.to}
            component={NavLink}
            to={item.to}
            end={'end' in item && item.end}
            sx={{
              borderRadius: 1,
              mb: 0.25,
              color: 'rgba(255,255,255,0.9)',
              '&.active': { bgcolor: 'rgba(255,255,255,0.15)', color: '#fff' },
              '&:hover': { bgcolor: 'rgba(255,255,255,0.08)' },
              '&.Mui-focusVisible': { outline: '2px solid rgba(255,255,255,0.35)', outlineOffset: '2px' },
            }}
          >
            <ListItemIcon sx={{ minWidth: 40, color: 'inherit' }}>{item.icon}</ListItemIcon>
            {(!collapsed || isMobile) && <ListItemText primary={item.label} primaryTypographyProps={{ fontSize: '0.95rem' }} />}
          </ListItemButton>
        ))}
      </List>
    </Box>
  )

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: 'background.default' }}>
      <ImpersonateBanner />
      <AppBar position="static" elevation={0} sx={{ bgcolor: 'background.paper', color: 'text.primary', borderBottom: 1, borderColor: 'divider' }}>
        <Toolbar disableGutters sx={{ px: { xs: 1.5, md: 2 }, minHeight: 56 }}>
          {isMobile && (
            <IconButton edge="start" onClick={() => setDrawerOpen(true)} aria-label="Abrir menu" sx={{ mr: 1 }}>
              <MenuIcon />
            </IconButton>
          )}
          <Box sx={{ flex: 1 }} />
          {user && (
            <>
              <IconButton
                onClick={(e) => setUserMenuAnchor(e.currentTarget)}
                size="small"
                sx={{
                  bgcolor: 'primary.main',
                  color: 'primary.contrastText',
                  '&:hover': { bgcolor: 'primary.dark' },
                }}
                aria-label="Menu do usuário"
              >
                <Typography variant="body2" fontWeight={600}>
                  {(user.full_name?.trim()?.[0] || user.email?.trim()?.[0] || '?').toUpperCase()}
                </Typography>
              </IconButton>
              <Menu
                anchorEl={userMenuAnchor}
                open={Boolean(userMenuAnchor)}
                onClose={() => setUserMenuAnchor(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
                slotProps={{ paper: { sx: { minWidth: 220 } } }}
              >
                {user.role === 'PROFESSIONAL' && (
                  <>
                    <MenuItem component={Link} to="/contract-templates" onClick={() => setUserMenuAnchor(null)}>
                      <ListItemIcon><DescriptionIcon fontSize="small" /></ListItemIcon>
                      Configurar contrato
                    </MenuItem>
                    <MenuItem component={Link} to="/schedule-config" onClick={() => setUserMenuAnchor(null)}>
                      <ListItemIcon><ScheduleIcon fontSize="small" /></ListItemIcon>
                      Configurar agenda
                    </MenuItem>
                  </>
                )}
                {(user.role === 'PROFESSIONAL' || user.role === 'SUPER_ADMIN') && (
                  <MenuItem component={Link} to="/profile" onClick={() => setUserMenuAnchor(null)}>
                    <ListItemIcon><PersonIcon fontSize="small" /></ListItemIcon>
                    Editar perfil
                  </MenuItem>
                )}
                {user.role === 'PROFESSIONAL' && (
                  <MenuItem component={Link} to="/appearance" onClick={() => setUserMenuAnchor(null)}>
                    <ListItemIcon><PaletteIcon fontSize="small" /></ListItemIcon>
                    Aparência
                  </MenuItem>
                )}
                <MenuItem disabled sx={{ fontSize: 11, color: 'text.secondary' }}>
                  ID: {user.id}
                </MenuItem>
                <MenuItem onClick={handleLogout}>Sair</MenuItem>
              </Menu>
            </>
          )}
          {!user && (
            <Typography variant="body2" color="text.secondary">
              Não conectado
            </Typography>
          )}
        </Toolbar>
      </AppBar>
      <Box component="main" sx={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {isMobile ? (
          <Drawer
            variant="temporary"
            open={drawerOpen}
            onClose={() => setDrawerOpen(false)}
            ModalProps={{ keepMounted: true }}
            sx={{ '& .MuiDrawer-paper': { width: DRAWER_WIDTH, boxSizing: 'border-box' } }}
          >
            {drawerContent}
          </Drawer>
        ) : (
          <Drawer
            variant="permanent"
            open
            sx={{
              width: drawerWidth,
              flexShrink: 0,
              '& .MuiDrawer-paper': {
                width: drawerWidth,
                boxSizing: 'border-box',
                transition: theme.transitions.create('width', { duration: theme.transitions.duration.enteringScreen }),
                overflowX: 'hidden',
              },
            }}
          >
            {drawerContent}
          </Drawer>
        )}
        <Box component="div" sx={{ flex: 1, minWidth: 0, overflow: 'auto' }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  )
}
