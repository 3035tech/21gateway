import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { expect, it, vi } from 'vitest'
import { api } from '../../lib/api'
import { useCompanyReturnRef } from './useCompanyReturnRef'

it('preserves the return to a company when opening an old numeric link', async () => {
  const publicRef = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      id: 42,
      publicRef,
      name: 'Empresa exemplo',
      reference: 'EX',
      active: true,
      createdAt: '2026-10-27T12:00:00Z',
      providerTokenConfigured: false,
      legacyProviderTokenFallback: false,
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  try {
    const { result } = renderHook(() => useCompanyReturnRef('42'), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    })
    await waitFor(() =>
      expect(result.current).toEqual({ publicRef, ready: true }),
    )
    expect(get).toHaveBeenCalledWith('/admin/companies/42', expect.anything())
  } finally {
    get.mockRestore()
    client.clear()
  }
})
