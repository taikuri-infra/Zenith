"use client";

import { useState, useRef } from "react";
import { motion, useInView } from "framer-motion";
import { Calculator, Server, Database, HardDrive } from "lucide-react";

interface SliderProps {
  label: string;
  icon: typeof Server;
  value: number;
  min: number;
  max: number;
  step: number;
  unit: string;
  onChange: (v: number) => void;
}

function Slider({ label, icon: Icon, value, min, max, step, unit, onChange }: SliderProps) {
  const percentage = ((value - min) / (max - min)) * 100;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500/10 border border-accent-500/15">
            <Icon className="h-3.5 w-3.5 text-accent-400" />
          </div>
          <span className="text-sm font-medium text-neutral-300">{label}</span>
        </div>
        <span className="text-sm font-semibold text-white tabular-nums">
          {value} {unit}
        </span>
      </div>
      <div className="relative">
        <div className="h-1.5 rounded-full bg-surface-300">
          <div
            className="h-full rounded-full bg-accent-500 transition-all duration-150"
            style={{ width: `${percentage}%` }}
          />
        </div>
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          className="absolute inset-0 w-full cursor-pointer opacity-0"
        />
      </div>
    </div>
  );
}

function estimateHetznerCost(apps: number, dbs: number, storage: number): number {
  // CX22 (2 vCPU, 4 GB) = 4.51 EUR/mo for management
  const management = 4.51;
  // Workers: ~1 CX22 per 5 apps
  const workerNodes = Math.ceil(apps / 5);
  const workers = workerNodes * 4.51;
  // DB nodes: ~1 CX22 per 2 databases
  const dbNodes = Math.ceil(dbs / 2);
  const dbCost = dbNodes * 4.51;
  // Storage: Hetzner volumes ~0.052 EUR/GB/mo
  const storageCost = storage * 0.052;

  return Math.round((management + workers + dbCost + storageCost) * 100) / 100;
}

function estimateAWSCost(apps: number, dbs: number, storage: number): number {
  // t3.medium per 3 apps at ~$35/mo each
  const compute = Math.ceil(apps / 3) * 35;
  // RDS db.t3.medium per DB at ~$50/mo each
  const database = dbs * 50;
  // S3 storage at ~$0.023/GB/mo + transfer
  const storageCost = storage * 0.023 + apps * 5; // data transfer estimate
  // ALB + misc services
  const misc = 25 + apps * 2;

  return Math.round(compute + database + storageCost + misc);
}

export function CostCalculator() {
  const [apps, setApps] = useState(5);
  const [dbs, setDbs] = useState(3);
  const [storage, setStorage] = useState(50);

  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-80px" });

  const zenithCost = estimateHetznerCost(apps, dbs, storage);
  const awsCost = estimateAWSCost(apps, dbs, storage);
  const savings = Math.round(((awsCost - zenithCost) / awsCost) * 100);

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.6 }}
      className="mx-auto max-w-2xl"
    >
      <div className="rounded-2xl border border-border bg-surface-50/50 p-6 md:p-8">
        <div className="mb-8 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent-500/10 border border-accent-500/15">
            <Calculator className="h-5 w-5 text-accent-400" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-white">Cost Calculator</h3>
            <p className="text-xs text-neutral-500">Estimate your monthly infrastructure cost</p>
          </div>
        </div>

        {/* Sliders */}
        <div className="space-y-6 mb-8">
          <Slider
            label="Applications"
            icon={Server}
            value={apps}
            min={1}
            max={50}
            step={1}
            unit="apps"
            onChange={setApps}
          />
          <Slider
            label="Databases"
            icon={Database}
            value={dbs}
            min={0}
            max={20}
            step={1}
            unit="databases"
            onChange={setDbs}
          />
          <Slider
            label="Storage"
            icon={HardDrive}
            value={storage}
            min={10}
            max={500}
            step={10}
            unit="GB"
            onChange={setStorage}
          />
        </div>

        {/* Results */}
        <div className="grid gap-4 sm:grid-cols-2">
          {/* Zenith cost */}
          <div className="rounded-xl border border-accent-500/25 bg-accent-500/5 p-5">
            <p className="text-xs font-medium uppercase tracking-wider text-accent-400 mb-1">
              Zenith on Hetzner
            </p>
            <div className="flex items-baseline gap-1">
              <span className="text-3xl font-bold text-accent-400">
                {"\u20AC"}{zenithCost.toFixed(0)}
              </span>
              <span className="text-sm text-neutral-500">/mo</span>
            </div>
          </div>

          {/* AWS cost */}
          <div className="rounded-xl border border-border bg-surface-200/50 p-5">
            <p className="text-xs font-medium uppercase tracking-wider text-neutral-500 mb-1">
              AWS equivalent
            </p>
            <div className="flex items-baseline gap-1">
              <span className="text-3xl font-bold text-neutral-400 line-through decoration-red-500/50">
                ${awsCost}
              </span>
              <span className="text-sm text-neutral-500">/mo</span>
            </div>
          </div>
        </div>

        {/* Savings badge */}
        <div className="mt-4 text-center">
          <span className="inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-sm">
            <span className="font-semibold text-accent-400">{savings}% cheaper</span>
            <span className="text-neutral-500">with Zenith</span>
          </span>
        </div>
      </div>
    </motion.div>
  );
}
