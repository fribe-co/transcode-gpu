/** @type {import('next').NextConfig} */

// Internal API URL for server-side rewrites (backend container in Docker)
const INTERNAL_API_URL = process.env.INTERNAL_API_URL || 'http://backend:8080';

const nextConfig = {
  output: 'standalone',
  reactStrictMode: true,
  // Turbopack config (Next.js 16+)
  turbopack: {},
  // Webpack config (for compatibility)
  webpack: (config, { isServer }) => {
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        net: false,
        tls: false,
      };
      // HLS.js support
      config.resolve.alias = {
        ...config.resolve.alias,
        'hls.js': require.resolve('hls.js'),
      };
    }
    return config;
  },
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${INTERNAL_API_URL}/api/:path*`,
      },
      {
        source: '/streams/:path*',
        destination: `${INTERNAL_API_URL}/streams/:path*`,
      },
      {
        source: '/logos/:path*',
        destination: `${INTERNAL_API_URL}/logos/:path*`,
      },
    ];
  },
};

module.exports = nextConfig;

