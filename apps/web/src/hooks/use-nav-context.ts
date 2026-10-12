"use client";

import { useEffect, useState } from "react";
import { IS_STANDALONE, readLabsEnabled } from "@/lib/runtime-env";
import type { NavContext } from "@/lib/nav-visibility";

/**
 * Sidebar visibility context. Labs starts off on the first render so the
 * server HTML (no window.__ENV) and the first client render match; the
 * runtime ZENITH_LABS flag is applied after mount.
 */
export function useNavContext(): NavContext {
  const [labs, setLabs] = useState(false);
  useEffect(() => {
    setLabs(readLabsEnabled());
  }, []);
  return { standalone: IS_STANDALONE, labs };
}
