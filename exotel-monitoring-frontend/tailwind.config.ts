import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  corePlugins: {
    preflight: false,
  },
  important: false,
  theme: {
    extend: {},
  },
  plugins: [],
};

export default config;
