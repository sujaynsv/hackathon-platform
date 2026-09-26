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
  user: UserProfile;
}

export interface VerifyEmailRequest {
  token: string;
}

export interface VerifyEmailResponse {
  message: string;
}

// ---- Submissions ----

export interface CreateSubmissionRequest {
  title: string;
  description?: string;
  trackId?: string;
  repoUrl?: string;
  demoUrl?: string;
}

export interface SubmissionDTO {
  submissionId: string;
  title: string;
  status: string;
  teamId: string;
}

