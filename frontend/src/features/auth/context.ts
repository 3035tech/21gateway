import { createContext, useContext } from 'react'
import type { SystemUser } from './api'
export const AuthContext = createContext<SystemUser | null>(null)
export const useUser = () => useContext(AuthContext)
