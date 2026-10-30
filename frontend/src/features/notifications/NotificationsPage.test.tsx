import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import i18n from '../../lib/i18n'
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
  expect(screen.getByRole('tab', { name: 'Ocorrências' })).toHaveAttribute(
    'aria-selected',
    'true',
  )
  await userEvent
    .setup()
    .click(screen.getByRole('tab', { name: 'Alertas por e-mail' }))
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

it('shows known incident copy in the selected language and keeps unknown copy', async () => {
  await i18n.changeLanguage('en-US')
  const get = vi.spyOn(api, 'get').mockImplementation(async (path) => {
    if (path === '/admin/notification-preferences') {
      return {
        data: {
          emailAvailable: true,
          items: [
            {
              code: 'unknown_sends',
              title: 'Envios com resultado incerto',
              emailEnabled: false,
            },
          ],
        },
      }
    }
    return {
      data: {
        items: [
          {
            id: 1,
            type: 'unknown_sends',
            title: 'Envios com resultado incerto',
            description:
              '3 envios sem resultado conclusivo; investigue antes de reenviar.',
            path: '/envios',
            openedAt: '2026-10-30T10:00:00Z',
            resolvedAt: null,
            readAt: null,
          },
          {
            id: 2,
            type: 'new_type',
            title: 'Título futuro',
            description: 'Descrição futura',
            path: '/envios',
            openedAt: '2026-10-30T09:00:00Z',
            resolvedAt: null,
            readAt: null,
          },
        ],
        unreadCount: 2,
        total: 2,
        page: 1,
        pageSize: 20,
      },
    }
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  try {
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <NotificationsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    )
    expect(
      await screen.findAllByText('Messages with uncertain outcomes'),
    ).toHaveLength(2)
    expect(
      screen.getAllByText(
        '3 messages have uncertain outcomes; investigate before resending.',
      ),
    ).toHaveLength(2)
    expect(screen.getAllByText('Título futuro')).toHaveLength(2)
    expect(screen.getAllByText('Descrição futura')).toHaveLength(2)
    await userEvent
      .setup()
      .click(screen.getByRole('tab', { name: 'Email alerts' }))
    expect(
      await screen.findByRole('checkbox', {
        name: 'Messages with uncertain outcomes: Send email',
      }),
    ).toBeInTheDocument()
    await i18n.changeLanguage('pt-BR')
    expect(
      screen.getByRole('checkbox', {
        name: 'Envios com resultado incerto: Enviar por e-mail',
      }),
    ).toBeInTheDocument()
  } finally {
    get.mockRestore()
    await i18n.changeLanguage('pt-BR')
  }
})
