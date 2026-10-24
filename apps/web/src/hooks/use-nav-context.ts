"use client";

import { IS_STANDALONE } from "@/lib/runtime-env";
import type { NavContext } from "@/lib/nav-visibility";

/** Sidebar visibility context (see lib/nav-visibility). */
export function useNavContext(): NavContext {
  return { standalone: IS_STANDALONE };
}
