import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { errorMessage } from '../../lib/admin'
import { useFeedback } from './feedbackContext'
export function useAdminMutation<T>(
  mutationFn: (value: T) => Promise<void>,
  onSuccess?: () => void,
  successKey: string | ((value: T) => string) = 'feedback.saved',
) {
  const client = useQueryClient()
  const feedback = useFeedback()
  const { t } = useTranslation()
  return useMutation({
    mutationFn,
    onSuccess: async (_result, value) => {
      await client.invalidateQueries({ queryKey: ['admin'] })
      await client.invalidateQueries({ queryKey: ['session'] })
      onSuccess?.()
      feedback.success(
        t(typeof successKey === 'function' ? successKey(value) : successKey),
      )
    },
    onError: (error) => feedback.error(errorMessage(error)),
  })
}
