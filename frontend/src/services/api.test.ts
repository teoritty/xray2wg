import { describe, it, expect } from "vitest";
import { ApiException, parseError } from "./api";

describe("parseError", () => {
  it("maps JSON error envelope to ApiException", async () => {
    const res = new Response(JSON.stringify({ error: { code: "VALIDATION", message: "bad" } }), {
      status: 400,
    });
    try {
      await parseError(res);
      expect.fail("expected throw");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException);
      const ex = e as ApiException;
      expect(ex.code).toBe("VALIDATION");
      expect(ex.message).toBe("bad");
    }
  });

  it("falls back when body is not JSON", async () => {
    const res = new Response("nope", { status: 502, statusText: "Bad Gateway" });
    try {
      await parseError(res);
      expect.fail("expected throw");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiException);
      expect((e as ApiException).code).toBe("HTTP_ERROR");
    }
  });
});
