import * as React from "react";
import { PatchDiff } from "@pierre/diffs/react";
import type { ComponentsByLanguage, SyntaxHighlighterProps } from "@assistant-ui/react-streamdown";
import { renderMermaid } from "beautiful-mermaid";
import rehypePrettyCode from "rehype-pretty-code";
import rehypeStringify from "rehype-stringify";
import { unified } from "unified";

const CodeFallback = ({ code, components }: { code: string; components: SyntaxHighlighterProps["components"] }) => {
  const { Pre, Code } = components;
  return (
    <Pre className="overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-2 text-xs">
      <Code>{code}</Code>
    </Pre>
  );
};

const jsonTheme = {
  light: "github-light",
  dark: "github-dark",
} as const;

const prettyCodeProcessor = unified()
  .use(rehypePrettyCode, { theme: jsonTheme, keepBackground: false })
  .use(rehypeStringify);

const renderPrettyCode = async (code: string, language: string) => {
  const trimmed = code.trim();
  if (!trimmed) {
    return "";
  }
  const tree = {
    type: "root",
    children: [
      {
        type: "element",
        tagName: "pre",
        properties: {},
        children: [
          {
            type: "element",
            tagName: "code",
            properties: { className: [`language-${language}`] },
            children: [{ type: "text", value: trimmed }],
          },
        ],
      },
    ],
  };
  const processed = await prettyCodeProcessor.run(tree as any);
  return String(prettyCodeProcessor.stringify(processed as any));
};

const normalizeJsonCode = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
};

const usePrettyCodeHtml = (code: string, language: string) => {
  const [html, setHtml] = React.useState<string>("");

  React.useEffect(() => {
    let cancelled = false;
    const run = async () => {
      try {
        const next = await renderPrettyCode(code, language);
        if (!cancelled) {
          setHtml(next);
        }
      } catch {
        if (!cancelled) {
          setHtml("");
        }
      }
    };
    if (code.trim()) {
      void run();
    } else {
      setHtml("");
    }
    return () => {
      cancelled = true;
    };
  }, [code, language]);

  return html;
};

export const UniversalCodeSyntaxHighlighter = ({ code, language, components }: SyntaxHighlighterProps) => {
  const resolvedLanguage = language?.trim() || "text";
  const html = usePrettyCodeHtml(code, resolvedLanguage);
  if (!html) {
    return <CodeFallback code={code} components={components} />;
  }
  return <div className="my-2 pretty-code" dangerouslySetInnerHTML={{ __html: html }} />;
};

export const PrettyJsonBlock = ({ value, raw }: { value?: unknown; raw?: string }) => {
  let code = typeof raw === "string" ? raw : "";
  if (!code && value != null) {
    if (typeof value === "string") {
      code = value;
    } else {
      try {
        code = JSON.stringify(value, null, 2);
      } catch {
        code = String(value);
      }
    }
  }
  const normalized = normalizeJsonCode(code);
  const html = usePrettyCodeHtml(normalized, "json");
  if (!html) {
    return (
      <pre className="overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-2 text-xs">
        {normalized || code}
      </pre>
    );
  }
  return <div className="pretty-code" dangerouslySetInnerHTML={{ __html: html }} />;
};

const DiffSyntaxHighlighter = ({ code, components }: SyntaxHighlighterProps) => {
  const trimmed = code.trim();
  if (!trimmed) {
    return <CodeFallback code={code} components={components} />;
  }
  return (
    <div className="my-2 overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-2 text-xs">
      <PatchDiff patch={trimmed} />
    </div>
  );
};

const MermaidSyntaxHighlighter = ({ code, components }: SyntaxHighlighterProps) => {
  const [svg, setSvg] = React.useState<string>("");
  const [error, setError] = React.useState<string | null>(null);
  const source = code.trim();

  React.useEffect(() => {
    let cancelled = false;
    if (!source) {
      setSvg("");
      setError(null);
      return () => {
        cancelled = true;
      };
    }
    setError(null);
    renderMermaid(source)
      .then((result) => {
        if (!cancelled) {
          setSvg(result);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
          setSvg("");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [source]);

  if (!source || error || !svg) {
    return <CodeFallback code={code} components={components} />;
  }

  return (
    <div className="my-2 overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-2">
      <div className="mermaid-diagram" dangerouslySetInnerHTML={{ __html: svg }} />
    </div>
  );
};

const JsonSyntaxHighlighter = ({ code, components }: SyntaxHighlighterProps) => {
  const normalized = normalizeJsonCode(code);
  const html = usePrettyCodeHtml(normalized, "json");
  if (!html) {
    return <CodeFallback code={code} components={components} />;
  }
  return <div className="my-2 pretty-code" dangerouslySetInnerHTML={{ __html: html }} />;
};

export const CHAT_CODE_LANGUAGES: ComponentsByLanguage = {
  diff: { SyntaxHighlighter: DiffSyntaxHighlighter },
  patch: { SyntaxHighlighter: DiffSyntaxHighlighter },
  mermaid: { SyntaxHighlighter: MermaidSyntaxHighlighter },
  json: { SyntaxHighlighter: JsonSyntaxHighlighter },
  jsonc: { SyntaxHighlighter: JsonSyntaxHighlighter },
};
