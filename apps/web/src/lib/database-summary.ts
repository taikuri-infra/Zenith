const ENGINES: [string, string][] = [["postgresql", "PostgreSQL"], ["mysql", "MySQL"], ["redis", "Redis"]];

export function databaseSummary(dbs: { engine: string; status: string }[]): string {
  const ready = dbs.filter((d) => d.status === "ready").length;
  const parts = [`${dbs.length} instances`, `${ready} ready`];
  for (const [engine, label] of ENGINES) {
    const n = dbs.filter((d) => d.engine === engine).length;
    if (n > 0) parts.push(`${n} ${label}`);
  }
  return parts.join(", ");
}
