import { readdir, readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const projectRoot = path.resolve(frontendRoot, "..");
const scanRoots = [
  path.join(frontendRoot, "src"),
  path.join(projectRoot, "internal"),
];
const allowedExtensions = new Set([".ts", ".tsx", ".go"]);

async function collectFiles(root) {
  const results = [];
  const entries = await readdir(root, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      results.push(...(await collectFiles(fullPath)));
      continue;
    }
    if (allowedExtensions.has(path.extname(entry.name))) {
      results.push(fullPath);
    }
  }
  return results;
}

function relative(filePath) {
  return path.relative(projectRoot, filePath) || filePath;
}

function collectTypeVersionFindings(content, filePath) {
  const findings = [];
  const pattern = /\b(?:type|interface|class|struct)\s+([A-Za-z0-9_]*(?:V1|V2|v1|v2))\b/g;
  for (const match of content.matchAll(pattern)) {
    findings.push(`${relative(filePath)}: version suffix in DTO/type name \`${match[1]}\``);
  }
  return findings;
}

function collectLegacyFieldFindings(content, filePath) {
  const findings = [];
  const pattern = /\b(?:ManifestJSON|CompatibilityJSON|manifestJson|compatibilityJson)\b/g;
  for (const match of content.matchAll(pattern)) {
    findings.push(`${relative(filePath)}: legacy field \`${match[0]}\``);
  }
  return findings;
}

function collectStoreImportFindings(content, filePath) {
  const findings = [];
  const storePattern = /from\s+["']@\/shared\/store\/(library|connectors|skills)["']/g;
  for (const match of content.matchAll(storePattern)) {
    findings.push(`${relative(filePath)}: transport contract imported from store path \`${match[1]}\``);
  }

  const settingsPattern = /import\s+(?:type\s+)?\{([^}]*)\}\s+from\s+["']@\/shared\/store\/settings["']/gs;
  for (const match of content.matchAll(settingsPattern)) {
    const specifiers = match[1]
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean);
    const invalid = specifiers.filter((value) => value !== "useSettingsStore");
    if (invalid.length > 0) {
      findings.push(
        `${relative(filePath)}: settings transport contract imported from store path \`${invalid.join(", ")}\``,
      );
    }
  }
  return findings;
}

async function main() {
  const files = (await Promise.all(scanRoots.map((root) => collectFiles(root)))).flat();
  const findings = [];

  for (const filePath of files) {
    const content = await readFile(filePath, "utf8");
    findings.push(...collectTypeVersionFindings(content, filePath));
    findings.push(...collectLegacyFieldFindings(content, filePath));
    findings.push(...collectStoreImportFindings(content, filePath));
  }

  if (findings.length > 0) {
    console.error("DTO audit failed:");
    for (const finding of findings) {
      console.error(`- ${finding}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log("DTO audit passed: no V1/V2 DTO names, legacy patch fields, or store-path transport imports found.");
}

await main();
