// A static file server for the browser tests that mimics GitHub Pages, so the
// tests fail on the same broken links a visitor would hit in production:
//
//   /foo          -> build/foo.html   (extensionless URLs resolve to .html)
//   /dir/         -> build/dir/index.html
//   anything else -> build/404.html, served with a 404 status
//
// Deliberately NOT the generator's own `--watch` server: that one injects a
// livereload <script> and an SSE connection, which would show up as extra
// network traffic in the "zero external requests" and console-error checks.
//
// Dependency-free — node:http and node:fs only. Import `startServer` to run it
// in-process (see shot.mjs), or run this file directly for a standalone server.

import { createServer } from "node:http";
import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
export const buildDir = path.resolve(here, "../../build");

const contentTypes = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".xml": "application/xml; charset=utf-8",
  ".txt": "text/plain; charset=utf-8",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
};

async function isFile(p) {
  try {
    return (await stat(p)).isFile();
  } catch {
    return false;
  }
}

async function isDir(p) {
  try {
    return (await stat(p)).isDirectory();
  } catch {
    return false;
  }
}

// resolvePath maps a request path to a file on disk using GitHub Pages' rules.
async function resolvePath(urlPath) {
  let decoded;
  try {
    decoded = decodeURIComponent(urlPath);
  } catch {
    return null;
  }
  const full = path.join(buildDir, path.normalize(decoded));
  // Never serve anything outside build/, whatever the URL claims.
  if (full !== buildDir && !full.startsWith(buildDir + path.sep)) return null;

  if (await isDir(full)) {
    const index = path.join(full, "index.html");
    return (await isFile(index)) ? index : null;
  }
  if (await isFile(full)) return full;
  if (!path.extname(full) && (await isFile(full + ".html"))) return full + ".html";
  return null;
}

async function handle(req, res) {
  const urlPath = new URL(req.url, "http://localhost").pathname;
  const file = await resolvePath(urlPath);

  if (!file) {
    const notFound = path.join(buildDir, "404.html");
    if (await isFile(notFound)) {
      res.writeHead(404, { "Content-Type": contentTypes[".html"] });
      res.end(await readFile(notFound));
      return;
    }
    res.writeHead(404, { "Content-Type": contentTypes[".txt"] });
    res.end("404 not found\n");
    return;
  }

  res.writeHead(200, {
    "Content-Type": contentTypes[path.extname(file)] || "application/octet-stream",
    "Cache-Control": "no-store",
  });
  res.end(await readFile(file));
}

// startServer listens on `port` (0 picks a free one) and resolves with the
// server plus its base URL. Call `close()` when you're done with it.
export function startServer({ port = 0, host = "127.0.0.1" } = {}) {
  return new Promise((resolve, reject) => {
    const server = createServer((req, res) => {
      handle(req, res).catch((err) => {
        res.writeHead(500, { "Content-Type": contentTypes[".txt"] });
        res.end(`server error: ${err.message}\n`);
      });
    });
    server.once("error", reject);
    server.listen(port, host, () => {
      const { port: actual } = server.address();
      resolve({
        server,
        baseURL: `http://${host}:${actual}`,
        close: () => new Promise((done) => server.close(done)),
      });
    });
  });
}

// Run directly (`node serve.mjs`) for a standalone server on PORT, default 4321.
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const { baseURL } = await startServer({ port: Number(process.env.PORT || 4321) });
  console.log(`serving ${buildDir} on ${baseURL}`);
}
