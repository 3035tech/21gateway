import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import { CompanyAccessPicker } from './CompanyAccessPicker'

it('keeps a large company selection compact and paginated', async () => {
  const onSelectedPageChange = vi.fn()
  const companies = Array.from({ length: 100 }, (_, index) => ({
    id: String(index + 1),
    name: `Empresa ${index + 1}`,
  }))
  render(
    <CompanyAccessPicker
      options={companies.slice(0, 20)}
      selected={companies.slice(0, 5)}
      selectedIds={companies.map((company) => company.id)}
      selectedCount={100}
      selectedPage={1}
      selectedPageSize={5}
      onSelectedPageChange={onSelectedPageChange}
      search=""
      onSearchChange={vi.fn()}
      onToggle={vi.fn()}
      page={1}
      pageSize={20}
      total={100}
      onPageChange={vi.fn()}
    />,
  )

  expect(screen.getByText('100 empresas selecionadas')).toBeVisible()
  const selected = screen.getByRole('list', { name: 'Empresas vinculadas' })
  expect(within(selected).getAllByRole('button')).toHaveLength(5)
  expect(screen.getByText('Página 1 de 20')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Empresa 20' })).toHaveAttribute(
    'aria-pressed',
    'true',
  )

  await userEvent.click(screen.getAllByRole('button', { name: 'Próxima' })[0])
  expect(onSelectedPageChange).toHaveBeenCalledWith(2)
})
