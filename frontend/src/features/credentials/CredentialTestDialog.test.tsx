import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { CredentialTestDialog } from './CredentialTestDialog'
import type { Credential } from './api'
import type { Company } from '../companies/api'

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () {
    this.setAttribute('open', '')
  }
  HTMLDialogElement.prototype.close = function () {
    this.removeAttribute('open')
  }
})

const credential: Credential = {
  id: 21,
  publicRef: '11111111-1111-4111-8111-111111111111',
  companyId: 2,
  companyName: 'Cliente exemplo',
  username: 'client-test',
  active: true,
  callbackUrl: '',
  callbacksEnabled: false,
  createdAt: '2026-10-27T12:00:00Z',
  expiresAt: null,
  providerTokenConfigured: true,
  whitelistEnabled: true,
  allowedIPs: ['192.0.2.10'],
  moCallbackUrl: '',
  moCallbacksEnabled: false,
  legacyAccountId: '',
}

const company: Company = {
  id: 2,
  publicRef: '11111111-1111-4111-8111-111111111111',
  name: 'Cliente exemplo',
  reference: 'CLIENTE',
  active: true,
  createdAt: '2026-10-27T12:00:00Z',
  providerTokenConfigured: true,
  legacyProviderTokenFallback: false,
  smsProvider: 'zoug',
}

it('tests the saved company token without an API credential', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      submissionState: 'accepted',
      idSms: 'provider-123',
      correlationId: 'test-123',
    },
  })
  try {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <CredentialTestDialog company={company} onClose={vi.fn()} />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    expect(screen.queryByText('1. Acesso do cliente')).not.toBeInTheDocument()
    await user.type(
      screen.getByRole('textbox', { name: /Número de teste/ }),
      '11987654321',
    )
    await user.click(
      screen.getByRole('checkbox', { name: /Confirmo que este número/ }),
    )
    await user.click(
      screen.getByRole('button', { name: 'Enviar um SMS de teste' }),
    )
    await waitFor(() =>
      expect(screen.getByText('Fornecedor aceitou o SMS')).toBeInTheDocument(),
    )
    expect(post).toHaveBeenCalledWith(
      '/admin/companies/by-ref/11111111-1111-4111-8111-111111111111/test-send',
      { destination: '11987654321' },
      { timeout: 20_000 },
    )
  } finally {
    post.mockRestore()
  }
})

it('shows an uncertain result without suggesting the Messages history', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      submissionState: 'unknown',
      correlationId: 'test-uncertain',
      failureCode: 'UNKNOWN_OUTCOME',
    },
  })
  try {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <CredentialTestDialog company={company} onClose={vi.fn()} />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    await user.type(
      screen.getByRole('textbox', { name: /Número de teste/ }),
      '11987654321',
    )
    await user.click(
      screen.getByRole('checkbox', { name: /Confirmo que este número/ }),
    )
    await user.click(
      screen.getByRole('button', { name: 'Enviar um SMS de teste' }),
    )
    await waitFor(() =>
      expect(
        screen.getByText('Resultado incerto: não reenvie automaticamente'),
      ).toBeInTheDocument(),
    )
    expect(screen.getByText('Referência: test-uncertain')).toBeInTheDocument()
    expect(
      screen.queryByText(/Consulte o histórico antes/),
    ).not.toBeInTheDocument()
  } finally {
    post.mockRestore()
  }
})

it('requires authorized destination confirmation and sends once', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      submissionState: 'accepted',
      idSms: '42',
      correlationId: 'test-123',
    },
  })
  try {
    const client = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <CredentialTestDialog credential={credential} onClose={vi.fn()} />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    const send = screen.getByRole('button', { name: 'Enviar um SMS de teste' })
    expect(send).toBeDisabled()
    await user.type(
      screen.getByRole('textbox', { name: /Número de teste/ }),
      '11987654321',
    )
    expect(
      screen.getByRole('textbox', { name: /Número de teste/ }),
    ).toHaveValue('(11) 98765-4321')
    expect(send).toBeDisabled()
    await user.click(
      screen.getByRole('checkbox', { name: /Confirmo que este número/ }),
    )
    await user.click(send)
    await waitFor(() =>
      expect(screen.getByText('Fornecedor aceitou o SMS')).toBeInTheDocument(),
    )
    expect(post).toHaveBeenCalledTimes(1)
    expect(post).toHaveBeenCalledWith(
      '/admin/credentials/21/test-send',
      { destination: '11987654321' },
      { timeout: 20_000 },
    )
    expect(
      screen.queryByRole('button', { name: 'Enviar um SMS de teste' }),
    ).not.toBeInTheDocument()
    expect(screen.getByText('ID do envio: 42')).toBeInTheDocument()
  } finally {
    post.mockRestore()
  }
})

it('sends a custom message to the masked mobile number', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: {
      submissionState: 'accepted',
      idSms: '43',
      correlationId: 'test-124',
    },
  })
  try {
    const client = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <CredentialTestDialog credential={credential} onClose={vi.fn()} />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    await user.type(
      screen.getByRole('textbox', { name: /Número de teste/ }),
      '11987654321',
    )
    await user.click(
      screen.getByRole('checkbox', { name: 'Usar mensagem padrão de teste' }),
    )
    const send = screen.getByRole('button', { name: 'Enviar um SMS de teste' })
    expect(send).toBeDisabled()
    await user.type(
      screen.getByRole('textbox', { name: /Mensagem de teste/ }),
      'Teste livre do token',
    )
    await user.click(
      screen.getByRole('checkbox', { name: /Confirmo que este número/ }),
    )
    await user.click(send)
    await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    expect(post).toHaveBeenCalledWith(
      '/admin/credentials/21/test-send',
      { destination: '11987654321', messageText: 'Teste livre do token' },
      { timeout: 20_000 },
    )
  } finally {
    post.mockRestore()
  }
})
