import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Box, CircularProgress } from '@mui/material'
import { AuthProvider } from './contexts/AuthContext'
import { BrandingProvider } from './contexts/BrandingContext'
import { AppShell } from './components/ui/AppShell'
import { ProtectedRoute } from './components/ProtectedRoute'
import { ErrorBoundary } from './components/ErrorBoundary'

const Home = lazy(() => import('./pages/Home').then((m) => ({ default: m.Home })))
const Login = lazy(() => import('./pages/Login').then((m) => ({ default: m.Login })))
const ForgotPassword = lazy(() => import('./pages/ForgotPassword').then((m) => ({ default: m.ForgotPassword })))
const ResetPassword = lazy(() => import('./pages/ResetPassword').then((m) => ({ default: m.ResetPassword })))
const Patients = lazy(() => import('./pages/Patients').then((m) => ({ default: m.Patients })))
const Backoffice = lazy(() => import('./pages/Backoffice').then((m) => ({ default: m.Backoffice })))
const BackofficeInvite = lazy(() => import('./pages/BackofficeInvite').then((m) => ({ default: m.BackofficeInvite })))
const BackofficeAudit = lazy(() => import('./pages/BackofficeAudit').then((m) => ({ default: m.BackofficeAudit })))
const BackofficeErrors = lazy(() => import('./pages/BackofficeErrors').then((m) => ({ default: m.BackofficeErrors })))
const SignContract = lazy(() => import('./pages/SignContract').then((m) => ({ default: m.SignContract })))
const VerifyContract = lazy(() => import('./pages/VerifyContract').then((m) => ({ default: m.VerifyContract })))
const RecordEntries = lazy(() => import('./pages/RecordEntries').then((m) => ({ default: m.RecordEntries })))
const PatientContracts = lazy(() => import('./pages/PatientContracts').then((m) => ({ default: m.PatientContracts })))
const RegisterProfessional = lazy(() => import('./pages/RegisterProfessional').then((m) => ({ default: m.RegisterProfessional })))
const RegisterPatient = lazy(() => import('./pages/RegisterPatient').then((m) => ({ default: m.RegisterPatient })))
const Remarcar = lazy(() => import('./pages/Remarcar').then((m) => ({ default: m.Remarcar })))
const ContractTemplates = lazy(() => import('./pages/ContractTemplates').then((m) => ({ default: m.ContractTemplates })))
const Appearance = lazy(() => import('./pages/Appearance').then((m) => ({ default: m.Appearance })))
const Profile = lazy(() => import('./pages/Profile').then((m) => ({ default: m.Profile })))
const ScheduleConfig = lazy(() => import('./pages/ScheduleConfig').then((m) => ({ default: m.ScheduleConfig })))
const Agenda = lazy(() => import('./pages/Agenda').then((m) => ({ default: m.Agenda })))

function PageFallback() {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 200 }}>
      <CircularProgress />
    </Box>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <BrandingProvider>
      <BrowserRouter>
        <ErrorBoundary>
          <Suspense fallback={<PageFallback />}>
          <Routes>
          <Route path="verify/:token" element={<VerifyContract />} />
          <Route path="sign-contract" element={<ErrorBoundary><SignContract /></ErrorBoundary>} />
          <Route path="/" element={<AppShell />}>
            <Route index element={<Home />} />
            <Route path="login" element={<Login />} />
            <Route path="admin/login" element={<Navigate to="/login" replace />} />
            <Route path="forgot-password" element={<ForgotPassword />} />
            <Route path="reset-password" element={<ResetPassword />} />
            <Route path="register" element={<RegisterProfessional />} />
            <Route path="register-patient" element={<RegisterPatient />} />
            <Route path="remarcar" element={<Remarcar />} />
            <Route
              path="patients"
              element={
                <ProtectedRoute roles={['PROFESSIONAL', 'SUPER_ADMIN', 'LEGAL_GUARDIAN']}>
                  <Patients />
                </ProtectedRoute>
              }
            />
            <Route
              path="patients/:patientId/contracts"
              element={
                <ProtectedRoute roles={['PROFESSIONAL', 'SUPER_ADMIN']}>
                  <PatientContracts />
                </ProtectedRoute>
              }
            />
            <Route
              path="patients/:patientId/record-entries"
              element={<Navigate to="../prontuario" replace />}
            />
            <Route
              path="patients/:patientId/prontuario"
              element={
                <ProtectedRoute roles={['PROFESSIONAL', 'SUPER_ADMIN', 'LEGAL_GUARDIAN']}>
                  <RecordEntries />
                </ProtectedRoute>
              }
            />
            <Route
              path="contract-templates"
              element={
                <ProtectedRoute roles={['PROFESSIONAL', 'SUPER_ADMIN']}>
                  <ContractTemplates />
                </ProtectedRoute>
              }
            />
            <Route
              path="appearance"
              element={
                <ProtectedRoute roles={['PROFESSIONAL']}>
                  <Appearance />
                </ProtectedRoute>
              }
            />
            <Route
              path="profile"
              element={
                <ProtectedRoute roles={['PROFESSIONAL']}>
                  <Profile />
                </ProtectedRoute>
              }
            />
            <Route
              path="schedule-config"
              element={
                <ProtectedRoute roles={['PROFESSIONAL', 'SUPER_ADMIN']}>
                  <ScheduleConfig />
                </ProtectedRoute>
              }
            />
            <Route
              path="agenda"
              element={
                <ProtectedRoute roles={['PROFESSIONAL', 'SUPER_ADMIN']}>
                  <Agenda />
                </ProtectedRoute>
              }
            />
            <Route
              path="backoffice"
              element={
                <ProtectedRoute roles={['SUPER_ADMIN']}>
                  <Backoffice />
                </ProtectedRoute>
              }
            />
            <Route
              path="backoffice/audit"
              element={
                <ProtectedRoute roles={['SUPER_ADMIN']}>
                  <BackofficeAudit />
                </ProtectedRoute>
              }
            />
            <Route
              path="backoffice/errors"
              element={
                <ProtectedRoute roles={['SUPER_ADMIN']}>
                  <BackofficeErrors />
                </ProtectedRoute>
              }
            />
            <Route
              path="backoffice/invites"
              element={
                <ProtectedRoute roles={['SUPER_ADMIN']}>
                  <BackofficeInvite />
                </ProtectedRoute>
              }
            />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
          </Suspense>
        </ErrorBoundary>
      </BrowserRouter>
      </BrandingProvider>
    </AuthProvider>
  )
}
