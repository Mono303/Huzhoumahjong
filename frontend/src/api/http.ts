const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('hz_token')
  const headers = new Headers(options.headers ?? {})

  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  })

  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    throw new Error(payload.error ?? 'Request failed')
  }

  return payload as T
}

export function apiBaseUrl(): string {
  return API_BASE_URL
}
