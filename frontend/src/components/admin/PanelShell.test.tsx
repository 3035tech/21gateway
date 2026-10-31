import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, expect, it, vi } from 'vitest'
import { PanelShell } from './PanelShell'

beforeEach(() => window.localStorage.clear())

it('collapses the desktop menu, keeps navigation accessible, and remembers the preference', async () => {
  const user = userEvent.setup()
  const props = {
    name: 'Admin',
    role: 'Administrador',
    admin: true,
    demo: true,
    onLogout: vi.fn(),
  }
  const { unmount } = render(
    <MemoryRouter>
      <PanelShell {...props}>Conteúdo</PanelShell>
    </MemoryRouter>,
  )

  await user.click(
    screen.getByRole('button', { name: 'Recolher menu lateral' }),
  )
  expect(
    screen.getByRole('button', { name: 'Expandir menu lateral' }),
  ).toHaveAttribute('aria-expanded', 'false')
  const overview = screen.getAllByRole('link', { name: 'Visão geral' })
  expect(overview.some((link) => link.title === 'Visão geral')).toBe(true)
  expect(window.localStorage.getItem('21mobile.sidebar.collapsed')).toBe('true')

  unmount()
  render(
    <MemoryRouter>
      <PanelShell {...props}>Conteúdo</PanelShell>
    </MemoryRouter>,
  )
  expect(
    screen.getByRole('button', { name: 'Expandir menu lateral' }),
  ).toBeInTheDocument()
  await user.click(
    screen.getByRole('button', { name: 'Expandir menu lateral' }),
  )
  expect(window.localStorage.getItem('21mobile.sidebar.collapsed')).toBe(
    'false',
  )
})

it('keeps management navigation scoped to the user role', () => {
  const props = {
    name: 'Manager',
    role: 'Gestor',
    managementOnly: true,
    demo: true,
    onLogout: vi.fn(),
  }
  const view = render(
    <MemoryRouter>
      <PanelShell {...props} admin={false}>Conteúdo</PanelShell>
    </MemoryRouter>,
  )

  expect(screen.getAllByRole('link', { name: 'Empresas' }).length).toBeGreaterThan(0)
  expect(screen.queryByRole('link', { name: 'Usuários' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Envios' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Callbacks' })).not.toBeInTheDocument()

  view.unmount()
  render(
    <MemoryRouter>
      <PanelShell {...props} admin>Conteúdo</PanelShell>
    </MemoryRouter>,
  )
  expect(screen.getAllByRole('link', { name: 'Usuários' }).length).toBeGreaterThan(0)
  expect(screen.queryByRole('link', { name: 'Envios' })).not.toBeInTheDocument()
})
