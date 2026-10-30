import { lazy, Suspense } from 'react'
import { Route, Routes, Navigate } from 'react-router-dom'
import { AuthProvider } from '../features/auth/AuthProvider'
import { useUser } from '../features/auth/context'
import { LoginPage } from '../features/auth/LoginPage'
import { ForgotPasswordPage } from '../features/auth/ForgotPasswordPage'
import { ResetPasswordPage } from '../features/auth/ResetPasswordPage'
import { AdminLayout } from './AdminLayout'
import { NotFoundPage } from './NotFoundPage'
import { useTranslation } from 'react-i18next'
const Overview = lazy(() =>
  import('../features/overview/OverviewPage').then((m) => ({
    default: m.OverviewPage,
  })),
)
const Companies = lazy(() =>
  import('../features/companies/CompaniesPage').then((m) => ({
    default: m.CompaniesPage,
  })),
)
const CompanyDetail = lazy(() =>
  import('../features/companies/CompanyDetailPage').then((m) => ({
    default: m.CompanyDetailPage,
  })),
)
const Users = lazy(() =>
  import('../features/users/UsersPage').then((m) => ({ default: m.UsersPage })),
)
const Credentials = lazy(() =>
  import('../features/credentials/CredentialsPage').then((m) => ({
    default: m.CredentialsPage,
  })),
)
const Messages = lazy(() =>
  import('../features/messages/MessagesPage').then((m) => ({
    default: m.MessagesPage,
  })),
)
const Schedules = lazy(() =>
  import('../features/schedules/SchedulesPage').then((m) => ({
    default: m.SchedulesPage,
  })),
)
const Callbacks = lazy(() =>
  import('../features/callbacks/CallbacksPage').then((m) => ({
    default: m.CallbacksPage,
  })),
)
const Audit = lazy(() =>
  import('../features/audit/AuditPage').then((m) => ({ default: m.AuditPage })),
)
const Notifications = lazy(() =>
  import('../features/notifications/NotificationsPage').then((m) => ({
    default: m.NotificationsPage,
  })),
)
const Profile = lazy(() =>
  import('../features/users/ProfilePage').then((m) => ({
    default: m.ProfilePage,
  })),
)
function AdminOnly({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation()
  return useUser()?.role === 'admin' ? (
    children
  ) : (
    <p role="alert">{t('app.forbiddenPage')}</p>
  )
}
function Home() {
  return <Overview />
}
export default function AdminApp() {
  const { t } = useTranslation()
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/esqueci-senha" element={<ForgotPasswordPage />} />
        <Route path="/resetar-senha" element={<ResetPasswordPage />} />
        <Route
          path="*"
          element={
            <AdminLayout>
              <Suspense fallback={<p role="status">{t('app.loadingPage')}</p>}>
                <Routes>
                  <Route path="/" element={<Home />} />
                  <Route
                    path="/operacao"
                    element={<Navigate to="/" replace />}
                  />
                  <Route path="/empresas" element={<Companies />} />
                  <Route path="/empresas/:id" element={<CompanyDetail />} />
                  <Route
                    path="/usuarios"
                    element={
                      <AdminOnly>
                        <Users />
                      </AdminOnly>
                    }
                  />
                  <Route
                    path="/usuarios/:recordId"
                    element={
                      <AdminOnly>
                        <Users />
                      </AdminOnly>
                    }
                  />
                  <Route path="/credenciais" element={<Credentials />} />
                  <Route
                    path="/credenciais/:recordId"
                    element={<Credentials />}
                  />
                  <Route path="/envios" element={<Messages />} />
                  <Route path="/agendamentos" element={<Schedules />} />
                  <Route path="/perfil" element={<Profile />} />
                  <Route path="/callbacks" element={<Callbacks />} />
                  <Route
                    path="/notificacoes"
                    element={
                      <AdminOnly>
                        <Notifications />
                      </AdminOnly>
                    }
                  />
                  <Route
                    path="/auditoria"
                    element={
                      <AdminOnly>
                        <Audit />
                      </AdminOnly>
                    }
                  />
                  <Route path="*" element={<NotFoundPage />} />
                </Routes>
              </Suspense>
            </AdminLayout>
          }
        />
      </Routes>
    </AuthProvider>
  )
}
