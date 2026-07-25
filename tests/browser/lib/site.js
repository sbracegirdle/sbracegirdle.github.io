// Discovery helpers for the specs. The generator decides what pages exist, so
// the tests read build/ rather than hard-coding a list that drifts every time a
// post is added. `npm test` regenerates build/ (pretest) before Playwright
// collects the specs, so this is always current.

import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { buildDir } from "../serve.mjs";

export { buildDir };

// htmlPages returns every generated page as a root-absolute URL path.
export function htmlPages() {
  const out = [];
  const walk = (dir, prefix) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.isDirectory()) walk(path.join(dir, entry.name), `${prefix}${entry.name}/`);
      else if (entry.name.endsWith(".html")) out.push(`${prefix}${entry.name}`);
    }
  };
  walk(buildDir, "/");
  return out.sort();
}

const datedPost = /^\/\d{4}-\d{2}-\d{2}-.+\.html$/;

// postPages returns the dated blog posts, newest first.
export function postPages() {
  return htmlPages().filter((p) => datedPost.test(p)).reverse();
}

// tagPages returns the per-tag listing pages under /tags/.
export function tagPages() {
  return htmlPages().filter((p) => p.startsWith("/tags/"));
}

export function read(urlPath) {
  return readFileSync(path.join(buildDir, urlPath.replace(/^\//, "")), "utf8");
}

// aTaggedPost picks a post that actually carries tag chips, so the tag
// assertions don't silently pass against an untagged post.
export function aTaggedPost() {
  const found = postPages().find((p) => read(p).includes('class="tag-list"'));
  if (!found) throw new Error("no post with tag chips found in build/ — did tag generation break?");
  return found;
}

// A representative sample: home, archive, tag index, one tag page, one post,
// plus the standalone static pages. Cheaper than every page for checks that
// don't need full coverage.
export function samplePages() {
  return [
    "/index.html",
    "/posts.html",
    "/tags.html",
    tagPages()[0],
    aTaggedPost(),
    "/about.html",
    "/style-guide.html",
    "/rust-quick-reference.html",
    "/sports.html",
    "/404.html",
  ].filter(Boolean);
}
