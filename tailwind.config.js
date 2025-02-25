/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./ui/**/*.{html,js}'],
  plugins: [
    require('daisyui'),
  ],
  theme: {
    container: {
      center: true,
    },
  },
  daisyui: {
    themes: ["dark", "cupcake"],
  },
}