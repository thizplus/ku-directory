/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        'db': ['DB', 'system-ui', 'sans-serif'],
        'sans': ['DB', 'system-ui', 'sans-serif'],
      },
      fontSize: {
        'base': ['18px', '1.6'], // เปลี่ยน base จาก 16px เป็น 18px
      }
    },
  },
  plugins: [],
}