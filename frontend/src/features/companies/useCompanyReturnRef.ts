import { useQuery } from '@tanstack/react-query'
import { isPublicRef } from '../../lib/publicRef'
import { getCompany } from './api'

export function useCompanyReturnRef(rawRef: string | null) {
  const legacyId = /^\d+$/.test(rawRef ?? '') ? Number(rawRef) : null
  const validLegacyId =
    legacyId !== null && Number.isSafeInteger(legacyId) && legacyId > 0
  const legacyCompany = useQuery({
    queryKey: ['admin', 'legacy-company-ref', legacyId],
    queryFn: ({ signal }) => getCompany(legacyId!, signal),
    enabled: validLegacyId,
  })

  return {
    publicRef: validLegacyId
      ? (legacyCompany.data?.publicRef ?? null)
      : isPublicRef(rawRef)
        ? rawRef
        : null,
    ready: !validLegacyId || !legacyCompany.isPending,
  }
}
