import { createContext, useContext } from 'react'

export const CompanyViewContext = createContext<number | undefined>(undefined)

export function useCompanyView() {
  return useContext(CompanyViewContext)
}
