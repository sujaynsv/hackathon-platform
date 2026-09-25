export interface ApiResponse<T> {
  data: T;
  meta: {
    requestId: string;
    timestamp: string;
    page?: number;
    totalPages?: number;
    totalCount?: number;
  };
}

export interface ApiError {
  code: string;
  message: string;
  requestId: string;
}

// ---- Auth ----

export interface UserProfile {
  id: string;
  email: string;
  displayName: string;
  avatarUrl: string | null;
  isAdmin: boolean;
  createdAt: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  displayName: string;
  captchaToken?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  user: UserProfile;
  accessToken: string;
  refreshToken?: string;
  requiresVerification?: boolean;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
}
