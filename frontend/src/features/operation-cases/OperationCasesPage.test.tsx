import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { AuthContext } from '../auth/context'
import { OperationCasesPage } from './OperationCasesPage'

HTMLDialogElement.prototype.showModal = function () {
  this.setAttribute('open', '')
}
HTMLDialogElement.prototype.close = function () {
  this.removeAttribute('open')
}

it('lets an operator claim an uncertain send without offering SMS resend', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      items: [
        {
          kind: 'unknown_send',
          sourceId: 42,
          attemptId: 42,
          companyId: 7,
          companyName: 'Example',
          createdAt: '2026-10-31T10:00:00Z',
          status: 'open',
          ownerId: null,
          ownerName: '',
          note: '',
          tracked: false,
          sourceActive: true,
        },
      ],
      total: 1,
      page: 1,
      pageSize: 25,
    },
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({ data: {} })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  try {
    render(
      <QueryClientProvider client={client}>
        <AuthContext.Provider
          value={{
            id: 9,
            publicRef: '550e8400-e29b-41d4-a716-446655440000',
            name: 'Operator',
            email: 'operator@example.test',
            role: 'operator',
            active: true,
            createdAt: '2026-10-31T10:00:00Z',
            companyIds: [7],
          }}
        >
          <MemoryRouter>
            <OperationCasesPage />
          </MemoryRouter>
        </AuthContext.Provider>
      </QueryClientProvider>,
    )
    expect(await screen.findByText('Example')).toBeInTheDocument()
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'Visualizar' }))
    expect(screen.getByText(/Esta tela não reenvia SMS/)).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Abrir registro de origem' }),
    ).toHaveAttribute('href', '/envios?search=42')
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'Assumir caso' }))
    expect(put).toHaveBeenCalledWith('/admin/operation-cases/unknown_send/42', {
      status: 'in_progress',
      note: '',
    })
  } finally {
    get.mockRestore()
    put.mockRestore()
  }
})
