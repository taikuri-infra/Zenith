import { Hero } from "@/components/hero";
import { StackStrip } from "@/components/stack-strip";
import { WhatIs } from "@/components/what-is";
import { Stack } from "@/components/stack";
import { Features } from "@/components/features";
import { Infra } from "@/components/infra";
import { Demo } from "@/components/demo";
import { QuickStart } from "@/components/quickstart";
import { OpenSource } from "@/components/open-source";

export default function Home() {
  return (
    <div className="relative">
      <Hero />
      <StackStrip />
      <WhatIs />
      <Stack />
      <Features />
      <Infra />
      <Demo />
      <QuickStart />
      <OpenSource />
    </div>
  );
}
