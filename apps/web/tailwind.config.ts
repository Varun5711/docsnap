import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        vault: {
          paper: "#f6f4ef",
          ink: "#151716",
          rail: "#e8e3d9",
          seal: "#245d63",
          wax: "#b04e3f",
          ledger: "#706a5d",
          proof: "#d4a017"
        }
      },
      boxShadow: {
        audit: "0 18px 60px rgba(36, 43, 40, 0.10)"
      }
    }
  },
  plugins: []
};

export default config;

