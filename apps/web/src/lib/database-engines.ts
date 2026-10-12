export interface EngineOption {
  id: string;
  label: string;
  enabled: boolean;
  /** Badge shown on a disabled option. */
  disabledNote?: string;
}

/**
 * Database engines offered in the create dialog. Self-host runs Postgres,
 * MySQL and Redis as Docker containers (no MongoDB). SaaS keeps its own
 * set: MySQL is not provisioned there yet.
 */
export function engineOptions(standalone: boolean): EngineOption[] {
  if (standalone) {
    return [
      { id: "postgresql", label: "PostgreSQL", enabled: true },
      { id: "mysql", label: "MySQL", enabled: true },
      { id: "mongodb", label: "MongoDB", enabled: false, disabledNote: "Not available on self-host" },
      { id: "redis", label: "Redis", enabled: true },
    ];
  }
  return [
    { id: "postgresql", label: "PostgreSQL", enabled: true },
    { id: "mysql", label: "MySQL", enabled: false, disabledNote: "Coming Soon" },
    { id: "mongodb", label: "MongoDB", enabled: true },
    { id: "redis", label: "Redis", enabled: true },
  ];
}
