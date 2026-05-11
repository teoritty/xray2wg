import { describe, it, expect, beforeEach } from "vitest";
import { useAuthStore } from "./auth";

describe("useAuthStore", () => {
  beforeEach(() => {
    useAuthStore.setState({ signedIn: false });
  });

  it("setSignedIn toggles session flag", () => {
    useAuthStore.getState().setSignedIn(true);
    expect(useAuthStore.getState().signedIn).toBe(true);
    useAuthStore.getState().clear();
    expect(useAuthStore.getState().signedIn).toBe(false);
  });
});
