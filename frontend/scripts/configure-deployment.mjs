import { cp, mkdir, writeFile } from "node:fs/promises";

const apiBase = (process.env.NIMMA_API_BASE || "").trim();
const frontendRoot = new URL("../", import.meta.url);
const outputDirectory = new URL("../dist/", import.meta.url);

if (apiBase) {
  let parsed;
  try {
    parsed = new URL(apiBase);
  } catch {
    throw new Error("NIMMA_API_BASE must be an absolute HTTPS API URL.");
  }
  if (parsed.protocol !== "https:" || parsed.search || parsed.hash || !parsed.pathname.replace(/\/+$/, "").endsWith("/api/v1")) {
    throw new Error("NIMMA_API_BASE must be an HTTPS URL ending in /api/v1, without query or fragment.");
  }
}

await writeFile(
  new URL("deployment-config.js", frontendRoot),
  `window.NIMMA_API_BASE = ${JSON.stringify(apiBase)};\n`,
  "utf8",
);

await mkdir(outputDirectory, { recursive: true });
for (const directory of ["assets", "pages", "public", "scripts", "styles"]) {
  await cp(new URL(`${directory}/`, frontendRoot), new URL(`${directory}/`, outputDirectory), { recursive: true });
}
for (const file of ["deployment-config.js", "index.html", "portal.html"]) {
  await cp(new URL(file, frontendRoot), new URL(file, outputDirectory));
}
