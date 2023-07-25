import { defineConfig } from "astro/config";

import tailwind from "@astrojs/tailwind";
import image from "@astrojs/image";

export default defineConfig({
  site: "https://letsbuild.cloud",
  integrations: [
    tailwind({
      applyBaseStyles: false,
    }),
    image(),
  ],
  experimental: {
    viewTransitions: true,
  },
});
