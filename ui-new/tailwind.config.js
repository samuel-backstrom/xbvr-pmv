/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        surface: {
          50: '#f0f1f5',
          100: '#d4d7e3',
          200: '#a8aec6',
          300: '#7c85aa',
          400: '#515c8d',
          500: '#363f6b',
          600: '#2a3155',
          700: '#1e2440',
          800: '#13172b',
          900: '#0a0d1a',
          950: '#060810',
        },
        accent: {
          DEFAULT: '#8b5cf6',
          hover: '#a78bfa',
          muted: '#7c3aed',
          glow: 'rgba(139, 92, 246, 0.3)',
        },
        cyber: {
          blue: '#38bdf8',
          teal: '#2dd4bf',
          pink: '#f472b6',
          amber: '#fbbf24',
          red: '#f87171',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      backdropBlur: {
        xs: '2px',
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.25s ease-out',
        'glow-pulse': 'glowPulse 2s ease-in-out infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(12px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(12px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        glowPulse: {
          '0%, 100%': { boxShadow: '0 0 20px rgba(139, 92, 246, 0.1)' },
          '50%': { boxShadow: '0 0 30px rgba(139, 92, 246, 0.25)' },
        },
      },
    },
  },
  plugins: [],
}
