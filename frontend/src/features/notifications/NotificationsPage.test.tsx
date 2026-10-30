import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { NotificationsPage } from './NotificationsPage'

it('shows save feedback and updates the email preference after confirmation', async () => {
  let emailEnabled = false
  const get = vi.spyOn(api, 'get').mockImplementation(async (path) => {
    if (path === '/admin/notification-preferences') {
      return {
        data: {
          emailAvailable: true,
          items: [
            {
              code: 'unknown_sends',
              title: 'Envios com resultado incerto',
              emailEnabled,
            },
          ],
        },
      }
    }
    return {
      data: { items: [], unreadCount: 0, total: 0, page: 1, pageSize: 20 },
    }
  })
  const put = vi.spyOn(api, 'put').mockImplementation(async () => {
    emailEnabled = true
    return {
      data: {
        code: 'unknown_sends',
        title: 'Envios com resultado incerto',
        emailEnabled,
      },
    }
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <NotificationsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  const toggle = await screen.findByRole('checkbox', {
    name: 'Envios com resultado incerto: Enviar por e-mail',
  })
  expect(toggle).not.toBeChecked()
  await userEvent.setup().click(toggle)
  expect(put).toHaveBeenCalledWith(
    '/admin/notification-preferences/unknown_sends',
    { emailEnabled: true },
  )
  expect(await screen.findByText('Salvo')).toHaveAttribute('role', 'status')
  expect(toggle).toBeChecked()
  get.mockRestore()
  put.mockRestore()
})
