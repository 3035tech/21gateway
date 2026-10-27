import { lazy, Suspense } from 'react'
import { BrowserRouter } from 'react-router-dom'

const AdminApp = lazy(() => import('./AdminApp'))

const DemoApp = import.meta.env.DEV
  ? lazy(() => import('../features/demo/DemoApp'))
  : null

export function App() {
  if (
    DemoApp &&
    (import.meta.env.VITE_DEMO_MODE === 'true' ||
      window.location.pathname === '/demo' ||
      window.location.pathname.startsWith('/demo/'))
  ) {
    return (
      <BrowserRouter
        basename={
          window.location.pathname.startsWith('/demo') ? '/demo' : undefined
        }
      >
        <Suspense fallback={<p className="p-8">Carregando demonstração…</p>}>
          <DemoApp />
        </Suspense>
      </BrowserRouter>
    )
  }
  return (
    <BrowserRouter>
      <Suspense fallback={<p className="p-8">Carregando painel…</p>}>
        <AdminApp />
      </Suspense>
    </BrowserRouter>
  )
}
