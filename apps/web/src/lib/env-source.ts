export function isManagedEnv(ev: { source: string; source_id?: string }): boolean {
  return ev.source === "managed_service";
}

export function managedEnvLabel(_ev: { source: string; source_id?: string }): string {
  return "from database";
}
