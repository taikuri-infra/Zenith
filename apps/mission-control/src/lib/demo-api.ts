/**
 * Demo API client that returns mock data with a realistic delay.
 * Mirrors the shape of the real ApiClient so pages can use it as a drop-in.
 */

import type {
  Cluster,
  Module,
  Tenant,
  AuditEntry,
  PlatformUpdate,
  UpdateHistoryEntry,
  InfraOverview,
  PlatformState,
  PlatformSettings,
  DashboardStats,
  Customer,
  Plan,
  CustomerStats,
} from "./api";

import {
  demoClusters,
  demoModules,
  demoTenants,
  demoAuditLog,
  demoPlatformUpdate,
  demoUpdateHistory,
  demoInfrastructure,
  demoPlatformState,
  demoPlatformSettings,
  demoDashboardStats,
  demoCustomers,
  demoPlans,
  demoCustomerStats,
} from "./demo-data";

// Simulate a short network delay so skeleton states flash briefly
const delay = (ms = 300) => new Promise<void>((r) => setTimeout(r, ms));

export const demoApi = {
  dashboard: {
    stats: async (): Promise<DashboardStats> => {
      await delay();
      return demoDashboardStats;
    },
  },

  clusters: {
    list: async (): Promise<Cluster[]> => {
      await delay();
      return demoClusters;
    },
    get: async (name: string): Promise<Cluster> => {
      await delay();
      const cluster = demoClusters.find((c) => c.name === name);
      if (!cluster) throw new Error(`Cluster "${name}" not found`);
      return cluster;
    },
    create: async (): Promise<Cluster> => {
      throw new Error("Not available in demo mode");
    },
    delete: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    upgrade: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  tenants: {
    list: async (): Promise<Tenant[]> => {
      await delay();
      return demoTenants;
    },
    get: async (id: string): Promise<Tenant> => {
      await delay();
      const tenant = demoTenants.find((t) => t.name === id);
      if (!tenant) throw new Error(`Tenant "${id}" not found`);
      return tenant;
    },
    suspend: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  modules: {
    list: async (): Promise<Module[]> => {
      await delay();
      return demoModules;
    },
    install: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    uninstall: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    update: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    updateAll: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
  },

  audit: {
    list: async (params?: {
      limit?: number;
      offset?: number;
      actor?: string;
      cluster?: string;
      period?: string;
    }): Promise<AuditEntry[]> => {
      await delay();
      let filtered = [...demoAuditLog];
      if (params?.actor) {
        filtered = filtered.filter((e) => e.actor === params.actor);
      }
      if (params?.cluster) {
        filtered = filtered.filter((e) => e.cluster === params.cluster);
      }
      if (params?.limit) {
        filtered = filtered.slice(0, params.limit);
      }
      return filtered;
    },
  },

  updates: {
    check: async (): Promise<PlatformUpdate> => {
      await delay();
      return demoPlatformUpdate;
    },
    apply: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    history: async (): Promise<UpdateHistoryEntry[]> => {
      await delay();
      return demoUpdateHistory;
    },
  },

  infrastructure: {
    overview: async (): Promise<InfraOverview> => {
      await delay();
      return demoInfrastructure;
    },
  },

  state: {
    get: async (): Promise<PlatformState> => {
      await delay();
      return demoPlatformState;
    },
    export: async (): Promise<string> => {
      await delay();
      return "# Zenith Platform State Export\n# Generated: 2026-02-15\n---\napiVersion: zenith.dev/v1\nkind: PlatformState\nspec:\n  version: v1.3.2\n  clusters: 3\n  tenants: 5\n  modules: 8";
    },
  },

  settings: {
    get: async (): Promise<PlatformSettings> => {
      await delay();
      return demoPlatformSettings;
    },
    update: async (): Promise<PlatformSettings> => {
      throw new Error("Not available in demo mode");
    },
  },

  customers: {
    list: async (): Promise<Customer[]> => {
      await delay();
      return demoCustomers;
    },
    get: async (id: string): Promise<Customer> => {
      await delay();
      const customer = demoCustomers.find((c) => c.id === id);
      if (!customer) throw new Error(`Customer "${id}" not found`);
      return customer;
    },
    create: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    update: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    delete: async (): Promise<void> => {
      throw new Error("Not available in demo mode");
    },
    suspend: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    activate: async (): Promise<Customer> => {
      throw new Error("Not available in demo mode");
    },
    stats: async (): Promise<CustomerStats> => {
      await delay();
      return demoCustomerStats;
    },
  },

  plans: {
    list: async (): Promise<Plan[]> => {
      await delay();
      return demoPlans;
    },
    create: async (): Promise<Plan> => {
      throw new Error("Not available in demo mode");
    },
    update: async (): Promise<Plan> => {
      throw new Error("Not available in demo mode");
    },
  },
};
