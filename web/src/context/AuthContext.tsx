'use client';

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from 'react';
import { apiClient, ApiClientError } from '@/lib/api';
import type { UserProfile, LoginRequest, RegisterRequest } from '@/types/api';

interface AuthContextValue {
  user: UserProfile | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (req: LoginRequest) => Promise<void>;
  register: (req: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const REFRESH_TOKEN_KEY = 'refresh_token';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Attempt silent refresh on mount to restore session
  useEffect(() => {
    const storedRefresh = sessionStorage.getItem(REFRESH_TOKEN_KEY);
    if (!storedRefresh) {
      setIsLoading(false);
      return;
    }
    apiClient
      .refresh(storedRefresh)
      .then(() => {
        // After refresh we don't have the user object back — 
        // for now mark as loading done. Future: add /auth/me endpoint.
        setIsLoading(false);
      })
      .catch(() => {
        sessionStorage.removeItem(REFRESH_TOKEN_KEY);
        setIsLoading(false);
      });
  }, []);

  const login = useCallback(async (req: LoginRequest) => {
    const data = await apiClient.login(req);
    setUser(data.user);
    if (data.refreshToken) {
      sessionStorage.setItem(REFRESH_TOKEN_KEY, data.refreshToken);
    }
  }, []);

  const register = useCallback(async (req: RegisterRequest) => {
    const data = await apiClient.register(req);
    if (!data.requiresVerification) {
      setUser(data.user);
      if (data.refreshToken) {
        sessionStorage.setItem(REFRESH_TOKEN_KEY, data.refreshToken);
      }
    }
  }, []);

  const logout = useCallback(async () => {
    const storedRefresh = sessionStorage.getItem(REFRESH_TOKEN_KEY) ?? undefined;
    await apiClient.logout(storedRefresh);
    sessionStorage.removeItem(REFRESH_TOKEN_KEY);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

// Re-export so consumers don't need to import from api.ts directly
export { ApiClientError };
