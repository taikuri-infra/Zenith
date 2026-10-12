/**
 * Sidebar visibility rules for the self-hosted edition.
 *
 * - saasOnly items: shown only in saas mode.
 * - labs items: non-core features, hidden in standalone unless ZENITH_LABS is on.
 * - everything else is core and always shown.
 */

export interface NavVisibility {
  saasOnly?: boolean;
  labs?: boolean;
}

export interface NavContext {
  standalone: boolean;
  labs: boolean;
}

export function parseLabsFlag(raw: string | undefined): boolean {
  const v = (raw ?? "").trim().toLowerCase();
  return v === "true" || v === "1" || v === "yes";
}

export function isNavItemVisible(item: NavVisibility, ctx: NavContext): boolean {
  if (!ctx.standalone) return true;
  if (item.saasOnly) return false;
  if (item.labs) return ctx.labs;
  return true;
}
