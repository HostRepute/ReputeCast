import { QueryClient } from '@tanstack/react-query'
import { ApiError } from '@/api/client'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: true,
      retry: (count, error) => {
        // Client errors won't fix themselves; retry network and 5xx twice.
        if (error instanceof ApiError && error.status < 500) return false
        return count < 2
      },
    },
    mutations: { retry: false },
  },
})
