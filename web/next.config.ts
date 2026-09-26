import type { NextConfig } from 'next'

const goServer = 'http://localhost:8080'

const config: NextConfig = {
  output: 'export',
  distDir: 'dist',
  images: { unoptimized: true },
  agentRules: false,
  ...(process.env.NODE_ENV === 'development' && {
    rewrites: async () => [{ source: '/ws', destination: `${goServer}/ws` }],
  }),
}

export default config
