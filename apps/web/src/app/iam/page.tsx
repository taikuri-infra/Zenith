import { Shell } from "@/components/shell";

const mockApiKeys = [
  {
    name: "Production API",
    prefix: "zen_prod_a8f2...x91k",
    scopes: ["read", "write", "deploy"],
    created: "Jan 15, 2026",
    lastUsed: "2 hours ago",
    status: "active" as const,
  },
  {
    name: "CI/CD Pipeline",
    prefix: "zen_ci_d4e7...m32n",
    scopes: ["deploy", "registry:push"],
    created: "Feb 1, 2026",
    lastUsed: "35 min ago",
    status: "active" as const,
  },
  {
    name: "Monitoring",
    prefix: "zen_mon_b1c9...k47p",
    scopes: ["read"],
    created: "Feb 10, 2026",
    lastUsed: "5 min ago",
    status: "active" as const,
  },
];

const mockTeamMembers = [
  { name: "Babak Dorani", email: "babak@startup.com", role: "Owner" as const, lastActive: "5 min ago", joined: "Nov 1, 2025" },
  { name: "Sarah Chen", email: "sarah@startup.com", role: "Admin" as const, lastActive: "2 hours ago", joined: "Nov 15, 2025" },
  { name: "Mike Johnson", email: "mike@startup.com", role: "Developer" as const, lastActive: "1 day ago", joined: "Jan 20, 2026" },
  { name: "Intern", email: "intern@startup.com", role: "Viewer" as const, lastActive: "1 week ago", joined: "Feb 5, 2026" },
];

const mockRoles = [
  { name: "Owner", description: "Full platform access, billing, danger zone", members: 1, color: "bg-purple-500/10 text-purple-400 border-purple-500/20" },
  { name: "Admin", description: "Manage services, deployments, team", members: 1, color: "bg-blue-500/10 text-blue-400 border-blue-500/20" },
  { name: "Developer", description: "Deploy, view logs, manage apps", members: 1, color: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" },
  { name: "Viewer", description: "Read-only access to all resources", members: 1, color: "bg-neutral-500/10 text-neutral-400 border-neutral-500/20" },
];

const roleBadgeStyles: Record<string, string> = {
  Owner: "bg-purple-500/10 text-purple-400",
  Admin: "bg-blue-500/10 text-blue-400",
  Developer: "bg-emerald-500/10 text-emerald-400",
  Viewer: "bg-neutral-500/10 text-neutral-400",
};

const scopeColors: Record<string, string> = {
  read: "bg-emerald-500/10 text-emerald-400",
  write: "bg-blue-500/10 text-blue-400",
  deploy: "bg-accent-500/10 text-accent-400",
  "registry:push": "bg-amber-500/10 text-amber-400",
};

export default function IAMPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">IAM</h1>
          <p className="text-sm text-neutral-500">Platform identity &amp; access management</p>
        </div>

        {/* API Keys */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">API Keys</h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Create API Key
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Key Prefix</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scopes</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Used</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockApiKeys.map((key) => (
                  <tr key={key.name} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{key.name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{key.prefix}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {key.scopes.map((scope) => (
                          <span
                            key={scope}
                            className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${scopeColors[scope] ?? "bg-surface-300 text-neutral-400"}`}
                          >
                            {scope}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{key.created}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{key.lastUsed}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                        Active
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Team Members */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Team Members</h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Invite Member
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Email</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Role</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Active</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Joined</th>
                </tr>
              </thead>
              <tbody>
                {mockTeamMembers.map((member) => (
                  <tr key={member.email} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{member.name}</td>
                    <td className="px-4 py-3 text-neutral-300">{member.email}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${roleBadgeStyles[member.role]}`}>
                        {member.role}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{member.lastActive}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{member.joined}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Roles */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Roles</h2>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {mockRoles.map((role) => (
              <div key={role.name} className={`rounded-lg border p-4 ${role.color}`}>
                <p className="text-sm font-medium">{role.name}</p>
                <p className="mt-1 text-xs opacity-70">{role.description}</p>
                <p className="mt-3 text-xs opacity-50">{role.members} member{role.members !== 1 ? "s" : ""}</p>
              </div>
            ))}
          </div>
        </section>
      </div>
    </Shell>
  );
}
