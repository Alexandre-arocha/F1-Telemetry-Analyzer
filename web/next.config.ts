import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // small image for the web Dockerfile; Vercel builds its own output
  output: process.env.VERCEL ? undefined : "standalone",
};

export default nextConfig;
