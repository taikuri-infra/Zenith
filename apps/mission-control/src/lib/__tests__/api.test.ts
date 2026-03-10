import { describe, it, expect, beforeEach } from "vitest";
import {
  getAccessToken,
  setTokens,
  clearTokens,
  isAuthenticated,
  ApiError,
} from "../api";

describe("Token management (mission-control)", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("getAccessToken returns null when no token stored", () => {
    expect(getAccessToken()).toBeNull();
  });

  it("setTokens stores tokens with mc_ prefix keys", () => {
    setTokens("access-mc", "refresh-mc");
    expect(localStorage.getItem("mc_token")).toBe("access-mc");
    expect(localStorage.getItem("mc_refresh_token")).toBe("refresh-mc");
  });

  it("getAccessToken returns stored token", () => {
    setTokens("my-token", "my-refresh");
    expect(getAccessToken()).toBe("my-token");
  });

  it("clearTokens removes both tokens", () => {
    setTokens("access", "refresh");
    clearTokens();
    expect(getAccessToken()).toBeNull();
    expect(localStorage.getItem("mc_refresh_token")).toBeNull();
  });

  it("isAuthenticated returns false when no token", () => {
    expect(isAuthenticated()).toBe(false);
  });

  it("isAuthenticated returns true when token exists", () => {
    setTokens("token", "refresh");
    expect(isAuthenticated()).toBe(true);
  });
});

describe("ApiError (mission-control)", () => {
  it("creates error with status and body", () => {
    const err = new ApiError(500, "Internal Server Error");
    expect(err.status).toBe(500);
    expect(err.body).toBe("Internal Server Error");
    expect(err.name).toBe("ApiError");
  });

  it("is an instance of Error", () => {
    const err = new ApiError(400, "bad");
    expect(err).toBeInstanceOf(Error);
  });
});
