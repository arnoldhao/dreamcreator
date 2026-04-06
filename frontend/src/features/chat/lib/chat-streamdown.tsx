import { INTERNAL, useMessagePartText } from "@assistant-ui/react";
import { harden } from "rehype-harden";
import rehypeRaw from "rehype-raw";
import rehypeSanitize from "rehype-sanitize";
import { Streamdown, type StreamdownProps } from "streamdown";
import {
  DEFAULT_SHIKI_THEME,
  type CodeHeaderProps,
  type ComponentsByLanguage,
  type SecurityConfig,
  type StreamdownTextComponents,
  type StreamdownTextPrimitiveProps,
  type SyntaxHighlighterProps,
} from "@assistant-ui/react-streamdown";
import type { ComponentPropsWithoutRef, ComponentType } from "react";
import { createContext, forwardRef, useContext, useMemo } from "react";

const { useSmoothStatus } = INTERNAL;

type StreamdownTextPrimitiveElement = HTMLDivElement;

type PreOverrideProps = ComponentPropsWithoutRef<"pre"> & {
  node?: unknown;
};

const PreContext = createContext<PreOverrideProps | null>(null);

function PreOverride({ children, node: _node, ...rest }: PreOverrideProps) {
  return (
    <PreContext.Provider value={{ node: _node, ...rest }}>
      <pre {...rest}>{children}</pre>
    </PreContext.Provider>
  );
}

function useIsCodeBlock() {
  return useContext(PreContext) !== null;
}

const LANGUAGE_REGEX = /language-([^\s]+)/;

type CodeProps = ComponentPropsWithoutRef<"code"> & {
  node?: unknown;
};

const DefaultPre = (props: ComponentPropsWithoutRef<"pre">) => <pre {...props} />;
const DefaultCode = (props: ComponentPropsWithoutRef<"code">) => <code {...props} />;

const extractCode = (children: unknown): string => {
  if (typeof children === "string") return children;
  if (!children || typeof children !== "object") return "";
  const element = children as { props?: Record<string, unknown> };
  const content = element.props?.children;
  return typeof content === "string" ? content : "";
};

type CodeAdapterOptions = {
  SyntaxHighlighter?: ComponentType<SyntaxHighlighterProps> | undefined;
  CodeHeader?: ComponentType<CodeHeaderProps> | undefined;
  componentsByLanguage?: ComponentsByLanguage | undefined;
};

function createCodeAdapter(options: CodeAdapterOptions) {
  const { SyntaxHighlighter: UserSyntaxHighlighter, CodeHeader: UserCodeHeader, componentsByLanguage } = options;
  const languageOverrides = componentsByLanguage ?? {};

  return function AdaptedCode({ node, className, children, ...props }: CodeProps) {
    const isCodeBlock = useIsCodeBlock();
    if (!isCodeBlock) {
      return (
        <code className={`aui-streamdown-inline-code ${className ?? ""}`.trim()} {...props}>
          {children}
        </code>
      );
    }

    const match = className?.match(LANGUAGE_REGEX);
    const language = match?.[1] ?? "";
    const code = extractCode(children);

    const SyntaxHighlighter =
      languageOverrides[language]?.SyntaxHighlighter ?? UserSyntaxHighlighter;
    const CodeHeader = languageOverrides[language]?.CodeHeader ?? UserCodeHeader;

    if (SyntaxHighlighter) {
      return (
        <>
          {CodeHeader && <CodeHeader language={language} code={code} />}
          <SyntaxHighlighter
            node={node as never}
            components={{ Pre: DefaultPre, Code: DefaultCode }}
            language={language}
            code={code}
          />
        </>
      );
    }

    return (
      <code className={className} {...props}>
        {children}
      </code>
    );
  };
}

function shouldUseCodeAdapter(options: CodeAdapterOptions): boolean {
  return !!(
    options.SyntaxHighlighter ||
    options.CodeHeader ||
    (options.componentsByLanguage && Object.keys(options.componentsByLanguage).length > 0)
  );
}

function useAdaptedComponents({
  components,
  componentsByLanguage,
}: {
  components?: StreamdownTextComponents | undefined;
  componentsByLanguage?: ComponentsByLanguage | undefined;
}): StreamdownProps["components"] {
  return useMemo(() => {
    const { SyntaxHighlighter, CodeHeader, ...htmlComponents } = components ?? {};
    const baseComponents = { pre: PreOverride };

    const codeAdapterOptions = {
      SyntaxHighlighter,
      CodeHeader,
      componentsByLanguage,
    };

    if (!shouldUseCodeAdapter(codeAdapterOptions)) {
      return { ...htmlComponents, ...baseComponents };
    }

    const AdaptedCode = createCodeAdapter(codeAdapterOptions);
    return {
      ...htmlComponents,
      ...baseComponents,
      code: AdaptedCode,
    };
  }, [components, componentsByLanguage]);
}

function buildSecurityRehypePlugins(
  security: SecurityConfig,
): NonNullable<StreamdownProps["rehypePlugins"]> {
  return [
    rehypeRaw,
    [rehypeSanitize, {}],
    [
      harden,
      {
        allowedImagePrefixes: security.allowedImagePrefixes ?? ["*"],
        allowedLinkPrefixes: security.allowedLinkPrefixes ?? ["*"],
        allowedProtocols: security.allowedProtocols ?? ["*"],
        allowDataImages: security.allowDataImages ?? true,
        defaultOrigin: security.defaultOrigin,
        blockedLinkClass: security.blockedLinkClass,
        blockedImageClass: security.blockedImageClass,
      },
    ],
  ];
}

const mergePlugins = (
  userPlugins: StreamdownTextPrimitiveProps["plugins"],
): Record<string, unknown> | undefined => {
  if (!userPlugins) {
    return undefined;
  }
  const merged: Record<string, unknown> = {};
  const keys = ["code", "math", "cjk"] as const;
  for (const key of keys) {
    const value = userPlugins[key];
    if (value === false) {
      continue;
    }
    if (value) {
      merged[key] = value;
    }
  }
  const mermaid = userPlugins.mermaid;
  if (mermaid && mermaid !== false) {
    merged.mermaid = mermaid;
  }
  return Object.keys(merged).length > 0 ? merged : undefined;
};

export const ChatStreamdownTextPrimitive = forwardRef<
  StreamdownTextPrimitiveElement,
  StreamdownTextPrimitiveProps
>(
  (
    {
      components,
      componentsByLanguage,
      preprocess,
      plugins: userPlugins,
      containerProps,
      containerClassName,
      caret,
      controls,
      linkSafety,
      remend,
      mermaid,
      parseIncompleteMarkdown,
      allowedTags,
      remarkRehypeOptions,
      security,
      BlockComponent,
      parseMarkdownIntoBlocksFn,
      mode = "streaming",
      className,
      shikiTheme,
      ...streamdownProps
    },
    ref,
  ) => {
    const { text } = useMessagePartText();
    const status = useSmoothStatus();

    const processedText = useMemo(
      () => (preprocess ? preprocess(text) : text),
      [text, preprocess],
    );

    const resolvedPlugins = useMemo(() => mergePlugins(userPlugins), [userPlugins]);

    const resolvedShikiTheme = useMemo(
      () => shikiTheme ?? (resolvedPlugins?.code ? DEFAULT_SHIKI_THEME : undefined),
      [shikiTheme, resolvedPlugins?.code],
    );

    const adaptedComponents = useAdaptedComponents({
      components,
      componentsByLanguage,
    });

    const mergedComponents = useMemo(() => {
      const {
        SyntaxHighlighter: _,
        CodeHeader: __,
        ...userHtmlComponents
      } = components ?? {};
      return { ...userHtmlComponents, ...adaptedComponents };
    }, [components, adaptedComponents]);

    const containerClass = useMemo(() => {
      const classes = [containerClassName, containerProps?.className]
        .filter(Boolean)
        .join(" ");
      return classes || undefined;
    }, [containerClassName, containerProps?.className]);

    const rehypePlugins = useMemo(
      () => (security ? buildSecurityRehypePlugins(security) : undefined),
      [security],
    );

    const optionalProps = {
      ...(className && { className }),
      ...(caret && { caret }),
      ...(controls !== undefined && { controls }),
      ...(linkSafety && { linkSafety }),
      ...(remend && { remend }),
      ...(mermaid && { mermaid }),
      ...(parseIncompleteMarkdown !== undefined && { parseIncompleteMarkdown }),
      ...(allowedTags && { allowedTags }),
      ...(resolvedPlugins && { plugins: resolvedPlugins }),
      ...(resolvedShikiTheme && { shikiTheme: resolvedShikiTheme }),
      ...(remarkRehypeOptions && { remarkRehypeOptions }),
      ...(rehypePlugins && { rehypePlugins }),
      ...(BlockComponent && { BlockComponent }),
      ...(parseMarkdownIntoBlocksFn && { parseMarkdownIntoBlocksFn }),
    };

    return (
      <div ref={ref} data-status={status.type} {...containerProps} className={containerClass}>
        <Streamdown
          mode={mode}
          isAnimating={status.type === "running"}
          components={mergedComponents}
          {...optionalProps}
          {...streamdownProps}
        >
          {processedText}
        </Streamdown>
      </div>
    );
  },
);

ChatStreamdownTextPrimitive.displayName = "ChatStreamdownTextPrimitive";
