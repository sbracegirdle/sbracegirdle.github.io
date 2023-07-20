/**
 * Map post file path to Jekyll compatible url.
 *
 * This is to avoid breaking my links transitioning from Jekyll to Astro.
 */
export const postFileToUrl = (file: string) => {
  const url = file.split("/").slice(-1)[0];

  let path = url.replace(/^(?:\d{4})-(?:\d{2})-(?:\d{2})-/, (match) => {
    return match.replace(/-/g, "/");
  });

  path = path.replace(/\.md$/, "");
  return path;
};
