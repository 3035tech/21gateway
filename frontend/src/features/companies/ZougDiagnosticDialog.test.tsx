import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { ZougDiagnosticDialog } from './ZougDiagnosticDialog'

HTMLDialogElement.prototype.showModal = function () {
  this.setAttribute('open', '')
}
HTMLDialogElement.prototype.close = function () {
  this.removeAttribute('open')
}

it('never includes a destination unless the extra cases are explicitly selected', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { runId: 'test-run', results: [] } })
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <ZougDiagnosticDialog
        publicRef="11111111-1111-4111-8111-111111111111"
        onClose={vi.fn()}
      />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  const include = screen.getByRole('checkbox', { name: /Incluir 2 casos/ })
  await user.click(include)
  await user.type(
    screen.getByRole('textbox', { name: /Número de teste autorizado/ }),
    '11999999999',
  )
  await user.click(include)
  await user.click(screen.getByRole('button', { name: 'Executar diagnóstico' }))
  await vi.waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/admin/companies/by-ref/11111111-1111-4111-8111-111111111111/zoug-diagnostic',
      { destination: '' },
      { timeout: 28_000 },
    ),
  )
  post.mockRestore()
})
