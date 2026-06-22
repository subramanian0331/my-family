/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        brand: {
          green: "#2d7a5f",
          teal: "#2a9d8f",
          blue: "#1e6b8f",
          lime: "#7cb342",
          brown: "#8d6e4a",
          cream: "#f4faf7",
          mist: "#e8f4fa",
          leaf: "#d4ebe3",
        },
        accent: {
          DEFAULT: "#2a9d8f",
          hover: "#238276",
        },
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
      },
    },
  },
  plugins: [],
};