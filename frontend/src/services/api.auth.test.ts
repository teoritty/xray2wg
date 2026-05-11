import { describe, it, expect, vi, afterEach } from "vitest";
import { ApiException, apiOk, authApi, parseError } from "./api";

describe("auth API (cookies, no token storage)", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    if (typeof localStorage !== "undefined") {
      localStorage.clear();
    }
  });

  it("parseError exposes request_id from backend envelope", async () => {
    const res = new Response(
      JSON.stringify({
        error: { code: "INTERNAL", message: "internal server error", request_id: "rid-abc" },
      }),
      { status: 500 },
    );
    try {
      await parseError(res);
      expect.fail("expected throw");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException);
      expect((e as ApiException).requestId).toBe("rid-abc");
    }
  });

  it("logout uses credentials include", async () => {
    const f = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    await authApi.logout();
    expect(f).toHaveBeenCalled();
    const last = f.mock.calls[f.mock.calls.length - 1];
    expect((last[1] as RequestInit).credentials).toBe("include");
  });

  it("auth/me uses credentials include", async () => {
    const f = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ user_id: 1 }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await authApi.me();
    expect(f).toHaveBeenCalled();
    const last = f.mock.calls[f.mock.calls.length - 1];
    expect((last[1] as RequestInit).credentials).toBe("include");
  });

  it("refresh uses credentials include", async () => {
    const f = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    await apiOk("/auth/refresh", { method: "POST", skipAuth: true });
    const last = f.mock.calls[f.mock.calls.length - 1];
    expect(String(last[0])).toContain("/auth/refresh");
    expect((last[1] as RequestInit).credentials).toBe("include");
  });

  it("localStorage has no tokens after login success path (cookies only)", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    await apiOk("/auth/login", {
      method: "POST",
      body: JSON.stringify({ password: "x" }),
      skipAuth: true,
    });
    if (typeof localStorage !== "undefined") {
      expect(localStorage.getItem("access_token")).toBeNull();
      expect(localStorage.getItem("auth-storage")).toBeNull();
    }
  });
});
