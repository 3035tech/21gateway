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
  await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
  expect(await screen.findByText('Informe o nome.')).toBeInTheDocument()
  expect(post).not.toHaveBeenCalled()
  await user.type(
    screen.getByRole('textbox', { name: /^Nome/ }),
    'Empresa teste',
  )
  await user.type(screen.getByLabelText('Referência'), 'CLI-001')
  await user.click(screen.getByRole('button', { name: 'Salvar alterações' }))
  await vi.waitFor(() => expect(close).toHaveBeenCalled())
  expect(post).toHaveBeenCalledWith('/admin/companies', {
    name: 'Empresa teste',
    reference: 'CLI-001',
    active: true,
  })
  post.mockRestore()
})
