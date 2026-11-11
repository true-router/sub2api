import plugin from 'tailwindcss/plugin'

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'background-success-default': 'rgb(var(--boardui-background-success-default) / <alpha-value>)',
        'background-warning-default': 'rgb(var(--boardui-background-warning-default) / <alpha-value>)',
        'background-error-default': 'rgb(var(--boardui-background-error-default) / <alpha-value>)',
        'background-info-default': 'rgb(var(--boardui-background-info-default) / <alpha-value>)',
        'background-feature-default': 'rgb(var(--boardui-background-feature-default) / <alpha-value>)',
        'text-primary': 'rgb(var(--boardui-text-primary) / <alpha-value>)',
        'text-secondary': 'rgb(var(--boardui-text-secondary) / <alpha-value>)',
        'text-tertiary': 'rgb(var(--boardui-text-tertiary) / <alpha-value>)',
        'text-placeholder': 'rgb(var(--boardui-text-placeholder) / <alpha-value>)',
        'text-disabled': 'rgb(var(--boardui-text-disabled) / <alpha-value>)',
        'text-white': 'rgb(var(--boardui-text-white) / <alpha-value>)',
        'text-accent': 'rgb(var(--boardui-text-accent) / <alpha-value>)',
        'text-error-primary': 'rgb(var(--boardui-text-error-primary) / <alpha-value>)',
        'text-success': 'rgb(var(--boardui-text-success) / <alpha-value>)',
        'text-warning': 'rgb(var(--boardui-text-warning) / <alpha-value>)',
        'text-info': 'rgb(var(--boardui-text-info) / <alpha-value>)',
        'text-feature': 'rgb(var(--boardui-text-feature) / <alpha-value>)',
        'background-full': 'rgb(var(--boardui-background-full) / <alpha-value>)',
        'background-primary-default': 'rgb(var(--boardui-background-primary-default) / <alpha-value>)',
        'background-primary-hover': 'rgb(var(--boardui-background-primary-hover) / <alpha-value>)',
        'background-primary-active': 'rgb(var(--boardui-background-primary-active) / <alpha-value>)',
        'background-secondary-default': 'rgb(var(--boardui-background-secondary-default) / <alpha-value>)',
        'background-secondary-hover': 'rgb(var(--boardui-background-secondary-hover) / <alpha-value>)',
        'background-tertiary-default': 'rgb(var(--boardui-background-tertiary-default) / <alpha-value>)',
        'background-quaternary-default': 'rgb(var(--boardui-background-quaternary-default) / <alpha-value>)',
        'background-accent-default': 'rgb(var(--boardui-background-accent-default) / <alpha-value>)',
        'background-accent-hover': 'rgb(var(--boardui-background-accent-hover) / <alpha-value>)',
        'background-accent-subtle': 'rgb(var(--boardui-background-accent-subtle) / <alpha-value>)',
        'background-success-subtle': 'rgb(var(--boardui-background-success-subtle) / <alpha-value>)',
        'background-warning-subtle': 'rgb(var(--boardui-background-warning-subtle) / <alpha-value>)',
        'background-error-subtle': 'rgb(var(--boardui-background-error-subtle) / <alpha-value>)',
        'background-info-subtle': 'rgb(var(--boardui-background-info-subtle) / <alpha-value>)',
        'background-feature-subtle': 'rgb(var(--boardui-background-feature-subtle) / <alpha-value>)',
        'background-overlay': 'rgb(var(--boardui-background-overlay) / <alpha-value>)',
        'border-button-default': 'rgb(var(--boardui-border-button-default) / <alpha-value>)',
        'border-button-hover': 'rgb(var(--boardui-border-button-hover) / <alpha-value>)',
        'border-button-active': 'rgb(var(--boardui-border-button-active) / <alpha-value>)',
        'border-focus-ring': 'rgb(var(--boardui-border-focus-ring) / <alpha-value>)',
        'border-error-default': 'rgb(var(--boardui-border-error-default) / <alpha-value>)',
        'border-table': 'rgb(var(--boardui-border-table) / <alpha-value>)',
        'separator-border': 'rgb(var(--boardui-separator-border) / <alpha-value>)',
        'foreground-icon-primary': 'rgb(var(--boardui-foreground-icon-primary) / <alpha-value>)',
        'foreground-icon-secondary': 'rgb(var(--boardui-foreground-icon-secondary) / <alpha-value>)',
        'foreground-icon-tertiary': 'rgb(var(--boardui-foreground-icon-tertiary) / <alpha-value>)',
        'foreground-icon-quaternary': 'rgb(var(--boardui-foreground-icon-quaternary) / <alpha-value>)',
        'foreground-full': 'rgb(var(--boardui-foreground-full) / <alpha-value>)',
        'chart-1': 'rgb(var(--boardui-chart-1) / <alpha-value>)',
        'chart-2': 'rgb(var(--boardui-chart-2) / <alpha-value>)',
        'chart-3': 'rgb(var(--boardui-chart-3) / <alpha-value>)',
        'chart-4': 'rgb(var(--boardui-chart-4) / <alpha-value>)',
        'chart-5': 'rgb(var(--boardui-chart-5) / <alpha-value>)',
        'chart-track': 'rgb(var(--boardui-chart-track) / <alpha-value>)',
        // 主色调 - Teal/Cyan 青色系
        primary: {
          50: '#f0fdfa',
          100: '#ccfbf1',
          200: '#99f6e4',
          300: '#5eead4',
          400: '#2dd4bf',
          500: '#14b8a6',
          600: '#0d9488',
          700: '#0f766e',
          800: '#115e59',
          900: '#134e4a',
          950: '#042f2e'
        },
        // 辅助色 - 深蓝灰
        accent: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617'
        },
        // 深色模式背景
        dark: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617'
        }
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 8px 32px rgba(0, 0, 0, 0.08)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.06)',
        glow: '0 0 20px rgba(20, 184, 166, 0.25)',
        'glow-lg': '0 0 40px rgba(20, 184, 166, 0.35)',
        card: '0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.06)',
        'card-hover': '0 10px 40px rgba(0, 0, 0, 0.08)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #14b8a6 0%, #0d9488 100%)',
        'gradient-dark': 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(20, 184, 166, 0.12) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(6, 182, 212, 0.08) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(20, 184, 166, 0.08) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(20, 184, 166, 0.25)' },
          '100%': { boxShadow: '0 0 30px rgba(20, 184, 166, 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: [
    plugin(({ addUtilities }) => {
      const styles = {
        'display-4': ['40px', '48px', '-1px'],
        'title-1': ['24px', '32px', '-0.5px'],
        'title-2': ['20px', '28px', '-0.4px'],
        'title-3': ['18px', '26px', '-0.3px'],
        headline: ['16px', '24px', '-0.2px'],
        body: ['14px', '20px', '-0.1px'],
        'body-2': ['13px', '20px', '0px'],
        'caption-1': ['12px', '16px', '0px'],
        'caption-2': ['11px', '16px', '0px']
      }
      const weights = { regular: '400', medium: '500', semibold: '600', bold: '700' }
      addUtilities(Object.fromEntries(Object.entries(styles).flatMap(([family, [fontSize, lineHeight, letterSpacing]]) =>
        Object.entries(weights).map(([weight, fontWeight]) => [`.text-${family}-${weight}`, { fontSize, lineHeight, letterSpacing, fontWeight }])
      )))
    })
  ]
}
