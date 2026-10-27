import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, expect, it, vi } from 'vitest'
import { CredentialForm } from './CredentialForm'
import { api } from '../../lib/api'
import { MemoryRouter } from 'react-router-dom'
import type { Credential } from './api'
const credential: Credential = {
  id: 1,
  publicRef: '11111111-1111-4111-8111-111111111111',
  companyId: 1,
  companyName: 'Empresa exemplo',
  username: 'client',
  active: true,
  callbackUrl: 'https://client.example/dlr',
  callbacksEnabled: true,
  createdAt: '2026-10-14T12:00:00Z',
  expiresAt: null,
  providerTokenConfigured: true,
  whitelistEnabled: true,
  allowedIPs: ['203.0.113.10', '198.51.100.0/24'],
  moCallbackUrl: 'https://client.example/mo',
  moCallbacksEnabled: true,
  legacyAccountId: '2340',
}
const company = {
  id: 1,
  publicRef: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  name: 'Empresa exemplo',
  reference: 'example',
  active: true,
  createdAt: '2026-10-14T12:00:00Z',
  providerTokenConfigured: true,
  legacyProviderTokenFallback: false,
}
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () {
    this.setAttribute('open', '')
  }
  HTMLDialogElement.prototype.close = function () {
    this.removeAttribute('open')
  }
})
function renderEditor(close: () => void) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <CredentialForm
          credential={credential}
          onClose={close}
          onSecret={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}
it('preserves callbacks, hidden IP values and the existing token when editing', async () => {
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { credential, password: '' } })
  const get = vi.spyOn(api, 'get').mockResolvedValue({ data: company })
  try {
    const close = vi.fn()
    renderEditor(close)
    const user = userEvent.setup()
    await user.click(screen.getByRole('tab', { name: 'Notificações' }))
    expect(
      screen.getByRole('textbox', { name: 'URL de status (DLR)' }),
    ).toHaveValue(credential.callbackUrl)
    await user.click(screen.getByRole('tab', { name: 'IPs' }))
    await user.click(
      screen.getByRole('checkbox', { name: 'Permitir apenas IPs autorizados' }),
    )
    expect(
      screen.queryByRole('textbox', { name: 'Adicionar IP ou rede' }),
    ).not.toBeInTheDocument()
    await user.click(
      screen.getByRole('checkbox', { name: 'Permitir apenas IPs autorizados' }),
    )
    expect(
      screen.getByRole('region', { name: 'IPs individuais' }),
    ).toHaveTextContent('203.0.113.10')
    expect(
      screen.getByRole('region', { name: 'Redes CIDR' }),
    ).toHaveTextContent('198.51.100.0/24')
    await user.type(
      screen.getByRole('textbox', { name: 'Adicionar IP ou rede' }),
      '999.1.1.1{Enter}',
    )
    expect(screen.getByRole('alert')).toHaveTextContent('IP ou rede inválida')
    await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
    expect(put).not.toHaveBeenCalled()
    await user.clear(
      screen.getByRole('textbox', { name: 'Adicionar IP ou rede' }),
    )
    await user.type(
      screen.getByRole('textbox', { name: 'Adicionar IP ou rede' }),
      '192.0.2.1, 192.0.2.0/24{Enter}',
    )
    expect(
      screen.getByRole('region', { name: 'IPs individuais' }),
    ).toHaveTextContent('192.0.2.1')
    expect(
      screen.getByRole('region', { name: 'Redes CIDR' }),
    ).toHaveTextContent('192.0.2.0/24')
    await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
    await waitFor(() => expect(close).toHaveBeenCalled())
    expect(put).toHaveBeenCalledWith(
      '/admin/credentials/1',
      expect.objectContaining({
        companyId: 1,
        providerToken: undefined,
        clearProviderToken: false,
        allowedIPs: [...credential.allowedIPs, '192.0.2.1', '192.0.2.0/24'],
        callbackUrl: credential.callbackUrl,
        callbacksEnabled: true,
        moCallbackUrl: credential.moCallbackUrl,
        moCallbacksEnabled: true,
      }),
    )
    expect(get).toHaveBeenCalledWith('/admin/companies/1', expect.anything())
  } finally {
    put.mockRestore()
    get.mockRestore()
  }
})
it('directs provider token management to the company and closes the dialog', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({ data: company })
  const close = vi.fn()
  try {
    renderEditor(close)
    expect(screen.queryByLabelText('Novo token')).not.toBeInTheDocument()
    const link = await screen.findByRole('link', {
      name: 'Gerenciar token na empresa',
    })
    expect(link).toHaveAttribute('href', `/empresas/${company.publicRef}`)
    await userEvent.setup().click(link)
    expect(close).toHaveBeenCalledOnce()
  } finally {
    get.mockRestore()
  }
})

it('uses the selected company when creating an operator credential', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      credential: {
        ...credential,
        id: 2,
        companyId: 7,
        companyName: 'Aurora Varejo',
      },
      password: 'generated-secret',
    },
  })
  try {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <CredentialForm
            initialCompanyId={7}
            initialCompanyName="Aurora Varejo"
            onClose={vi.fn()}
            onSecret={vi.fn()}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(screen.getByRole('textbox', { name: 'Empresa' })).toHaveValue(
      'Aurora Varejo',
    )
    expect(
      screen.queryByRole('combobox', { name: 'Empresa' }),
    ).not.toBeInTheDocument()
    const user = userEvent.setup()
    expect(
      screen.getByRole('checkbox', {
        name: 'Gerar usuário Basic Auth aleatório',
      }),
    ).toBeChecked()
    expect(
      screen.queryByRole('textbox', { name: 'Usuário Basic Auth' }),
    ).not.toBeInTheDocument()
    expect(screen.getByText('Gerado ao salvar')).toBeInTheDocument()
    await user.click(
      screen.getByRole('checkbox', {
        name: 'Gerar usuário Basic Auth aleatório',
      }),
    )
    await user.type(
      screen.getByRole('textbox', { name: 'Usuário Basic Auth' }),
      'aurora-api',
    )
    await user.click(screen.getByRole('radio', { name: /Definir manualmente/ }))
    await user.type(
      screen.getByLabelText(/Digite a senha Basic Auth/),
      'senha-longa-para-aurora',
    )
    await user.click(screen.getByRole('button', { name: 'Criar credencial' }))
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/admin/credentials',
        expect.objectContaining({
          companyId: 7,
          username: 'aurora-api',
          manualPassword: 'senha-longa-para-aurora',
          generateUsername: false,
        }),
      ),
    )
  } finally {
    post.mockRestore()
  }
})

it('submits generated credentials without a username input', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      credential: { ...credential, id: 3, username: 'generated-user' },
      password: 'generated-secret',
    },
  })
  try {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const onSecret = vi.fn()
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <CredentialForm
            initialCompanyId={1}
            initialCompanyName="Empresa exemplo"
            onClose={vi.fn()}
            onSecret={onSecret}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(
      screen.queryByRole('textbox', { name: 'Usuário Basic Auth' }),
    ).not.toBeInTheDocument()
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'Criar credencial' }))
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/admin/credentials',
        expect.objectContaining({
          companyId: 1,
          generateUsername: true,
          username: '',
          manualPassword: undefined,
        }),
      ),
    )
    expect(onSecret).toHaveBeenCalledWith('generated-user', 'generated-secret')
  } finally {
    post.mockRestore()
  }
})
