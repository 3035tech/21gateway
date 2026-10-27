import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { AuditPage } from './AuditPage'

it('shows the current actor name and sends audit filters and sorting to the API', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      items: [
        {
          id: 9,
          actorId: 3,
          actorName: 'Marina Costa',
          action: 'updated',
          resource: 'companies',
          resourceId: 42,
          createdAt: '2026-10-27T12:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 25,
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <AuditPage />
    </QueryClientProvider>,
  )
  expect(await screen.findByText('Marina Costa #3')).toBeInTheDocument()
  expect(get).toHaveBeenCalledWith(
    '/admin/audit',
    expect.objectContaining({
      params: expect.objectContaining({ sort: 'createdAt', direction: 'desc' }),
      signal: expect.any(AbortSignal),
    }),
  )
  const user = userEvent.setup()
  await user.type(
    screen.getByRole('spinbutton', { name: 'ID do responsável' }),
    '3',
  )
  await user.click(screen.getByRole('combobox', { name: 'Ação' }))
  await user.click(screen.getByRole('option', { name: 'Cadastro atualizado' }))
  await user.click(screen.getByRole('combobox', { name: 'Recurso' }))
  await user.click(screen.getByRole('option', { name: 'Empresas' }))
  await vi.waitFor(() =>
    expect(get).toHaveBeenLastCalledWith(
      '/admin/audit',
      expect.objectContaining({
        params: expect.objectContaining({
          actorId: 3,
          action: 'updated',
          resource: 'companies',
        }),
        signal: expect.any(AbortSignal),
      }),
    ),
  )
  get.mockRestore()
})
