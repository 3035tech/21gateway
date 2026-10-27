import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it } from 'vitest'
import { FeedbackProvider } from './Feedback'
import { useFeedback } from './feedbackContext'

function Actions() {
  const feedback = useFeedback()
  return (
    <>
      <button onClick={() => feedback.success('Cadastro salvo')}>
        Success
      </button>
      <button onClick={() => feedback.error('Falha ao salvar')}>Error</button>
    </>
  )
}

it('announces action results and lets the user dismiss them', async () => {
  const user = userEvent.setup()
  render(
    <FeedbackProvider>
      <Actions />
    </FeedbackProvider>,
  )
  await user.click(screen.getByRole('button', { name: 'Success' }))
  expect(screen.getByRole('status')).toHaveTextContent('Cadastro salvo')
  await user.click(screen.getByRole('button', { name: 'Error' }))
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
  expect(screen.getByRole('alert')).toHaveTextContent('Falha ao salvar')
  await user.click(screen.getByRole('button', { name: 'Fechar aviso' }))
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})
