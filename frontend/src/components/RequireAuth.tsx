import type { ReactElement } from "react";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Navigate, useLocation } from "react-router-dom";
import { authApi } from "../services/api";
import { useAuthStore } from "../store/auth";

export function RequireAuth({ children }: { children: ReactElement }) {
  const loc = useLocation();
  const clear = useAuthStore((s) => s.clear);
  const setSignedIn = useAuthStore((s) => s.setSignedIn);

  const { isLoading, isError, isSuccess } = useQuery({
    queryKey: ["session"],
    queryFn: ({ signal }) => authApi.me({ signal }),
    retry: false,
    staleTime: 30_000,
  });

  useEffect(() => {
    if (isSuccess) setSignedIn(true);
    if (isError) clear();
  }, [isSuccess, isError, setSignedIn, clear]);

  if (isLoading) {
    return (
      <div className="flex min-h-full items-center justify-center text-[#94a3b8]">Loading…</div>
    );
  }
  if (isError) {
    return <Navigate to="/login" replace state={{ from: loc.pathname }} />;
  }
  return children;
}
