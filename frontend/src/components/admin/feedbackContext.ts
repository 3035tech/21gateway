import { createContext, useContext } from 'react'

export type Feedback = {
  success: (message: string) => void
  error: (message: string) => void
}

export const FeedbackContext = createContext<Feedback>({
  success: () => {},
  error: () => {},
})

export function useFeedback(): Feedback {
  return useContext(FeedbackContext)
}
