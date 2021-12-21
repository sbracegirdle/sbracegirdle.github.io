module.exports = {
  content: ["./**/*.html"],
  theme: {
    fontSize: {
      "2xs": "0.75rem",
      xs: "0.875rem",
      sm: "1rem",
      base: "1.2rem",
      lg: "1.25rem",
      xl: "1.5rem",
      "2xl": "1.875rem",
      "3xl": "2.25rem",
      "4xl": "3rem",
      "5xl": "4rem",
      "6xl": "5rem",
    },
    fontFamily: {
      sans: ["Lato", "sans-serif"],
      serif: ["Merriweather", "serif"],
    },
    extend: {
      colors: {
        flickr: {
          darker: "#A90E58",
          dark: "#C71269",
          default: "#F02385",
          light: "#F855A0",
          lighter: "#FE74B3",
        },
        purpler: {
          default: "#7209b7",
        },
      },
    },
  },
  plugins: [],
};
