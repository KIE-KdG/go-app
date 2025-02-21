/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./ui/**/*.{html,js}'],
  plugins: [
    require('daisyui'),
  ],
  daisyui: {
    themes: ["dark", "cupcake"],
  },
}