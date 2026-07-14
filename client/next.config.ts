import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./lib/i18n/request.ts");

const nextConfig: NextConfig = {
  // Allow dev HMR / RSC assets when the browser origin is not localhost
  // (e.g. ngrok HTTPS tunnel → next dev :3000).
  allowedDevOrigins: [
    "127.0.0.1",
    "localhost",
    "*.ngrok-free.dev",
    "*.ngrok-free.app",
    "*.ngrok.io",
  ],
  typedRoutes: true,
};

export default withNextIntl(nextConfig);
