/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#0F6AF2',
        red: '#EB3737',
        green: '#389F3D',
        purple: '#8B4AF5',
        pink: '#CD4AF5',
        orange: '#FA9005',
        yellow: '#FCC90D',
        cyan: '#22B8D3',
        'gray-900': '#000000',
        'gray-700': '#5C637A',
        'gray-500': '#9A9EAD',
        'gray-300': '#D8D9DF',
        'gray-100': '#F4F4F6',
      },
    },
  },
  plugins: [],
}
