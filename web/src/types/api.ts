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

export interface UpdateSubmissionRequest {
  title?: string;
  description?: string;
  repoUrl?: string;
  demoUrl?: string;
  videoUrl?: string;
  trackId?: string;
}

export interface SubmissionDTO {
  submissionId: string;
  title: string;
  status: string;
  teamId: string;
}

export interface SubmissionDetail {
  id: string;
  title: string;
  description: string;
  status: string;
  teamId: string;
  eventId: string;
  trackId: string | null;
  repoUrl: string;
  demoUrl: string;
  videoUrl: string;
  coverUrl: string;
  finalScore: number | null;
  createdAt: string;
  updatedAt: string;
  submittedAt: string | null;
}

export interface SubmissionGalleryRow {
  SubmissionID: string;
  Title: string;
  Status: string;
  TeamID: string;
  TeamName: string;
  EventID: string;
  TrackID: string | null;
  TrackName: string | null;
  RepoURL: string | null;
  DemoURL: string | null;
  VideoURL: string | null;
  CoverURL: string | null;
  FinalScore: number | null;
  CreatedAt: string;
  UpdatedAt: string;
  SubmittedAt: string | null;
}

export interface FileDTO {
  id: string;
  name: string;
  url: string;
  role: string;
  sizeBytes: number;
}

