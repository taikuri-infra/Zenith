"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { Modal } from "@/components/modal";
import { useState } from "react";

interface Realm {
  name: string;
  users: number;
  clients: number;
  idps: string;
  sessions: number;
  status: "active" | "stopped";
}

interface User {
  email: string;
  name: string;
  status: "active" | "pending";
  lastLogin: string;
  mfa: boolean;
}

interface Client {
  clientId: string;
  name: string;
  type: string;
  protocol: string;
  redirectUris: string[];
  enabled: boolean;
}

interface Provider {
  name: string;
  connected: boolean;
  label: string;
  users: number;
}

const initialRealms: Realm[] = [
  { name: "production", users: 1247, clients: 8, idps: "Google, GitHub", sessions: 89, status: "active" },
  { name: "staging", users: 23, clients: 4, idps: "\u2014", sessions: 3, status: "active" },
];

const initialUsers: User[] = [
  { email: "sarah@startup.com", name: "Sarah Chen", status: "active", lastLogin: "2 hours ago", mfa: true },
  { email: "mike@startup.com", name: "Mike Johnson", status: "active", lastLogin: "1 day ago", mfa: false },
  { email: "anna@startup.com", name: "Anna Schmidt", status: "active", lastLogin: "3 hours ago", mfa: true },
  { email: "james@startup.com", name: "James Wilson", status: "pending", lastLogin: "2 weeks ago", mfa: false },
  { email: "priya@startup.com", name: "Priya Patel", status: "pending", lastLogin: "never", mfa: false },
];

const initialClients: Client[] = [
  { clientId: "web-app", name: "Web Application", type: "public", protocol: "openid-connect", redirectUris: ["https://app.startup.com/callback"], enabled: true },
  { clientId: "mobile-app", name: "Mobile App", type: "public", protocol: "openid-connect", redirectUris: ["com.startup.app://callback"], enabled: true },
  { clientId: "admin-panel", name: "Admin Panel", type: "confidential", protocol: "openid-connect", redirectUris: ["https://admin.startup.com/callback"], enabled: true },
  { clientId: "partner-api", name: "Partner API", type: "confidential", protocol: "openid-connect", redirectUris: ["\u2014"], enabled: true },
];

const initialProviders: Provider[] = [
  { name: "Google", connected: true, label: "Sign in with Google", users: 847 },
  { name: "GitHub", connected: true, label: "Sign in with GitHub", users: 203 },
];

export default function AuthPage() {
  const [realms, setRealms] = useState<Realm[]>(initialRealms);
  const [users, setUsers] = useState<User[]>(initialUsers);
  const [clients, setClients] = useState<Client[]>(initialClients);
  const [providers, setProviders] = useState<Provider[]>(initialProviders);

  // Realm modal
  const [showCreateRealm, setShowCreateRealm] = useState(false);
  const [realmName, setRealmName] = useState("");

  // User modal
  const [showAddUser, setShowAddUser] = useState(false);
  const [userEmail, setUserEmail] = useState("");
  const [userName, setUserName] = useState("");

  // Client modal
  const [showRegisterClient, setShowRegisterClient] = useState(false);
  const [clientId, setClientId] = useState("");
  const [clientName, setClientName] = useState("");
  const [clientType, setClientType] = useState("public");

  // Provider modal
  const [showAddProvider, setShowAddProvider] = useState(false);
  const [providerName, setProviderName] = useState("Google");

  const handleCreateRealm = () => {
    if (!realmName.trim()) return;
    const newRealm: Realm = {
      name: realmName.trim(),
      users: 0,
      clients: 0,
      idps: "\u2014",
      sessions: 0,
      status: "active",
    };
    setRealms((prev) => [...prev, newRealm]);
    setShowCreateRealm(false);
    setRealmName("");
  };

  const handleAddUser = () => {
    if (!userEmail.trim()) return;
    const newUser: User = {
      email: userEmail.trim(),
      name: userName.trim() || userEmail.trim().split("@")[0],
      status: "pending",
      lastLogin: "never",
      mfa: false,
    };
    setUsers((prev) => [...prev, newUser]);
    setShowAddUser(false);
    setUserEmail("");
    setUserName("");
  };

  const handleRegisterClient = () => {
    if (!clientId.trim()) return;
    const newClient: Client = {
      clientId: clientId.trim(),
      name: clientName.trim() || clientId.trim(),
      type: clientType,
      protocol: "openid-connect",
      redirectUris: ["\u2014"],
      enabled: true,
    };
    setClients((prev) => [...prev, newClient]);
    setShowRegisterClient(false);
    setClientId("");
    setClientName("");
    setClientType("public");
  };

  const redirectUriMap: Record<string, string> = {
    Google: "https://auth.zenith.cloud/callback/google",
    GitHub: "https://auth.zenith.cloud/callback/github",
    GitLab: "https://auth.zenith.cloud/callback/gitlab",
    Microsoft: "https://auth.zenith.cloud/callback/microsoft",
  };

  const handleAddProvider = () => {
    const newProvider: Provider = {
      name: providerName,
      connected: true,
      label: `Sign in with ${providerName}`,
      users: 0,
    };
    setProviders((prev) => [...prev, newProvider]);
    setShowAddProvider(false);
    setProviderName("Google");
  };

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
            <button
              onClick={() => setShowCreateRealm(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
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
                {realms.map((realm) => (
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
            <button
              onClick={() => setShowAddUser(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
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
                {users.map((user) => (
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
            <button
              onClick={() => setShowRegisterClient(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
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
                {clients.map((client) => (
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
            {providers.map((provider) => (
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
              <button
                onClick={() => setShowAddProvider(true)}
                className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
              >
                + Add Provider
              </button>
              <p className="mt-2 text-xs text-neutral-500">SAML, OIDC, LDAP, Azure AD</p>
            </div>
          </div>
        </section>
      </div>

      {showCreateRealm && (
        <Modal title="Create Realm" onClose={() => setShowCreateRealm(false)}>
          <form onSubmit={(e) => { e.preventDefault(); handleCreateRealm(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Realm Name</label>
              <input
                type="text"
                value={realmName}
                onChange={(e) => setRealmName(e.target.value)}
                placeholder="my-realm"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowCreateRealm(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Create</button>
            </div>
          </form>
        </Modal>
      )}

      {showAddUser && (
        <Modal title="Add User" onClose={() => setShowAddUser(false)}>
          <form onSubmit={(e) => { e.preventDefault(); handleAddUser(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Email</label>
              <input
                type="email"
                value={userEmail}
                onChange={(e) => setUserEmail(e.target.value)}
                placeholder="user@example.com"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={userName}
                onChange={(e) => setUserName(e.target.value)}
                placeholder="Full Name"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowAddUser(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Add User</button>
            </div>
          </form>
        </Modal>
      )}

      {showRegisterClient && (
        <Modal title="Register Client" onClose={() => setShowRegisterClient(false)}>
          <form onSubmit={(e) => { e.preventDefault(); handleRegisterClient(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Client ID</label>
              <input
                type="text"
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
                placeholder="my-client"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={clientName}
                onChange={(e) => setClientName(e.target.value)}
                placeholder="Client Name"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Type</label>
              <select
                value={clientType}
                onChange={(e) => setClientType(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="public">public</option>
                <option value="confidential">confidential</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowRegisterClient(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Register</button>
            </div>
          </form>
        </Modal>
      )}

      {showAddProvider && (
        <Modal title="Add Identity Provider" onClose={() => setShowAddProvider(false)}>
          <form onSubmit={(e) => { e.preventDefault(); handleAddProvider(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Provider Name</label>
              <select
                value={providerName}
                onChange={(e) => setProviderName(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="Google">Google</option>
                <option value="GitHub">GitHub</option>
                <option value="GitLab">GitLab</option>
                <option value="Microsoft">Microsoft</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Redirect URI</label>
              <input
                type="text"
                readOnly
                value={redirectUriMap[providerName] || ""}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none cursor-not-allowed"
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowAddProvider(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Add Provider</button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
