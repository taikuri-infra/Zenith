"use client";

import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import { Check, X, ArrowRight } from "lucide-react";

const providers = [
  {
    name: "Zenith",
    tagline: "on Hetzner",
    price: "~\u20AC20",
    highlight: true,
  },
  {
    name: "AWS",
    tagline: "ECS + RDS",
    price: "$580+",
    highlight: false,
  },
  {
    name: "GCP",
    tagline: "GKE + Cloud SQL",
    price: "$520+",
    highlight: false,
  },
  {
    name: "Heroku",
    tagline: "Standard",
    price: "$450+",
    highlight: false,
  },
  {
    name: "Railway",
    tagline: "Pro plan",
    price: "$300+",
    highlight: false,
  },
];

const features = [
  { name: "10 apps with custom domains", zenith: true, aws: true, gcp: true, heroku: true, railway: true },
  { name: "5 managed databases", zenith: true, aws: true, gcp: true, heroku: true, railway: true },
  { name: "100GB object storage", zenith: true, aws: true, gcp: true, heroku: false, railway: false },
  { name: "Built-in auth service", zenith: true, aws: false, gcp: false, heroku: false, railway: false },
  { name: "API gateway included", zenith: true, aws: true, gcp: true, heroku: false, railway: false },
  { name: "Full monitoring stack", zenith: true, aws: true, gcp: true, heroku: false, railway: false },
  { name: "No vendor lock-in", zenith: true, aws: false, gcp: false, heroku: false, railway: false },
  { name: "Open source (MIT)", zenith: true, aws: false, gcp: false, heroku: false, railway: false },
];

export function PricingComparison() {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  return (
    <div ref={ref}>
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={isInView ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.6 }}
        className="overflow-x-auto rounded-2xl border border-border"
      >
        <table className="w-full min-w-[700px] text-sm">
          <thead>
            <tr className="border-b border-border bg-surface-50/50">
              <th className="p-4 text-left text-sm font-medium text-neutral-400 w-[200px]">
                Scenario: 10 apps, 5 databases, 100GB storage
              </th>
              {providers.map((p) => (
                <th key={p.name} className="p-4 text-center">
                  <div
                    className={`inline-flex flex-col items-center rounded-lg px-3 py-2 ${
                      p.highlight
                        ? "bg-accent-500/10 border border-accent-500/20"
                        : ""
                    }`}
                  >
                    <span
                      className={`text-sm font-semibold ${
                        p.highlight ? "text-accent-400" : "text-white"
                      }`}
                    >
                      {p.name}
                    </span>
                    <span className="text-[11px] text-neutral-500">
                      {p.tagline}
                    </span>
                  </div>
                </th>
              ))}
            </tr>
            {/* Price row */}
            <tr className="border-b border-border">
              <td className="p-4 text-sm font-medium text-neutral-300">
                Monthly cost
              </td>
              {providers.map((p) => (
                <td key={p.name} className="p-4 text-center">
                  <span
                    className={`text-lg font-bold ${
                      p.highlight ? "text-accent-400" : "text-white"
                    }`}
                  >
                    {p.price}
                  </span>
                  <span className="text-xs text-neutral-500">/mo</span>
                </td>
              ))}
            </tr>
          </thead>
          <tbody>
            {features.map((f, i) => {
              const values = [f.zenith, f.aws, f.gcp, f.heroku, f.railway];
              return (
                <tr
                  key={f.name}
                  className={`border-b border-border/50 ${
                    i % 2 === 0 ? "bg-surface-50/20" : ""
                  }`}
                >
                  <td className="p-4 text-sm text-neutral-300">{f.name}</td>
                  {values.map((v, j) => (
                    <td key={j} className="p-4 text-center">
                      {v ? (
                        <div className={`inline-flex h-5 w-5 items-center justify-center rounded-full ${j === 0 ? "bg-accent-500/15" : ""}`}>
                          <Check
                            className={`h-3.5 w-3.5 ${
                              j === 0 ? "text-accent-400" : "text-neutral-500"
                            }`}
                          />
                        </div>
                      ) : (
                        <X className="mx-auto h-3.5 w-3.5 text-neutral-700" />
                      )}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </motion.div>

      {/* Savings callout */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={isInView ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.5, delay: 0.3 }}
        className="mt-6 flex items-center justify-center"
      >
        <div className="inline-flex items-center gap-3 rounded-full border border-accent-500/20 bg-accent-500/5 px-5 py-2.5">
          <ArrowRight className="h-4 w-4 text-accent-400" />
          <span className="text-sm text-neutral-300">
            Save up to <span className="font-semibold text-accent-400">96%</span> compared to AWS for the same workload
          </span>
        </div>
      </motion.div>
    </div>
  );
}
