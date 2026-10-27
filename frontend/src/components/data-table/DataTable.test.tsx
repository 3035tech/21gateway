import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { DataTable } from './DataTable'
import { useListState } from './useListState'

function Harness({ error }: { error?: string }) {
  const state = useListState()
  return (
    <>
      <DataTable
        columns={[{ accessorKey: 'name', header: 'Nome' }]}
        data={[{ id: 'company-1', name: 'Empresa de teste' }]}
        total={75}
        getRowId={(row) => row.id}
        error={error}
        {...state}
      />
      <output aria-label="Estado da consulta">
        {JSON.stringify({
          pagination: state.pagination,
          sorting: state.sorting,
          search: state.search,
        })}
      </output>
    </>
  )
}

function queryState() {
  return JSON.parse(
    screen.getByLabelText('Estado da consulta').textContent ?? '{}',
  )
}

describe('DataTable server query controls', () => {
  it('changes page and resets pagination when page size changes', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Próxima' }))
    expect(queryState().pagination).toEqual({ pageIndex: 1, pageSize: 25 })
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Itens por página' }),
      '50',
    )
    expect(queryState().pagination).toEqual({ pageIndex: 0, pageSize: 50 })
  })
  it('changes sort direction by clicking the header and resets page', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Próxima' }))
    await user.click(screen.getByRole('button', { name: 'Nome' }))
    expect(queryState().sorting).toEqual([{ id: 'name', desc: false }])
    expect(queryState().pagination.pageIndex).toBe(0)
    expect(screen.getByRole('columnheader')).toHaveAttribute(
      'aria-sort',
      'ascending',
    )
    await user.click(screen.getByRole('button', { name: 'Nome' }))
    expect(queryState().sorting).toEqual([{ id: 'name', desc: true }])
  })
  it('applies search only on submit and resets page', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('button', { name: 'Próxima' }))
    await user.type(
      screen.getByRole('searchbox', { name: 'Buscar' }),
      '  exemplo  ',
    )
    expect(queryState().search).toBe('')
    await user.click(screen.getByRole('button', { name: 'Buscar' }))
    expect(queryState().search).toBe('exemplo')
    expect(queryState().pagination.pageIndex).toBe(0)
  })
  it('does not present stale rows as a successful result when the query fails', () => {
    render(<Harness error="Não foi possível carregar os resultados." />)
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Não foi possível carregar os resultados.',
    )
    expect(screen.queryByText('Empresa de teste')).not.toBeInTheDocument()
    expect(
      screen.getByText('Resultados indisponíveis. Tente atualizar a consulta.'),
    ).toBeInTheDocument()
  })
})
