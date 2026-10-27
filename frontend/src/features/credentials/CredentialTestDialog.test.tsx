import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeAll, expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { CredentialTestDialog } from './CredentialTestDialog'
import type { Credential } from './api'

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
  companyId: 2,
  companyName: 'Cliente exemplo',
  username: 'client-test',
  active: true,
  callbackUrl: '',
  callbacksEnabled: false,
  createdAt: '2026-10-27T12:00:00Z',
  providerTokenConfigured: true,
  whitelistEnabled: true,
  allowedIPs: ['192.0.2.10'],
  moCallbackUrl: '',
  moCallbacksEnabled: false,
  legacyAccountId: '',
}

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
