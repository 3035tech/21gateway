import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Archive, Eye } from 'lucide-react'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { TableIconButton, TableIconLink } from './TableIconAction'

it('keeps icon actions identifiable by mouse, keyboard and screen reader', async () => {
  const archive = vi.fn()
  render(
    <MemoryRouter>
      <TableIconLink icon={Eye} label="Visualizar" to="/registro/1" />
      <TableIconButton
        icon={Archive}
        label="Arquivar"
        danger
        onClick={archive}
      />
    </MemoryRouter>,
  )
  const view = screen.getByRole('link', { name: 'Visualizar' })
  const action = screen.getByRole('button', { name: 'Arquivar' })
  expect(view).toHaveAttribute('title', 'Visualizar')
  expect(action).toHaveAttribute('title', 'Arquivar')
  expect(view).toHaveAttribute('href', '/registro/1')
  await userEvent.setup().click(action)
  expect(archive).toHaveBeenCalledOnce()
})
