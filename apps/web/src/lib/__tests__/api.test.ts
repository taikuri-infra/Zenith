import { describe, it, expect, beforeEach } from "vitest";
import {
  getAccessToken,
  setTokens,
  clearTokens,
  isAuthenticated,
  ApiError,
  UnauthorizedError,
} from "../api";

describe("Token management", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("getAccessToken returns null when no token stored", () => {
    expect(getAccessToken()).toBeNull();
  });

  it("setTokens stores tokens in localStorage", () => {
    setTokens("access-123", "refresh-456");
    expect(localStorage.getItem("zenith_access_token")).toBe("access-123");
    expect(localStorage.getItem("zenith_refresh_token")).toBe("refresh-456");
  });

  it("getAccessToken returns stored token", () => {
    setTokens("my-token", "my-refresh");
    expect(getAccessToken()).toBe("my-token");
  });

  it("clearTokens removes both tokens", () => {
    setTokens("access", "refresh");
    clearTokens();
    expect(getAccessToken()).toBeNull();
    expect(localStorage.getItem("zenith_refresh_token")).toBeNull();
  });

  it("isAuthenticated returns false when no token", () => {
    expect(isAuthenticated()).toBe(false);
  });

  it("isAuthenticated returns true when token exists", () => {
    setTokens("token", "refresh");
    expect(isAuthenticated()).toBe(true);
  });
});

describe("ApiError", () => {
  it("creates error with status and message", () => {
    const err = new ApiError(404, "Not Found");
    expect(err.status).toBe(404);
    expect(err.statusText).toBe("Not Found");
    expect(err.name).toBe("ApiError");
    expect(err.message).toBe("API Error 404: Not Found");
  });

  it("stores optional body", () => {
    const body = { detail: "missing" };
    const err = new ApiError(400, "Bad Request", body);
    expect(err.body).toEqual(body);
  });
});

describe("UnauthorizedError", () => {
  it("creates a 401 error", () => {
    const err = new UnauthorizedError();
    expect(err.status).toBe(401);
    expect(err.name).toBe("UnauthorizedError");
  });

  it("is an instance of ApiError", () => {
    const err = new UnauthorizedError();
    expect(err).toBeInstanceOf(ApiError);
  });
});
