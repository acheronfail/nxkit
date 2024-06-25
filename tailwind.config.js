/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      boxShadow: {
        'inset-border': 'inset 0 0 0 1px',
      },
    },
  },
  plugins: [],
};
