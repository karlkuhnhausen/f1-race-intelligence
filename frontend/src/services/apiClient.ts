// API base client — all frontend data access flows through this module.
// Enforces the tier boundary: frontend → backend API only.

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';

interface ApiError {
  status: number;
  message: string;
}

class ApiClientError extends Error {
  status: number;
  constructor({ status, message }: ApiError) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${API_BASE}${path}`;
  const resp = await fetch(url, {
    ...init,
    headers: {
      'Accept': 'application/json',
      ...init?.headers,
    },
  });

  if (!resp.ok) {
    throw new ApiClientError({
      status: resp.status,
      message: `API ${resp.status}: ${resp.statusText}`,
    });
  }

  return resp.json() as Promise<T>;
}

export const apiClient = {
  get: <T>(path: string) => request<T>(path),
};

export { ApiClientError };
export type { ApiError };
