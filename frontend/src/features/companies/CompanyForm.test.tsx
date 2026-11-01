import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, expect, it, vi } from 'vitest'
import { CompanyForm } from './CompanyForm'
import { api } from '../../lib/api'
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () {
    this.setAttribute('open', '')
  }
  HTMLDialogElement.prototype.close = function () {
    this.removeAttribute('open')
  }
})
it('validates company name and persists the submitted body', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({ data: {} })
  const close = vi.fn()
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <CompanyForm onClose={close} />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: 'Criar empresa' }))
  expect(await screen.findByText('Informe o nome.')).toBeInTheDocument()
  expect(post).not.toHaveBeenCalled()
  await user.type(
    screen.getByRole('textbox', { name: /^Nome/ }),
    'Empresa teste',
  )
  await user.type(screen.getByLabelText('Referência'), 'CLI-001')
  await user.click(screen.getByRole('button', { name: 'Criar empresa' }))
  await vi.waitFor(() => expect(close).toHaveBeenCalled())
  expect(post).toHaveBeenCalledWith('/admin/companies', {
    name: 'Empresa teste',
    reference: 'CLI-001',
    active: true,
    smsProvider: 'zoug',
  })
  post.mockRestore()
})

it('submits the provider-issued token on the company record', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({ data: {} })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <CompanyForm
        company={{
          id: 4,
          publicRef: '11111111-1111-4111-8111-111111111111',
          name: 'Empresa teste',
          reference: 'CLI-004',
          active: true,
          createdAt: '2026-10-27T10:00:00Z',
          providerTokenConfigured: false,
          legacyProviderTokenFallback: false,
          smsProvider: 'zoug',
          providerRoutes: [
            { channel: 'sms', provider: 'zoug', status: 'active' },
          ],
        }}
        onClose={vi.fn()}
      />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.type(
    screen.getByLabelText(/^Token fornecido pela Zoug/),
    'provider-test-token',
  )
  await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
  await vi.waitFor(() =>
    expect(put).toHaveBeenCalledWith('/admin/companies/4', {
      name: 'Empresa teste',
      reference: 'CLI-004',
      active: true,
      providerToken: 'provider-test-token',
    }),
  )
  put.mockRestore()
})

it('adds a planned RCS route without replacing the active SMS route', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({ data: {} })
  const client = new QueryClient()
  render(
    <QueryClientProvider client={client}>
      <CompanyForm onClose={vi.fn()} />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.type(screen.getByRole('textbox', { name: /^Nome/ }), 'Empresa RCS')
  expect(screen.getByText('SMS · Zoug')).toBeInTheDocument()
  expect(screen.getByText('Integrada')).toBeInTheDocument()
  await user.click(screen.getByText('Adicionar rota por canal'))
  await user.click(screen.getByRole('combobox', { name: 'Provedor' }))
  await user.click(screen.getByRole('option', { name: 'Zenvia' }))
  expect(
    screen.getByLabelText(/^Token fornecido pela Zoug/),
  ).toBeInTheDocument()
  await user.click(screen.getByRole('combobox', { name: 'Canal' }))
  expect(screen.getByRole('option', { name: 'SMS' })).toBeInTheDocument()
  expect(screen.getByRole('option', { name: 'RCS' })).toBeInTheDocument()
  expect(screen.getByRole('option', { name: 'WhatsApp' })).toBeInTheDocument()
  await user.click(screen.getByRole('option', { name: 'RCS' }))
  expect(
    screen.getByRole('button', { name: 'Adicionar rota em preparação' }),
  ).toBeEnabled()

  await user.click(screen.getByRole('combobox', { name: 'Provedor' }))
  await user.click(screen.getByRole('option', { name: 'Google' }))
  expect(
    screen.getByText(/A 21 ainda não tem conta parceira do Google/),
  ).toBeInTheDocument()
  await user.click(screen.getByRole('combobox', { name: 'Canal' }))
  expect(screen.getByRole('option', { name: 'RCS' })).toBeInTheDocument()
  expect(screen.queryByRole('option', { name: 'SMS' })).not.toBeInTheDocument()
  await user.click(screen.getByRole('option', { name: 'RCS' }))
  await user.click(
    screen.getByRole('button', { name: 'Adicionar rota em preparação' }),
  )
  expect(screen.getByText('RCS · Google')).toBeInTheDocument()
  expect(
    screen.getByText('Parceria Google pendente; envio indisponível.'),
  ).toBeInTheDocument()
  expect(screen.getByText('SMS · Zoug')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Criar empresa' }))
  await vi.waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/admin/companies',
      expect.objectContaining({
        smsProvider: 'zoug',
        plannedRoutes: [{ channel: 'rcs', provider: 'google' }],
      }),
    ),
  )
  post.mockRestore()
})

it('preserves planned routes on unrelated edits and removes them only on request', async () => {
  const put = vi.spyOn(api, 'put').mockResolvedValue({ data: {} })
  const company = {
    id: 8,
    publicRef: '11111111-1111-4111-8111-111111111111',
    name: 'Empresa teste',
    reference: 'CLI-008',
    active: true,
    createdAt: '2026-10-27T10:00:00Z',
    providerTokenConfigured: false,
    legacyProviderTokenFallback: false,
    smsProvider: 'zoug',
    providerRoutes: [
      { channel: 'sms' as const, provider: 'zoug', status: 'active' as const },
      {
        channel: 'rcs' as const,
        provider: 'google',
        status: 'planned' as const,
      },
    ],
  }
  const user = userEvent.setup()
  const first = render(
    <QueryClientProvider client={new QueryClient()}>
      <CompanyForm company={company} onClose={vi.fn()} />
    </QueryClientProvider>,
  )
  expect(screen.getByText('RCS · Google')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
  await vi.waitFor(() =>
    expect(put).toHaveBeenCalledWith('/admin/companies/8', {
      name: 'Empresa teste',
      reference: 'CLI-008',
      active: true,
    }),
  )
  first.unmount()
  put.mockClear()
  render(
    <QueryClientProvider client={new QueryClient()}>
      <CompanyForm company={company} onClose={vi.fn()} />
    </QueryClientProvider>,
  )
  await user.click(
    screen.getByRole('button', {
      name: 'Remover rota RCS em preparação',
    }),
  )
  await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
  await vi.waitFor(() =>
    expect(put).toHaveBeenCalledWith('/admin/companies/8', {
      name: 'Empresa teste',
      reference: 'CLI-008',
      active: true,
      plannedRoutes: [],
    }),
  )
  put.mockRestore()
})
