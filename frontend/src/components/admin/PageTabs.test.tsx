import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Activity, KeyRound } from 'lucide-react'
import { useState } from 'react'
import { expect, it } from 'vitest'
import { PageTabs, PageTabPanel } from './PageTabs'

function Example() {
  const [active, setActive] = useState<'operation' | 'access'>('operation')
  return (
    <>
      <PageTabs
        idPrefix="example"
        label="Seções"
        tabs={[
          { id: 'operation', label: 'Operação', icon: Activity },
          { id: 'access', label: 'Acessos', icon: KeyRound },
        ]}
        active={active}
        onChange={setActive}
      />
      <PageTabPanel
        idPrefix="example"
        tab="operation"
        active={active === 'operation'}
      >
        Saúde da integração
      </PageTabPanel>
      <PageTabPanel
        idPrefix="example"
        tab="access"
        active={active === 'access'}
      >
        Credenciais da API
      </PageTabPanel>
    </>
  )
}

it('switches panels by click and arrow keys with accessible tab state', async () => {
  const user = userEvent.setup()
  render(<Example />)
  const operation = screen.getByRole('tab', { name: 'Operação' })
  const access = screen.getByRole('tab', { name: 'Acessos' })
  expect(operation).toHaveAttribute('aria-selected', 'true')
  expect(screen.getByRole('tabpanel', { name: 'Operação' })).toBeVisible()
  await user.click(access)
  expect(access).toHaveAttribute('aria-selected', 'true')
  expect(screen.getByRole('tabpanel', { name: 'Acessos' })).toBeVisible()
  expect(
    screen.queryByRole('tabpanel', { name: 'Operação' }),
  ).not.toBeInTheDocument()
  await user.keyboard('{ArrowLeft}')
  expect(operation).toHaveAttribute('aria-selected', 'true')
  expect(operation).toHaveFocus()
})
