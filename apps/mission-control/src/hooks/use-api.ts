"use client";

import { useState, useEffect, useCallback } from "react";

/**
 * Hook for fetching data from the API.
 * Automatically calls the fetcher on mount and whenever deps change.
 */
export function useApi<T>(
  fetcher: () => Promise<T>,
  deps: any[] = []
) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetcher();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return { data, loading, error, refetch };
}

/**
 * Hook that tries the real API first, falling back to demo data
 * when the API returns empty results or errors. Useful for pages
 * where the backend returns minimal/empty mock data.
 */
export function useApiWithFallback<T>(
  fetcher: () => Promise<T>,
  fallback: () => Promise<T>,
  isEmpty?: (data: T) => boolean,
  deps: any[] = []
) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [isDemo, setIsDemo] = useState(false);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    setIsDemo(false);
    try {
      const result = await fetcher();
      const empty = isEmpty
        ? isEmpty(result)
        : Array.isArray(result)
          ? result.length === 0
          : result == null;
      if (empty) {
        const fb = await fallback();
        setData(fb);
        setIsDemo(true);
      } else {
        setData(result);
      }
    } catch {
      try {
        const fb = await fallback();
        setData(fb);
        setIsDemo(true);
      } catch (fbErr) {
        setError(fbErr instanceof Error ? fbErr : new Error(String(fbErr)));
      }
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return { data, loading, error, refetch, isDemo };
}

/**
 * Hook for performing mutations (POST, PUT, DELETE, PATCH).
 * Does NOT auto-execute -- call `execute(input)` manually.
 */
export function useMutation<TInput, TOutput = void>(
  mutator: (input: TInput) => Promise<TOutput>
) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const execute = async (input: TInput): Promise<TOutput> => {
    setLoading(true);
    setError(null);
    try {
      const result = await mutator(input);
      return result;
    } catch (err) {
      const wrapped = err instanceof Error ? err : new Error(String(err));
      setError(wrapped);
      throw wrapped;
    } finally {
      setLoading(false);
    }
  };

  return { execute, loading, error };
}
