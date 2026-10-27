import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import { MfaCodeInput } from './MfaCodeInput'

it('advances between six digits and submits once when complete', async () => {
  const user = userEvent.setup()
  const complete = vi.fn()
  function Harness() {
    const [value, setValue] = useState('')
    return (
      <MfaCodeInput value={value} onChange={setValue} onComplete={complete} />
    )
  }
  render(<Harness />)
  for (let index = 1; index <= 6; index++) {
    const input = screen.getByRole('textbox', { name: `Dígito ${index} de 6` })
    await user.type(input, String(index))
    if (index < 6)
      expect(
        screen.getByRole('textbox', { name: `Dígito ${index + 1} de 6` }),
      ).toHaveFocus()
  }
  expect(complete).toHaveBeenCalledTimes(1)
  expect(complete).toHaveBeenCalledWith('123456')
})
