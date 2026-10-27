import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { getReadiness } from '../../lib/api'
import { SystemPage } from './SystemPage'

vi.mock('../../lib/api', () => ({ getReadiness: vi.fn() }))
const readiness = vi.mocked(getReadiness)

beforeEach(() => readiness.mockReset())

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  render(
    <QueryClientProvider client={client}>
      <SystemPage />
    </QueryClientProvider>,
  )
}

describe('connection feedback', () => {
  it('does not show a previous success while a new check is running', async () => {
    readiness
      .mockResolvedValueOnce({ status: 'ok' })
      .mockImplementationOnce(() => new Promise(() => {}))
    const user = userEvent.setup()
    renderPage()
    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent('Conectado à API'),
    )
    await user.click(
      screen.getByRole('button', { name: 'Verificar novamente' }),
    )
    expect(screen.getByRole('status')).toHaveTextContent('Verificando conexão')
  })
  it('keeps internal errors private and can recover on a new check', async () => {
    readiness
      .mockRejectedValueOnce(new Error('password=secret SQL details'))
      .mockResolvedValueOnce({ status: 'ok' })
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('Conexão indisponível')
    expect(
      screen.queryByText(/password=secret|SQL details/),
    ).not.toBeInTheDocument()
    await user.click(
      screen.getByRole('button', { name: 'Verificar novamente' }),
    )
    await screen.findByText('Conectado à API')
    expect(readiness).toHaveBeenCalledTimes(2)
  })
})
