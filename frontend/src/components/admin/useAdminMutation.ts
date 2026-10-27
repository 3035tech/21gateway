import { useMutation, useQueryClient } from '@tanstack/react-query'
export function useAdminMutation<T>(
  mutationFn: (value: T) => Promise<void>,
  onSuccess?: () => void,
) {
  const client = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['admin'] })
      await client.invalidateQueries({ queryKey: ['session'] })
      onSuccess?.()
    },
  })
}
