/**
 * Sidebar visibility rules.
 *
 * Zenith is self-hosted: every feature is available. The only items hidden in
 * standalone are the ones that exist solely for a hosted cloud (billing, paid
 * support, the cloud's own node pool).
 */

export interface NavVisibility {
  cloudOnly?: boolean;
}

export interface NavContext {
  standalone: boolean;
}

export function isNavItemVisible(item: NavVisibility, ctx: NavContext): boolean {
  return !(ctx.standalone && item.cloudOnly);
}
