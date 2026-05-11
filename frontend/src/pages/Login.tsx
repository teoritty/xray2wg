import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input, Label } from "../components/ui/Form";
import { authApi } from "../services/api";

export function LoginPage() {
  const nav = useNavigate();
  const qc = useQueryClient();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");

  const { data: health, isLoading } = useQuery({
    queryKey: ["setup-status"],
    queryFn: ({ signal }) => authApi.setupStatus({ signal }),
  });

  const bootstrap = useMutation({
    mutationFn: () => authApi.bootstrap(password, confirm),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["session"] });
      nav("/", { replace: true });
    },
  });

  const login = useMutation({
    mutationFn: () => authApi.login(password),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["session"] });
      nav("/", { replace: true });
    },
  });

  if (isLoading || !health) {
    return (
      <div className="flex min-h-full items-center justify-center">
        <p className="text-[#94a3b8]">Checking…</p>
      </div>
    );
  }

  const setup = health.needs_setup;

  return (
    <div className="flex min-h-full items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <h1 className="mb-6 text-center text-xl font-semibold text-[#e2e8f0]">
          {setup ? "Create admin password" : "Sign in"}
        </h1>
        <div className="space-y-4">
          <div>
            <Label htmlFor="pw">Password</Label>
            <Input
              id="pw"
              type="password"
              autoComplete={setup ? "new-password" : "current-password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          {setup ? (
            <div>
              <Label htmlFor="cf">Confirm password</Label>
              <Input
                id="cf"
                type="password"
                autoComplete="new-password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </div>
          ) : null}
          {(bootstrap.error || login.error) && (
            <p className="text-sm text-[#ef4444]">
              {(bootstrap.error as Error)?.message ?? (login.error as Error)?.message}
            </p>
          )}
          <Button
            className="w-full"
            disabled={bootstrap.isPending || login.isPending}
            onClick={() => {
              if (setup) bootstrap.mutate();
              else login.mutate();
            }}
          >
            {setup ? "Save and continue" : "Login"}
          </Button>
        </div>
      </Card>
    </div>
  );
}
