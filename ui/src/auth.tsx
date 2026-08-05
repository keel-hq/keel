import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react"
import { api, tokenStore } from "@/lib/api"
import type { UserInfo } from "@/types"

interface AuthValue {
  user: UserInfo | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}
const AuthContext = createContext<AuthValue | undefined>(undefined)
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const clear = useCallback(() => {
    tokenStore.clear()
    setUser(null)
    setLoading(false)
  }, [])
  useEffect(() => {
    const unauthorized = () => clear()
    window.addEventListener("keel:unauthorized", unauthorized)
    api
      .user()
      .then(setUser)
      .catch(clear)
      .finally(() => setLoading(false))
    return () => window.removeEventListener("keel:unauthorized", unauthorized)
  }, [clear])
  async function login(username: string, password: string) {
    await api.login(username, password)
    setUser(await api.user())
  }
  async function logout() {
    const logoutURL =
      user?.auth_mode === "external-proxy" ? user.logout_url : undefined
    try {
      await api.logout()
    } finally {
      clear()
      if (logoutURL) window.location.assign(logoutURL)
    }
  }
  return (
    <AuthContext.Provider value={{ user, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error("useAuth must be used inside AuthProvider")
  return value
}
