import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
  // Disable overly strict rules for data fetching patterns
  {
    rules: {
      // Allow async data fetching in useEffect - common pattern for initial data loading
      "react-hooks/set-state-in-effect": "off",
      // Allow variables to be accessed in callbacks before declaration (hoisting)
      "react-hooks/immutability": "off",
    },
  },
]);

export default eslintConfig;
