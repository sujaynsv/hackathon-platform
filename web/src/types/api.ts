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
