import { readdir, readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const projectRoot = path.resolve(frontendRoot, "..");
const localeDir = path.join(frontendRoot, "src", "shared", "i18n", "locales");
const scanRoots = [
  {
    root: path.join(frontendRoot, "src"),
    thresholds: {
      ".ts": 1200,
      ".tsx": 1500,
    },
  },
  {
    root: path.join(projectRoot, "internal"),
    thresholds: {
      ".go": 1400,
    },
  },
];
const allowedExtensions = new Set([".ts", ".tsx", ".go"]);
const oversizeBaseline = {
  "frontend/src/features/library/components/LibraryConfigPage.tsx": 7000,
  "frontend/src/features/library/index.tsx": 5000,
  "frontend/src/features/library/components/LibraryWorkspacePage.tsx": 4100,
  "frontend/src/features/cron/CronPage.tsx": 4400,
  "frontend/src/features/settings/calls/components/CallsToolsTab.tsx": 2100,
  "frontend/src/features/library/components/SubtitleStylePresetManager.tsx": 2200,
  "frontend/src/features/library/components/TaskDialog.tsx": 1850,
  "frontend/src/features/setup/SetupCenterDialog.tsx": 2000,
  "frontend/src/features/settings/integration/ChannelsSection.tsx": 1600,
  "frontend/src/features/settings/provider/index.tsx": 1600,
  "frontend/src/shared/contracts/library.ts": 1300,
  "internal/application/channels/telegram/bot_service.go": 5000,
  "internal/application/gateway/tools/browser_tools.go": 3150,
  "internal/application/browsercdp/session.go": 3362,
  "internal/application/library/service/service.go": 2750,
  "internal/application/gateway/cron/scheduler.go": 2650,
  "internal/application/memory/service/service.go": 2300,
  "internal/application/gateway/tools/skills_tools.go": 1900,
  "internal/application/gateway/tools/builtin_specs.go": 1850,
  "internal/application/gateway/tools/web_tools.go": 1700,
  "internal/app/bootstrap.go": 1700,
  "internal/application/gateway/subagent/service.go": 1650,
  "internal/application/gateway/tools/message_tools.go": 1500,
  "internal/application/externaltools/service/service.go": 2550,
  "internal/application/gateway/tools/image_tools.go": 1500,
  "internal/application/thread/service/service.go": 1600,
};
const localeKeyPattern = /^[a-z0-9][A-Za-z0-9_-]*(?:\.[a-z0-9][A-Za-z0-9_-]*)*$/;

async function collectFiles(root) {
  const results = [];
  const entries = await readdir(root, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      if (fullPath.includes(`${path.sep}bindings`)) {
        continue;
      }
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
  return path.relative(projectRoot, filePath).split(path.sep).join("/");
}

function countLines(content) {
  return content.split(/\r?\n/).length;
}

async function collectOversizeFindings() {
  const findings = [];

  for (const scanRoot of scanRoots) {
    const files = await collectFiles(scanRoot.root);
    for (const filePath of files) {
      const extension = path.extname(filePath);
      const threshold = scanRoot.thresholds[extension];
      if (!threshold) {
        continue;
      }
      const content = await readFile(filePath, "utf8");
      const lineCount = countLines(content);
      if (lineCount <= threshold) {
        continue;
      }
      const relativePath = relative(filePath);
      const baseline = oversizeBaseline[relativePath];
      if (baseline == null) {
        findings.push(
          `${relativePath}: ${lineCount} lines exceeds ${threshold}-line threshold and is not in the oversize baseline`,
        );
        continue;
      }
      if (lineCount > baseline) {
        findings.push(`${relativePath}: ${lineCount} lines exceeds its oversize baseline cap of ${baseline}`);
      }
    }
  }

  return findings;
}

function flattenLocale(input, prefix = "", output = []) {
  for (const [key, value] of Object.entries(input)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === "object" && !Array.isArray(value)) {
      flattenLocale(value, nextKey, output);
      continue;
    }
    output.push(nextKey);
  }
  return output;
}

async function collectLocaleKeyFindings() {
  const findings = [];
  const localeFiles = await readdir(localeDir);
  for (const fileName of localeFiles) {
    if (!fileName.endsWith(".json")) {
      continue;
    }
    const filePath = path.join(localeDir, fileName);
    const content = JSON.parse(await readFile(filePath, "utf8"));
    const keys = flattenLocale(content);
    for (const key of keys) {
      if (localeKeyPattern.test(key)) {
        continue;
      }
      findings.push(`frontend/src/shared/i18n/locales/${fileName}: invalid locale key \`${key}\``);
    }
  }
  return findings;
}

async function main() {
  const findings = [
    ...(await collectOversizeFindings()),
    ...(await collectLocaleKeyFindings()),
  ];

  if (findings.length > 0) {
    console.error("Structure audit failed:");
    for (const finding of findings) {
      console.error(`- ${finding}`);
    }
    process.exitCode = 1;
    return;
  }

  console.log("Structure audit passed: file-size thresholds and locale key naming rules are within the current baseline.");
}

await main();
