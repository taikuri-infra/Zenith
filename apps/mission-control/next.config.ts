import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  transpilePackages: ["@zenith/ui"],
  eslint: {
    ignoreDuringBuilds: true,
  },
};

export default nextConfig;
