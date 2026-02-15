import { Shell } from "@/components/shell";
import { projectPlan } from "@/lib/mock-data";

const resourceUsage = [
  { resource: "Apps", quantity: "6 services", cost: "12.00" },
  { resource: "Databases", quantity: "3 instances", cost: "8.40" },
  { resource: "Storage", quantity: "12.3 GB used", cost: "2.00" },
  { resource: "Planets", quantity: "5 nodes (CX22)", cost: "5.00" },
];

const totalCost = "27.40";

export default function BillingPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Billing</h1>
          <p className="text-sm text-neutral-500">Plan, usage, and cost breakdown</p>
        </div>

        {/* Current Plan */}
        <section>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-white">Current Plan</p>
                  <span className="inline-flex items-center rounded-full bg-accent-500/10 px-2.5 py-0.5 text-xs font-medium text-accent-400">
                    {projectPlan}
                  </span>
                </div>
                <p className="mt-1 text-xs text-neutral-500">
                  6 apps, 3 databases, 5 compute nodes, 60 GB storage
                </p>
              </div>
              <div className="text-right">
                <p className="text-2xl font-semibold text-white">&euro;{totalCost}</p>
                <p className="text-xs text-neutral-500">per month</p>
              </div>
            </div>
          </div>
        </section>

        {/* Resource Usage Breakdown */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Resource Usage</h2>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Resource</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Usage</th>
                  <th className="px-4 py-2.5 text-right text-xs font-medium text-neutral-500">Cost</th>
                </tr>
              </thead>
              <tbody>
                {resourceUsage.map((item) => (
                  <tr key={item.resource} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{item.resource}</td>
                    <td className="px-4 py-3 text-neutral-400">{item.quantity}</td>
                    <td className="px-4 py-3 text-right font-mono text-xs text-neutral-300">&euro;{item.cost}</td>
                  </tr>
                ))}
                <tr className="bg-surface-100">
                  <td className="px-4 py-3 font-medium text-white">Total</td>
                  <td className="px-4 py-3"></td>
                  <td className="px-4 py-3 text-right font-mono text-sm font-semibold text-white">&euro;{totalCost}/mo</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Billing Info */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Billing Information</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-surface-300">
                  <svg className="h-4.5 w-4.5 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                  </svg>
                </div>
                <div>
                  <p className="text-sm text-neutral-300">Visa ending in 4242</p>
                  <p className="text-xs text-neutral-500">Expires 12/2027</p>
                </div>
              </div>
              <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
                Update
              </button>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
