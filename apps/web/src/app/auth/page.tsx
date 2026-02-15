import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";

const mockRealms = [
  { name: "production", users: 1247, clients: 8, idps: "Google, GitHub", sessions: 89, status: "active" as const },
  { name: "staging", users: 23, clients: 4, idps: "\u2014", sessions: 3, status: "active" as const },
];

const mockUsers = [
  { email: "sarah@startup.com", name: "Sarah Chen", status: "active" as const, lastLogin: "2 hours ago", mfa: true },
  { email: "mike@startup.com", name: "Mike Johnson", status: "active" as const, lastLogin: "1 day ago", mfa: false },
  { email: "anna@startup.com", name: "Anna Schmidt", status: "active" as const, lastLogin: "3 hours ago", mfa: true },
  { email: "james@startup.com", name: "James Wilson", status: "pending" as const, lastLogin: "2 weeks ago", mfa: false },
  { email: "priya@startup.com", name: "Priya Patel", status: "pending" as const, lastLogin: "never", mfa: false },
];

const mockClients = [
  { clientId: "web-app", name: "Web Application", type: "public", protocol: "openid-connect", redirectUris: ["https://app.startup.com/callback"], enabled: true },
  { clientId: "mobile-app", name: "Mobile App", type: "public", protocol: "openid-connect", redirectUris: ["com.startup.app://callback"], enabled: true },
  { clientId: "admin-panel", name: "Admin Panel", type: "confidential", protocol: "openid-connect", redirectUris: ["https://admin.startup.com/callback"], enabled: true },
  { clientId: "partner-api", name: "Partner API", type: "confidential", protocol: "openid-connect", redirectUris: ["\u2014"], enabled: true },
];

const mockProviders = [
  { name: "Google", connected: true, label: "Sign in with Google", users: 847 },
  { name: "GitHub", connected: true, label: "Sign in with GitHub", users: 203 },
];

export default function AuthPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Auth Service</h1>
          <p className="text-sm text-neutral-500">Built-in identity provider for your applications</p>
        </div>

        {/* Info Banner */}
        <div className="rounded-lg border border-accent-500/30 bg-accent-500/5 px-4 py-3">
          <p className="text-xs text-accent-400">
            Powered by Zenith Auth (OpenID Connect + SAML) &mdash; Kong Gateway validates JWT tokens automatically
          </p>
        </div>

        {/* Realms */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Realms</h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Create Realm
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Realm</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Users</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Clients</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Identity Providers</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Active Sessions</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockRealms.map((realm) => (
                  <tr key={realm.name} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{realm.name}</td>
                    <td className="px-4 py-3 text-neutral-300">{realm.users.toLocaleString()}</td>
                    <td className="px-4 py-3 text-neutral-300">{realm.clients}</td>
                    <td className="px-4 py-3 text-neutral-400">{realm.idps}</td>
                    <td className="px-4 py-3 text-neutral-300">{realm.sessions}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={realm.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Users */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Users <span className="text-neutral-500 font-normal">&mdash; production realm</span></h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Add User
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Email</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Login</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">MFA</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Actions</th>
                </tr>
              </thead>
              <tbody>
                {mockUsers.map((user) => (
                  <tr key={user.email} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{user.email}</td>
                    <td className="px-4 py-3 text-neutral-300">{user.name}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={user.status} />
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{user.lastLogin}</td>
                    <td className="px-4 py-3">
                      {user.mfa ? (
                        <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                          </svg>
                          Enabled
                        </span>
                      ) : (
                        <span className="text-xs text-neutral-500">Disabled</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <button className="text-xs text-accent-400 hover:text-accent-300 transition-colors">Edit</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Clients */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Clients</h2>
            <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
              + Register Client
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Client ID</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Name</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Type</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Protocol</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Redirect URIs</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Enabled</th>
                </tr>
              </thead>
              <tbody>
                {mockClients.map((client) => (
                  <tr key={client.clientId} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-mono text-xs text-accent-400">{client.clientId}</td>
                    <td className="px-4 py-3 font-medium text-white">{client.name}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex rounded-full bg-surface-300 px-2 py-0.5 text-xs text-neutral-300">
                        {client.type}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-400">{client.protocol}</td>
                    <td className="px-4 py-3">
                      {client.redirectUris.map((uri) => (
                        <span key={uri} className="block font-mono text-xs text-neutral-400">{uri}</span>
                      ))}
                    </td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                        Enabled
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Identity Providers */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Identity Providers</h2>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {mockProviders.map((provider) => (
              <div key={provider.name} className="rounded-lg border border-border bg-surface-100 p-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-white">{provider.name}</span>
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                    Connected
                  </span>
                </div>
                <p className="mt-1 text-xs text-neutral-500">{provider.label}</p>
                <p className="mt-2 text-xs text-neutral-400">{provider.users.toLocaleString()} users</p>
              </div>
            ))}
            <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border bg-surface-100 p-4">
              <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
                + Add Provider
              </button>
              <p className="mt-2 text-xs text-neutral-500">SAML, OIDC, LDAP, Azure AD</p>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
