import type { ApiResponse, AuthResponse, LoginRequest, RegisterRequest, TokenPair } from '../types/api';

const IS_SERVER = typeof window === 'undefined';
const API_BASE = IS_SERVER
  ? (process.env.INTERNAL_API_URL ?? 'http://api:8080/api/v1')
  : (process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080/api/v1');

// In-memory token — never persisted to localStorage (XSS mitigation)
let _accessToken: string | null = null;

export class ApiClientError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
    public readonly requestId: string,
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<ApiResponse<T>> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string>),
  };

  if (_accessToken) {
    headers['Authorization'] = `Bearer ${_accessToken}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
  });

  const body = await res.json();

  if (!res.ok) {
    throw new ApiClientError(
      body.error?.code ?? 'UNKNOWN_ERROR',
      body.error?.message ?? 'An unexpected error occurred',
      res.status,
      body.meta?.requestId ?? '',
    );
  }

  return body as ApiResponse<T>;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  patch: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};

export const apiClient = {
  setToken(token: string | null) {
    _accessToken = token;
  },

  clearToken() {
    _accessToken = null;
  },

  async register(req: RegisterRequest): Promise<AuthResponse> {
    const res = await api.post<AuthResponse>('/auth/register', req);
    if (res.data.accessToken) {
      _accessToken = res.data.accessToken;
    }
    return res.data;
  },

  async login(req: LoginRequest): Promise<AuthResponse> {
    const res = await api.post<AuthResponse>('/auth/login', req);
    if (res.data.accessToken) {
      _accessToken = res.data.accessToken;
    }
    return res.data;
  },

  async refresh(refreshToken: string): Promise<TokenPair> {
    const res = await api.post<TokenPair>('/auth/refresh', { refreshToken });
    if (res.data.accessToken) {
      _accessToken = res.data.accessToken;
    }
    return res.data;
  },

  async logout(refreshToken?: string): Promise<void> {
    await api.post('/auth/logout', { refreshToken }).catch(() => {});
    _accessToken = null;
  },

  async verifyEmail(token: string): Promise<{ message: string }> {
    const res = await api.post<{ message: string }>('/auth/verify-email', { token });
    return res.data;
  },

  async createEvent(req: import('../types/api').CreateEventRequest): Promise<import('../types/api').EventDetailDTO> {
    const res = await api.post<import('../types/api').EventDetailDTO>('/events', req);
    return res.data;
  },

  async updateEvent(slug: string, req: import('../types/api').UpdateEventRequest): Promise<import('../types/api').EventDetailDTO> {
    const res = await api.patch<import('../types/api').EventDetailDTO>(`/events/${slug}`, req);
    return res.data;
  },
};

