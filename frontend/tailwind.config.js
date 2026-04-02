/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Sandstone palette
        brand: {
          DEFAULT: '#92400e', // amber-800 — primary brand
          light: '#d97706',   // amber-600 — accent
        },
      },
      fontFamily: {
        sans: [
          'Inter', 'ui-sans-serif', 'system-ui', '-apple-system',
          'BlinkMacSystemFont', 'Segoe UI', 'sans-serif',
        ],
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
