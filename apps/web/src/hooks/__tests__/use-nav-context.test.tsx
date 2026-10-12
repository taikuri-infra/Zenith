import { describe, it, expect, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { useNavContext } from "../use-nav-context";

function Probe() {
  const ctx = useNavContext();
  return <span>{ctx.labs ? "labs-on" : "labs-off"}</span>;
}

describe("useNavContext", () => {
  afterEach(() => {
    delete window.__ENV;
  });

  it("first render has labs off so server and client HTML match", () => {
    window.__ENV = { NEXT_PUBLIC_ZENITH_LABS: "true" };
    expect(renderToString(<Probe />)).toContain("labs-off");
  });

  it("turns labs on after mount when the runtime flag is set", () => {
    window.__ENV = { NEXT_PUBLIC_ZENITH_LABS: "true" };
    const { result } = renderHook(() => useNavContext());
    expect(result.current.labs).toBe(true);
  });

  it("keeps labs off after mount when the runtime flag is unset", () => {
    const { result } = renderHook(() => useNavContext());
    expect(result.current.labs).toBe(false);
  });
});
