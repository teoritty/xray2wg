import { useCallback, useEffect, useRef } from "react";

/**
 * Returns an AbortSignal for one logical fetch. Calling getSignal() again
 * aborts the previous signal so overlapping requests are cancelled.
 */
export function useAbortableFetch() {
  const ctrlRef = useRef<AbortController | null>(null);

  const getSignal = useCallback(() => {
    ctrlRef.current?.abort();
    const c = new AbortController();
    ctrlRef.current = c;
    return c.signal;
  }, []);

  useEffect(() => {
    return () => {
      ctrlRef.current?.abort();
    };
  }, []);

  return { getSignal };
}
