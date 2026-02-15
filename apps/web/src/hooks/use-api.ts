"use client";

import { useCallback, useEffect, useState } from "react";
import { ApiError, UnauthorizedError } from "@/lib/api";

/**
 * Generic hook for API data fetching with loading and error states.
 */
export function useApi<T>(
  fetcher: () => Promise<T>,
  deps: unknown[] = []
): {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => void;
} {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetcher();
      setData(result);
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        if (typeof window !== "undefined") {
          window.location.href = "/login";
        }
        return;
      }
      if (err instanceof ApiError) {
        setError(`Error ${err.status}: ${err.statusText}`);
      } else if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("An unknown error occurred");
      }
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}

/**
 * Hook for API mutations (POST, PUT, DELETE) with loading and error states.
 */
export function useMutation<TData, TVariables>(
  mutationFn: (variables: TVariables) => Promise<TData>
): {
  mutate: (variables: TVariables) => Promise<TData | null>;
  data: TData | null;
  loading: boolean;
  error: string | null;
} {
  const [data, setData] = useState<TData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const mutate = useCallback(
    async (variables: TVariables) => {
      setLoading(true);
      setError(null);
      try {
        const result = await mutationFn(variables);
        setData(result);
        return result;
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          if (typeof window !== "undefined") {
            window.location.href = "/login";
          }
          return null;
        }
        if (err instanceof ApiError) {
          setError(`Error ${err.status}: ${err.statusText}`);
        } else if (err instanceof Error) {
          setError(err.message);
        } else {
          setError("An unknown error occurred");
        }
        return null;
      } finally {
        setLoading(false);
      }
    },
    [mutationFn]
  );

  return { mutate, data, loading, error };
}
