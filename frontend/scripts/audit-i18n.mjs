import fs from "node:fs";
import path from "node:path";
import ts from "typescript";

const rootDir = process.cwd();
const strict = process.argv.includes("--strict");
const jsonOnly = process.argv.includes("--json");
const pruneUnused = process.argv.includes("--prune-unused");

const localeDir = path.join(rootDir, "src", "shared", "i18n", "locales");
const sourceDir = path.join(rootDir, "src");
const englishStyleSkipKeys = new Set([
  "app.name",
  "library.config.subtitleStyles.exportProfileStyleDocument",
  "library.workspace.header.videoEditing",
  "library.workspace.header.subtitleEditing",
  "library.workspace.header.speechToSubtitle",
  "library.workspace.dialogs.exportSubtitle.style",
]);
const englishStylePreservePhrases = [
  "Dream Creator",
  "Apple",
  "macOS",
  "iOS",
  "Chat Completions",
  "OpenAI",
  "OpenRouter",
  "Bun",
  "ClawHub",
  "DreamCreator",
  "Playwright",
  "FFmpeg",
  "FontGet",
  "LibASS",
  "Vidstack",
  "GitHub",
  "YouTube",
  "Bilibili",
  "Telegram",
  "Markdown",
  "WebSocket",
  "JSON Schema",
  "AI",
  "JSON",
  "API",
  "HTTP",
  "HTTPS",
  "URL",
  "PDF",
  "MIME",
  "TTS",
  "ACK",
  "SSE",
  "VRM",
  "VRMA",
  "GLB",
  "ASS",
  "SSA",
  "SRT",
  "VTT",
  "TTML",
  "FCPXML",
  "CPS",
  "CPL",
  "CRF",
  "BOM",
  "UTC",
  "N/A",
  "OS",
  "Nix",
  "Wails",
  "yt-dlp",
  "User-Agent",
  "Accept-Language",
  "Content-Type",
  "Gateway",
  "ElevenLabs",
  "SKILL.md",
  "ClawDBot",
  "ClawDis",
];
const englishStylePreserveWords = new Set(englishStylePreservePhrases.filter((item) => !item.includes(" ")));
const englishStyleAlwaysLowerWords = new Set([
  "a",
  "an",
  "the",
  "and",
  "or",
  "but",
  "for",
  "nor",
  "at",
  "by",
  "to",
  "from",
  "of",
  "in",
  "on",
  "off",
  "up",
  "out",
  "as",
  "is",
  "are",
  "be",
  "been",
  "being",
  "into",
  "over",
  "with",
  "without",
  "per",
  "same",
  "after",
  "before",
  "when",
  "while",
  "that",
  "this",
  "these",
  "those",
  "all",
  "no",
  "not",
  "now",
  "than",
  "once",
  "only",
  "via",
  "max",
  "min",
]);
const englishTitleCaseLowerWords = new Set([
  "a",
  "an",
  "the",
  "and",
  "or",
  "but",
  "nor",
  "for",
  "so",
  "yet",
  "as",
  "at",
  "by",
  "en",
  "in",
  "of",
  "off",
  "on",
  "per",
  "to",
  "up",
  "via",
]);
const englishSentenceBoundaryChars = new Set([".", "!", "?"]);
const englishTitleStyleExplicitPatterns = [
  /^app\.settings\.title\./,
  /^settings\.gateway\.detailsPanel\.sections\./,
  /^settings\.gateway\.detailsPanel\.(contextTabs|httpTabs|talkTabs|heartbeatTabs)\./,
  /^settings\.gateway\.detailsPanel\.(gateway\.controlPlane|runtime\.(maxSteps|toolLoop\.(enabled|warn|critical|global|history|detectorGeneric|detectorPoll|detectorPingPong)|contextWarn|contextHard|compactionMode|compactionReserveTokens|compactionKeepRecent|compactionReserveFloor|compactionMaxHistoryShare|compactionMemoryFlushEnabled|compactionMemoryFlushSoft|compactionMemoryFlushPrompt|compactionMemoryFlushSystem)|queue\.(globalConcurrency|sessionConcurrency|laneMain|laneSubagent|laneCron)|cron\.(enabled|maxConcurrentRuns|sessionRetention|runLogMaxBytes|runLogKeepLines)|heartbeat\.(enabled|periodicEnabled|runSession|notificationCenter|openNotifications|triggerLabel|lastStatus\.label|lastNotice\.label|promptAppend|includeReasoning|suppressToolErrorWarnings|activeStart|activeEnd|activeTimezone|trigger|spec\.items|spec\.updatedAt)|subagents\.(maxDepth|maxChildren|maxConcurrent|model)|http\.(maxBodyBytes|maxUrlParts|files\.(urlAllowlist|allowedMimes|maxBytes|maxChars|maxRedirects|pdfMaxPages|pdfMaxPixels|pdfMinTextChars)|images\.(urlAllowlist|allowedMimes|maxBytes|maxRedirects))|channelHealth\.minutes|voice\.(enabled)|voiceWake\.(enabled)|talk\.(voiceAliases|outputFormat|apiKey|interruptOnSpeech)|tts\.(status|provider|voiceId|modelId|format)|voiceWakeTriggers\.label)$/,
  /^settings\.gateway\.(change3davatar\.pickTitle|change3dmotion\.pickTitle|changeName\.title|readiness\.(readyTitle|incompleteTitle)|model\.(agentTitle|embeddingTitle|imageTitle))$/,
  /^settings\.(memory\.summary\.title|usage\.overview\.title|general\.download\.dialogTitle|general\.proxy\.dialogTitle|general\.advanced\.menuBarVisibility\.label|calls\.skills\.sources\.emptyTitle|calls\.skills\.security\.groups\.(package_write|deps_write|config_write|source_write)\.label|tools\.browserControl\.ssrfSection|skills\.listTitle|connectors\.loginTitle|externalTools\.releaseNotesTitle|provider\.models\.manage\.title|provider\.models\.manage\.invalidTitle|provider\.custom\.title|provider\.delete\.title|about\.advanced\.unlockedTitle)$/,
  /^settings\.integration\.channels\.(config\.groups\.columns\.requireMention|reset\.button)$/,
  /^cron\.(runs\.detailTitle|dialog\.(deleteTitle|bulkDeleteTitle|editTitle|viewTitle|newTitle)|columns\.|overview\.chart\.title$)/,
  /^library\.(columns\.|tools\.optionalTitle|resources\.(libraryInfoTitle|fileInfoTitle|recordInfoTitle|currentRecordsTitle|deleteLibraryTitle)|download\.(title|inputTitle)|task\.(deleteFilesTitle|deleteSuccessTitle|deleteFailedTitle|bulkDeleteSuccessTitle|bulkDeleteFailedTitle|failedCheckTitle)|file\.(deleteFilesTitle|deleteSuccessTitle|deleteFailedTitle|bulkDeleteSuccessTitle|bulkDeleteFailedTitle)|rowMenu\.renameTitle|overview\.chart\.title|preview\.imageTitle|import\.(videoPickerTitle|subtitlePickerTitle|unsupportedTitle|dialog\.targetTitle)|config\.(saveFailedTitle|taskRuntime\.(translateTitle|proofreadTitle)|videoExportPresets\.(savedTitle|saveFailedTitle|deletedTitle|deleteFailedTitle)|subtitleStyles\.(allStylesTitle|createFormTitle|unsavedTitle|importFailedTitle|importGuideTitle|exportSucceededTitle|exportFailedTitle|previewInfoTitle|monoStyleSectionTitle|bilingualMetaSectionTitle|primarySourceTitle|secondarySourceTitle|primaryStyleTitle|secondaryStyleTitle|overviewTitle|defaultsTitle|deliveryReadinessTitle|fontManagementTitle|referencedFontsTitle|styleSourcesTitle|browseSourceFailedTitle|fontSourcesTitle|syncFontSourceSuccessTitle|syncFontSourceFailedTitle|installUserFontSuccessTitle|installUserFontFailedTitle|installMachineFontSuccessTitle|installMachineFontFailedTitle|importSourceItemSuccessTitle|importSourceItemFailedTitle))|workspace\.(emptyTitle|table\.(timelineTitle|editorTitle)|dialogs\.(exportVideo\.(subtitleHandlingTitle|trackMappingTitle)|languageTask\.(title|qaRealtimeTitle|issueBreakdownTitle|restoreOriginalTitle)|importSubtitle\.(title|normalizationTitle|guidelineInheritanceTitle))|preview\.placeholderTitle|waveform\.title|review\.(lockedTitle|applySuccessTitle|applyFailedTitle|discardSuccessTitle|discardFailedTitle)|notifications\.(translationReadyTitle|proofreadReadyTitle|qaReviewReadyTitle|saveFailedTitle|noSubtitleTrackTitle|originalRestoredTitle|restoreFailedTitle|noFileSelectedTitle|openFileFailedTitle|noVideoSelectedTitle|noPresetSelectedTitle|exportQueuedTitle|exportFailedTitle|moduleConfigUnavailableTitle|promptProfileSavedTitle|savePromptProfileFailedTitle|translationQueuedTitle|translationFailedTitle|proofreadQueuedTitle|proofreadFailedTitle|importProfileReadyTitle)))$/,
  /^(gateway\.logs\.title|debug\.(status\.title|channels\.title|message\.frontend\.(toastTitle|notificationTitle|dialogTitle)|message\.realtime\.notifyTitle)|chat\.composer\.attachDialogTitle|chat\.welcome\.entry\.items\.(assistant|providers|model|generic)\.action|chat\.tools\.approvalTool\.title|productMode\.options\.(full|download)\.(title|action)|notifications\.footer\.codes\.(appUpdate|externalToolsUpdate)\.title|notifications\.empty\.title)$/,
];
const englishTitleStyleHeuristicPattern = /(\.title$|Title$|\.label$|\.button$|\.action$|columns\.|Tabs\.|sections\.|Section$|dialogTitle$)/;
const englishTitleStyleSentencePattern = /\?|\.{1,3}$|\b(is|are|was|were|am|can't|cannot|need|needs|shows?\s+up|please)\b/i;
const zhGlossaryReplacements = [
  ["External Tools", "外部工具"],
  ["Runtime", "运行时"],
  ["Mono", "单语"],
  ["Bilingual", "双语"],
  ["Builtin", "内置"],
  ["Topic", "主题"],
  ["Toast", "提示"],
];

function flatten(input, prefix = "", output = {}) {
  for (const [key, value] of Object.entries(input)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === "object" && !Array.isArray(value)) {
      flatten(value, nextKey, output);
    } else {
      output[nextKey] = String(value);
    }
  }
  return output;
}

function walk(dir, output = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const next = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (next.includes(`${path.sep}bindings`) || next.includes(`${path.sep}shared${path.sep}i18n`)) {
        continue;
      }
      walk(next, output);
      continue;
    }
    if (!next.endsWith(".ts") && !next.endsWith(".tsx")) {
      continue;
    }
    output.push(next);
  }
  return output;
}

function relative(filePath) {
  return path.relative(rootDir, filePath).split(path.sep).join("/");
}

function isI18nCallExpression(node) {
  if (!ts.isCallExpression(node)) {
    return false;
  }
  if (ts.isIdentifier(node.expression)) {
    return node.expression.text === "t" || node.expression.text === "translate";
  }
  if (ts.isPropertyAccessExpression(node.expression)) {
    return node.expression.name.text === "t" || node.expression.name.text === "translate";
  }
  return false;
}

function isExplicitLanguageArg(node, sourceFile) {
  const text = node.getText(sourceFile).trim();
  return /(^language\b)|(\blanguage\b\s+as\s+)|(as\s+"en"\s*\|\s*"zh-CN")|(^"en"$)|(^"zh-CN"$)/.test(text);
}

function splitWord(word) {
  const parts = [];
  let buffer = "";
  for (const char of word) {
    if (char === "-" || char === "/") {
      if (buffer) {
        parts.push(buffer);
      }
      parts.push(char);
      buffer = "";
      continue;
    }
    buffer += char;
  }
  if (buffer) {
    parts.push(buffer);
  }
  return parts;
}

function maskEnglishPreservePhrases(value) {
  let text = value;
  englishStylePreservePhrases.forEach((phrase, index) => {
    const escaped = phrase.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    text = text.replace(new RegExp(escaped, "g"), `@@${index}@@`);
  });
  return text;
}

function unmaskEnglishPreservePhrases(value) {
  return value.replace(/@@(\d+)@@/g, (_, index) => englishStylePreservePhrases[Number(index)] ?? "");
}

function isEnglishStylePreservedWord(segment) {
  return (
    englishStylePreserveWords.has(segment) ||
    /^[A-Z0-9]+(?:_[A-Z0-9]+)*$/.test(segment) ||
    /^[a-z0-9]+(?:_[a-z0-9]+)+$/.test(segment) ||
    /^[A-Za-z]+=[A-Za-z0-9_-]+$/.test(segment) ||
    (/^[A-Za-z0-9-]+$/.test(segment) && /[A-Z].*[A-Z]/.test(segment) && /[a-z]/.test(segment))
  );
}

function splitAffixes(token) {
  const leading = token.match(/^[^A-Za-z{]*/)?.[0] ?? "";
  const trailing = token.match(/[^A-Za-z}]*$/)?.[0] ?? "";
  return {
    leading,
    trailing,
    core: token.slice(leading.length, token.length - trailing.length),
  };
}

function normalizeEnglishTitleSegment(segment, isFirst, isLast) {
  if (segment === "-" || segment === "/") {
    return segment;
  }
  if (!segment || !/[A-Za-z]/.test(segment)) {
    return segment;
  }
  if (isEnglishStylePreservedWord(segment)) {
    return segment;
  }
  if (segment.includes("/")) {
    const parts = splitWord(segment);
    const wordIndexes = parts.map((part, index) => (part !== "/" ? index : null)).filter((index) => index !== null);
    return parts
      .map((part, index) => {
        const position = wordIndexes.indexOf(index);
        if (position === -1) {
          return part;
        }
        return normalizeEnglishTitleSegment(part, isFirst && position === 0, isLast && position === wordIndexes.length - 1);
      })
      .join("");
  }
  if (segment.includes("-")) {
    const parts = splitWord(segment);
    const wordIndexes = parts.map((part, index) => (part !== "-" ? index : null)).filter((index) => index !== null);
    return parts
      .map((part, index) => {
        const position = wordIndexes.indexOf(index);
        if (position === -1) {
          return part;
        }
        return normalizeEnglishTitleSegment(part, isFirst && position === 0, isLast && position === wordIndexes.length - 1);
      })
      .join("");
  }
  const lower = segment.toLowerCase();
  if (!isFirst && !isLast && englishTitleCaseLowerWords.has(lower)) {
    return lower;
  }
  return `${lower.charAt(0).toUpperCase()}${lower.slice(1)}`;
}

function shouldUseEnglishTitleCase(key, value) {
  if (englishStyleSkipKeys.has(key) || typeof value !== "string" || !/[A-Za-z]/.test(value)) {
    return false;
  }
  if (englishTitleStyleExplicitPatterns.some((pattern) => pattern.test(key))) {
    return true;
  }
  if (!englishTitleStyleHeuristicPattern.test(key)) {
    return false;
  }
  if (englishTitleStyleSentencePattern.test(value)) {
    return false;
  }
  if (/^chat\.welcome\.entry\./.test(key) || /^notifications\.center\.codes\./.test(key) || /^productMode\.title$/.test(key)) {
    return false;
  }
  return true;
}

function normalizeEnglishSentenceValue(key, value) {
  if (englishStyleSkipKeys.has(key) || typeof value !== "string" || !/[A-Za-z]/.test(value)) {
    return value;
  }
  const text = maskEnglishPreservePhrases(value);
  const tokens = text.match(/\{[^}]+\}|@@\d+@@|[A-Za-z][A-Za-z0-9'/-]*|[^A-Za-z{@]+|[@][^@]+[@]?/g) ?? [text];
  let sentenceStart = true;
  const output = tokens
    .map((token) => {
      if (token.startsWith("@@")) {
        sentenceStart = false;
        return token;
      }
      if (token.startsWith("{") && token.endsWith("}")) {
        sentenceStart = false;
        return token;
      }
      if (!/[A-Za-z]/.test(token)) {
        const trimmed = token.trimEnd();
        const lastChar = trimmed.charAt(trimmed.length - 1);
        if (englishSentenceBoundaryChars.has(lastChar)) {
          sentenceStart = true;
        }
        return token;
      }
      const leading = token.match(/^[^A-Za-z{]*/)?.[0] ?? "";
      const trailing = token.match(/[^A-Za-z}]*$/)?.[0] ?? "";
      const core = token.slice(leading.length, token.length - trailing.length);
      const segments = splitWord(core);
      let nextSentenceStart = sentenceStart;
      const normalized = segments
        .map((segment) => {
          if (segment === "-" || segment === "/") {
            return segment;
          }
          if (!segment || !/[A-Za-z]/.test(segment)) {
            return segment;
          }
          if (
            isEnglishStylePreservedWord(segment)
          ) {
            nextSentenceStart = false;
            return segment;
          }
          if (/^[A-Z][a-z0-9]+(?:'[A-Za-z]+)?$/.test(segment) || /^[a-z][a-z0-9]+(?:'[A-Za-z]+)?$/.test(segment)) {
            const lower = segment.toLowerCase();
            const nextValue = nextSentenceStart ? `${lower.charAt(0).toUpperCase()}${lower.slice(1)}` : (englishStyleAlwaysLowerWords.has(lower) ? lower : lower);
            nextSentenceStart = false;
            return nextValue;
          }
          if (nextSentenceStart && /^[a-z]/.test(segment)) {
            nextSentenceStart = false;
            return `${segment.charAt(0).toUpperCase()}${segment.slice(1)}`;
          }
          nextSentenceStart = false;
          return segment;
        })
        .join("");
      sentenceStart = false;
      return `${leading}${normalized}${trailing}`;
    })
    .join("");
  return unmaskEnglishPreservePhrases(output);
}

function normalizeEnglishTitleValue(key, value) {
  if (englishStyleSkipKeys.has(key) || typeof value !== "string" || !/[A-Za-z]/.test(value)) {
    return value;
  }
  const text = maskEnglishPreservePhrases(value);
  const parts = text.split(/(\s+)/);
  const wordIndexes = parts
    .map((part, index) => {
      if (!part || /^\s+$/.test(part) || /^@@\d+@@$/.test(part) || (part.startsWith("{") && part.endsWith("}"))) {
        return null;
      }
      const { core } = splitAffixes(part);
      return /[A-Za-z]/.test(core) ? index : null;
    })
    .filter((index) => index !== null);
  const output = parts
    .map((part, index) => {
      if (!part || /^\s+$/.test(part) || /^@@\d+@@$/.test(part) || (part.startsWith("{") && part.endsWith("}"))) {
        return part;
      }
      const position = wordIndexes.indexOf(index);
      if (position === -1) {
        return part;
      }
      const { leading, trailing, core } = splitAffixes(part);
      return `${leading}${normalizeEnglishTitleSegment(core, position === 0, position === wordIndexes.length - 1)}${trailing}`;
    })
    .join("");
  return unmaskEnglishPreservePhrases(output);
}

function normalizeEnglishLocaleValue(key, value) {
  if (shouldUseEnglishTitleCase(key, value)) {
    return normalizeEnglishTitleValue(key, value);
  }
  return normalizeEnglishSentenceValue(key, value);
}

function normalizeZhGlossaryValue(value) {
  if (typeof value !== "string") {
    return value;
  }
  let nextValue = value;
  for (const [from, to] of zhGlossaryReplacements) {
    nextValue = nextValue.split(from).join(to);
  }
  return nextValue;
}

function filterLocaleTree(input, usedSet, prefix = "") {
  const output = {};
  for (const [key, value] of Object.entries(input)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === "object" && !Array.isArray(value)) {
      const nextValue = filterLocaleTree(value, usedSet, nextKey);
      if (Object.keys(nextValue).length > 0) {
        output[key] = nextValue;
      }
      continue;
    }
    if (usedSet.has(nextKey)) {
      output[key] = value;
    }
  }
  return output;
}

const enSource = JSON.parse(fs.readFileSync(path.join(localeDir, "en.json"), "utf8"));
const zhSource = JSON.parse(fs.readFileSync(path.join(localeDir, "zh-CN.json"), "utf8"));
const en = flatten(enSource);
const zh = flatten(zhSource);
const files = walk(sourceDir);
const localeKeys = new Set([...Object.keys(en), ...Object.keys(zh)]);
const sortedLocaleKeys = [...localeKeys];

const usedKeys = new Set();
const hardcodedChinese = [];
const unresolvedDynamicKeys = [];
const i18nCallViolations = [];
const keyPattern = /\b(?:t|translate)\(\s*(["'`])([^"'`]+)\1/g;
const propertyKeyPattern = /\b(?:labelKey|descriptionKey|reasonKey)\s*:\s*(["'`])([^"'`]+)\1/g;
const stringLiteralPattern = /(["'`])([A-Za-z0-9._-]+)\1/g;

function resolveDynamicTemplate(raw) {
  const match = /^([A-Za-z0-9._-]*)\$\{[^}\n]+\}([A-Za-z0-9._-]*)$/.exec(raw);
  if (!match) {
    return false;
  }
  const [, prefix, suffix] = match;
  if (!`${prefix}${suffix}`.includes(".")) {
    return false;
  }
  const matchedKeys = sortedLocaleKeys.filter((key) => key.startsWith(prefix) && key.endsWith(suffix));
  if (matchedKeys.length === 0) {
    unresolvedDynamicKeys.push(raw);
    return false;
  }
  for (const key of matchedKeys) {
    usedKeys.add(key);
  }
  return true;
}

for (const file of files) {
  const text = fs.readFileSync(file, "utf8");
  const sourceFile = ts.createSourceFile(
    file,
    text,
    ts.ScriptTarget.Latest,
    true,
    file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  );

  function visit(node) {
    if (isI18nCallExpression(node)) {
      const calleeName = ts.isPropertyAccessExpression(node.expression) ? node.expression.name.text : node.expression.text;
      const secondArg = node.arguments[1];
      const hasFallbackLikeArg =
        calleeName === "translate" ||
        (node.arguments.length >= 2 && secondArg && !isExplicitLanguageArg(secondArg, sourceFile));
      if (hasFallbackLikeArg) {
        const line = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1;
        i18nCallViolations.push({
          file: relative(file),
          line,
          content: node.getText(sourceFile),
        });
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);

  for (const match of text.matchAll(keyPattern)) {
    const key = match[2];
    if (key.includes("${")) {
      resolveDynamicTemplate(key);
    } else {
      usedKeys.add(key);
    }
  }

  for (const match of text.matchAll(propertyKeyPattern)) {
    const key = match[2];
    if (key.includes("${")) {
      resolveDynamicTemplate(key);
    } else if (localeKeys.has(key)) {
      usedKeys.add(key);
    }
  }

  for (const match of text.matchAll(stringLiteralPattern)) {
    const candidate = match[2];
    if (localeKeys.has(candidate)) {
      usedKeys.add(candidate);
    }
  }

  for (const [index, line] of text.split(/\r?\n/).entries()) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    if (/^\/\/|^\/\*|^\*|^\{\/\*/.test(trimmed)) {
      continue;
    }
    if (/\b(?:t|translate)\(/.test(line)) {
      continue;
    }
    if (/[\u4e00-\u9fff]/.test(line)) {
      hardcodedChinese.push({
        file: relative(file),
        line: index + 1,
        content: trimmed,
      });
    }
  }
}

const enKeys = new Set(Object.keys(en));
const zhKeys = new Set(Object.keys(zh));

const missingInZh = [...enKeys].filter((key) => !zhKeys.has(key));
const extraInZh = [...zhKeys].filter((key) => !enKeys.has(key));
const unusedInEn = [...enKeys].filter((key) => !usedKeys.has(key));
const missingDefs = [...usedKeys].filter((key) => !enKeys.has(key));
const concreteMissingDefs = missingDefs.filter((key) => !key.includes("${"));
const dynamicMissingDefs = [...new Set(unresolvedDynamicKeys)];
const englishStyleViolations = Object.entries(en)
  .filter(([key, value]) => normalizeEnglishLocaleValue(key, value) !== value)
  .map(([key, value]) => ({ key, value, expected: normalizeEnglishLocaleValue(key, value) }));
const zhGlossaryViolations = Object.entries(zh)
  .filter(([, value]) => normalizeZhGlossaryValue(value) !== value)
  .map(([key, value]) => ({ key, value, expected: normalizeZhGlossaryValue(value) }));

const summary = {
  locale: {
    enCount: enKeys.size,
    zhCount: zhKeys.size,
    usedKeyCount: usedKeys.size,
    missingInZhCount: missingInZh.length,
    extraInZhCount: extraInZh.length,
    unusedInEnCount: unusedInEn.length,
    missingDefinitionCount: missingDefs.length,
    concreteMissingDefinitionCount: concreteMissingDefs.length,
    dynamicMissingDefinitionCount: dynamicMissingDefs.length,
    i18nCallViolationCount: i18nCallViolations.length,
    englishStyleViolationCount: englishStyleViolations.length,
    zhGlossaryViolationCount: zhGlossaryViolations.length,
  },
  samples: {
    concreteMissingDefinitions: concreteMissingDefs.slice(0, 40),
    dynamicMissingDefinitions: dynamicMissingDefs.slice(0, 20),
    unusedInEn: unusedInEn.slice(0, 40),
    hardcodedChinese: hardcodedChinese.slice(0, 40),
    i18nCallViolations: i18nCallViolations.slice(0, 40),
    englishStyleViolations: englishStyleViolations.slice(0, 40),
    zhGlossaryViolations: zhGlossaryViolations.slice(0, 40),
  },
};

if (jsonOnly) {
  console.log(JSON.stringify(summary, null, 2));
} else {
  console.log("i18n audit summary");
  console.log(`- locale keys: en=${summary.locale.enCount}, zh-CN=${summary.locale.zhCount}`);
  console.log(`- used keys in source: ${summary.locale.usedKeyCount}`);
  console.log(`- missing definitions: ${summary.locale.missingDefinitionCount} (concrete=${summary.locale.concreteMissingDefinitionCount}, dynamic=${summary.locale.dynamicMissingDefinitionCount})`);
  console.log(`- unused en keys: ${summary.locale.unusedInEnCount}`);
  console.log(`- missing zh-CN keys: ${summary.locale.missingInZhCount}`);
  console.log(`- extra zh-CN keys: ${summary.locale.extraInZhCount}`);
  console.log(`- hardcoded Chinese lines: ${hardcodedChinese.length}`);
  console.log(`- invalid i18n calls: ${i18nCallViolations.length}`);
  console.log(`- english style violations: ${englishStyleViolations.length}`);
  console.log(`- zh glossary violations: ${zhGlossaryViolations.length}`);

  if (concreteMissingDefs.length > 0) {
    console.log("\nmissing concrete keys:");
    for (const key of concreteMissingDefs.slice(0, 20)) {
      console.log(`- ${key}`);
    }
  }

  if (dynamicMissingDefs.length > 0) {
    console.log("\ndynamic keys that need manual review:");
    for (const key of dynamicMissingDefs.slice(0, 10)) {
      console.log(`- ${key}`);
    }
  }

  if (hardcodedChinese.length > 0) {
    console.log("\nhardcoded Chinese samples:");
    for (const item of hardcodedChinese.slice(0, 20)) {
      console.log(`- ${item.file}:${item.line} ${item.content}`);
    }
  }

  if (i18nCallViolations.length > 0) {
    console.log("\ninvalid i18n call samples:");
    for (const item of i18nCallViolations.slice(0, 20)) {
      console.log(`- ${item.file}:${item.line} ${item.content}`);
    }
  }

  if (englishStyleViolations.length > 0) {
    console.log("\nenglish style samples:");
    for (const item of englishStyleViolations.slice(0, 20)) {
      console.log(`- ${item.key}: ${item.value} -> ${item.expected}`);
    }
  }

  if (zhGlossaryViolations.length > 0) {
    console.log("\nzh glossary samples:");
    for (const item of zhGlossaryViolations.slice(0, 20)) {
      console.log(`- ${item.key}: ${item.value} -> ${item.expected}`);
    }
  }
}

if (pruneUnused) {
  const nextEn = filterLocaleTree(enSource, usedKeys);
  const nextZh = filterLocaleTree(zhSource, usedKeys);
  fs.writeFileSync(path.join(localeDir, "en.json"), `${JSON.stringify(nextEn, null, 2)}\n`);
  fs.writeFileSync(path.join(localeDir, "zh-CN.json"), `${JSON.stringify(nextZh, null, 2)}\n`);
}

if (strict) {
  const hasBlockingIssues =
    missingInZh.length > 0 ||
    extraInZh.length > 0 ||
    concreteMissingDefs.length > 0 ||
    dynamicMissingDefs.length > 0 ||
    unusedInEn.length > 0 ||
    hardcodedChinese.length > 0 ||
    i18nCallViolations.length > 0 ||
    englishStyleViolations.length > 0 ||
    zhGlossaryViolations.length > 0;
  if (hasBlockingIssues) {
    process.exitCode = 1;
  }
}
