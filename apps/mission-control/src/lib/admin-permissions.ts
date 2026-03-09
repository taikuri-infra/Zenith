// Admin permission groups for Mission Control RBAC
export type AdminRoleType = "owner" | "admin" | "support" | "viewer";

export const PERMISSION_GROUPS = [
  "war_room",
  "analytics",
  "customers",
  "crm",
  "support_tickets",
  "quality",
  "services",
  "clusters",
  "infrastructure",
  "observability",
  "security",
  "modules",
  "backups",
  "gitops",
  "registry",
  "admin_settings",
] as const;

export type PermissionGroup = (typeof PERMISSION_GROUPS)[number];

const ROLE_PERMISSIONS: Record<AdminRoleType, readonly PermissionGroup[]> = {
  owner: PERMISSION_GROUPS,
  admin: PERMISSION_GROUPS.filter((p) => p !== "admin_settings"),
  support: [
    "war_room",
    "analytics",
    "customers",
    "crm",
    "support_tickets",
    "quality",
  ],
  viewer: [
    "war_room",
    "analytics",
    "customers",
    "crm",
    "support_tickets",
    "quality",
    "services",
    "clusters",
    "infrastructure",
    "observability",
    "security",
    "backups",
    "gitops",
    "registry",
  ],
};

export function hasPermission(
  role: AdminRoleType,
  group: PermissionGroup
): boolean {
  return ROLE_PERMISSIONS[role]?.includes(group) ?? false;
}

export function getPermissions(role: AdminRoleType): readonly PermissionGroup[] {
  return ROLE_PERMISSIONS[role] ?? [];
}
