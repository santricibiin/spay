import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
  useEffect,
} from "react";
import { api, setToken, getToken } from "../api/client";

interface AuthState {
  isAuthenticated: boolean;
  loading: boolean;
  login: (password: string, captchaId: string, captchaValue: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (getToken()) {
      setIsAuthenticated(true);
    }
    setLoading(false);
  }, []);

  const login = useCallback(
    async (password: string, captchaId: string, captchaValue: string) => {
      const { token } = await api.post<{ token: string }>("/admin/login", {
        password,
        captcha_id: captchaId,
        captcha_value: captchaValue,
      });
      setToken(token);
      setIsAuthenticated(true);
    },
    [],
  );

  const logout = useCallback(() => {
    setToken(null);
    setIsAuthenticated(false);
  }, []);

  return (
    <AuthContext.Provider value={{ isAuthenticated, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be inside AuthProvider");
  return ctx;
}