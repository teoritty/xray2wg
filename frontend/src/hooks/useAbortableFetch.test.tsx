import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useAbortableFetch } from "./useAbortableFetch";

describe("useAbortableFetch", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("aborts previous signal when getSignal is called again", () => {
    const { result } = renderHook(() => useAbortableFetch());
    const a = result.current.getSignal();
    const spy = vi.fn();
    a.addEventListener("abort", spy);
    act(() => {
      result.current.getSignal();
    });
    expect(spy).toHaveBeenCalled();
    expect(a.aborted).toBe(true);
  });

  it("aborts on unmount", () => {
    const { result, unmount } = renderHook(() => useAbortableFetch());
    const sig = result.current.getSignal();
    const spy = vi.fn();
    sig.addEventListener("abort", spy);
    unmount();
    expect(spy).toHaveBeenCalled();
    expect(sig.aborted).toBe(true);
  });
});
