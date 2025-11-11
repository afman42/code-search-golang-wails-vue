import DOMPurify from "dompurify";
import { toastManager } from "../composables/useToast";

// Interface for syntax highlighting options
export interface SyntaxHighlightOptions {
  language?: string;
  query?: string;
  addLineNumbers?: boolean;
}

let hljsModule: any = null;
let isHighlightingLoaded = false;

// Function to load highlight.js dynamically
export const loadHighlightJs = async (): Promise<boolean> => {
  if (isHighlightingLoaded) {
    return true;
  }

  try {
    // Dynamically import only the languages we commonly use
    const hljsCore = await import("highlight.js/lib/core");
    hljsModule = hljsCore.default;

    // Import and register only the languages we commonly use
    const goLang = await import("highlight.js/lib/languages/go");
    const jsLang = await import("highlight.js/lib/languages/javascript");
    const tsLang = await import("highlight.js/lib/languages/typescript");
    const javaLang = await import("highlight.js/lib/languages/java");
    const pyLang = await import("highlight.js/lib/languages/python");
    const rbLang = await import("highlight.js/lib/languages/ruby");
    const phpLang = await import("highlight.js/lib/languages/php");
    const cppLang = await import("highlight.js/lib/languages/cpp");
    const cLang = await import("highlight.js/lib/languages/c");
    const htmlLang = await import("highlight.js/lib/languages/xml"); // HTML is a subset of XML in highlight.js
    const cssLang = await import("highlight.js/lib/languages/css");
    const jsonLang = await import("highlight.js/lib/languages/json");
    const yamlLang = await import("highlight.js/lib/languages/yaml");
    const markdownLang = await import("highlight.js/lib/languages/markdown");
    const sqlLang = await import("highlight.js/lib/languages/sql");
    const bashLang = await import("highlight.js/lib/languages/bash");
    const rustLang = await import("highlight.js/lib/languages/rust");
    const swiftLang = await import("highlight.js/lib/languages/swift");
    const kotlinLang = await import("highlight.js/lib/languages/kotlin");
    const scalaLang = await import("highlight.js/lib/languages/scala");
    const dartLang = await import("highlight.js/lib/languages/dart");
    const luaLang = await import("highlight.js/lib/languages/lua");
    const perlLang = await import("highlight.js/lib/languages/perl");
    const rLang = await import("highlight.js/lib/languages/r");
    const coffeeLang = await import("highlight.js/lib/languages/coffeescript");

    hljsModule.registerLanguage("go", goLang.default);
    hljsModule.registerLanguage("javascript", jsLang.default);
    hljsModule.registerLanguage("typescript", tsLang.default);
    hljsModule.registerLanguage("java", javaLang.default);
    hljsModule.registerLanguage("python", pyLang.default);
    hljsModule.registerLanguage("ruby", rbLang.default);
    hljsModule.registerLanguage("php", phpLang.default);
    hljsModule.registerLanguage("cpp", cppLang.default);
    hljsModule.registerLanguage("c", cLang.default);
    hljsModule.registerLanguage("html", htmlLang.default);
    hljsModule.registerLanguage("xml", htmlLang.default);
    hljsModule.registerLanguage("css", cssLang.default);
    hljsModule.registerLanguage("json", jsonLang.default);
    hljsModule.registerLanguage("yaml", yamlLang.default);
    hljsModule.registerLanguage("markdown", markdownLang.default);
    hljsModule.registerLanguage("sql", sqlLang.default);
    hljsModule.registerLanguage("bash", bashLang.default);
    hljsModule.registerLanguage("rust", rustLang.default);
    hljsModule.registerLanguage("swift", swiftLang.default);
    hljsModule.registerLanguage("kotlin", kotlinLang.default);
    hljsModule.registerLanguage("scala", scalaLang.default);
    hljsModule.registerLanguage("dart", dartLang.default);
    hljsModule.registerLanguage("lua", luaLang.default);
    hljsModule.registerLanguage("perl", perlLang.default);
    hljsModule.registerLanguage("r", rLang.default);
    hljsModule.registerLanguage("coffeescript", coffeeLang.default);

    isHighlightingLoaded = true;
    toastManager.success("Syntax Highlight Successfully Loaded");
    return true;
  } catch (e) {
    console.error("Error loading highlight.js", e);
    toastManager.success("Error loading highlight.js");
    return false;
  }
};

// Function to detect language from file extension
export const detectLanguage = (filePath: string): string => {
  if (!filePath) return "text";
  const ext = filePath.split(".").pop()?.toLowerCase() || "";
  const languages: Record<string, string> = {
    go: "go",
    js: "javascript",
    ts: "typescript",
    java: "java",
    py: "python",
    rb: "ruby",
    php: "php",
    cpp: "cpp",
    hpp: "cpp",
    h: "c",
    c: "c",
    html: "html",
    htm: "html",
    xml: "xml",
    css: "css",
    json: "json",
    yaml: "yaml",
    yml: "yaml",
    md: "markdown",
    sql: "sql",
    sh: "bash",
    bash: "bash",
    rs: "rust",
    swift: "swift",
    kt: "kotlin",
    scala: "scala",
    dart: "dart",
    lua: "lua",
    pl: "perl",
    r: "r",
    coffee: "coffeescript",
  };
  return languages[ext] || "text";
};

// Utility function to escape HTML
const escapeHtml = (unsafe: string): string => {
  if (!unsafe) return "";
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
};

// Main function to highlight code
export const highlightCode = async (
  code: string,
  options: SyntaxHighlightOptions = {},
): Promise<string> => {
  // If highlighting is not loaded, load it first
  if (!isHighlightingLoaded) {
    const loaded = await loadHighlightJs();
    if (!loaded) {
      // If highlight.js fails to load, return plain escaped text
      return escapeHtml(code);
    }
  }

  const { language = "text", query = "", addLineNumbers = true } = options;

  if (!code) {
    return "";
  }

  // For very large files, we'll process in chunks to improve performance
  const lines = code.split(/\r?\n/);

  // If file is very large, apply syntax highlighting line by line to avoid performance issues
  if (lines.length > 1000) {
    // For large files, we'll do a simplified approach to avoid performance issues
    let html = "";
    for (let i = 0; i < lines.length && i < 10000; i++) {
      // Limit to 10k lines to prevent browser crashes
      const lineNumber = i + 1;
      let lineContent = escapeHtml(lines[i]);

      // Apply syntax highlighting to individual lines if possible
      try {
        // Check if language is supported before applying syntax highlighting
        if (hljsModule && hljsModule.getLanguage(language)) {
          lineContent = hljsModule.highlight(lineContent, {
            language: language,
          }).value;
        } else {
          // If language is not supported, just escape HTML to prevent XSS
          lineContent = escapeHtml(lines[i]);
        }
      } catch (e) {
        // If syntax highlighting fails, use plain HTML escaped content
        lineContent = escapeHtml(lines[i]);
      }

      // Highlight query matches if query exists
      if (query) {
        try {
          const regex = new RegExp(
            `(${query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
            "gi",
          );
          lineContent = lineContent.replace(
            regex,
            '<mark class="highlight-match">$1</mark>',
          );
        } catch (e) {
          // If regex fails, continue without highlighting
        }
      }

      // Sanitize the line content to prevent XSS
      lineContent = DOMPurify.sanitize(lineContent, {
        ALLOWED_TAGS: ["mark", "span"],
        ALLOWED_ATTR: ["class", "style", "data-line"],
      });

      if (addLineNumbers) {
        // Add line with number
        html += `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${lineNumber}">${lineNumber}</span><span class="code-line">${lineContent || " "}</span>\n`;
      } else {
        html += `<span class="code-line">${lineContent || " "}</span>\n`;
      }
    }

    // Add note if we truncated the file
    if (lines.length > 10000) {
      html += `<span class="line-number" data-line="...">...</span><span class="code-line comment">/* File truncated - showing first 10,000 lines */</span>\n`;
    }

    return html;
  } else {
    // For smaller files, apply syntax highlighting to the whole content
    let highlightedCodeResult = code;

    try {
      // Check if language is supported before applying syntax highlighting
      if (hljsModule && hljsModule.getLanguage(language)) {
        highlightedCodeResult = hljsModule.highlight(code, {
          language: language,
        }).value;
      } else {
        // If language is not supported, just escape HTML to prevent XSS
        highlightedCodeResult = escapeHtml(code);
      }
    } catch (e) {
      // If syntax highlighting fails, use plain HTML escaped content
      highlightedCodeResult = escapeHtml(code);
    }

    // Split code into lines
    const codeLines = highlightedCodeResult.split(/\r?\n/);
    let html = "";

    for (let i = 0; i < codeLines.length; i++) {
      const lineNumber = i + 1;
      let lineContent = codeLines[i];

      // Highlight query matches if query exists
      if (query) {
        try {
          const regex = new RegExp(
            `(${query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
            "gi",
          );
          lineContent = lineContent.replace(
            regex,
            '<mark class="highlight-match">$1</mark>',
          );
        } catch (e) {
          // If regex fails, continue without highlighting
        }
      }

      // Sanitize the line content to prevent XSS
      lineContent = DOMPurify.sanitize(lineContent, {
        ALLOWED_TAGS: ["mark", "span"],
        ALLOWED_ATTR: ["class", "style", "data-line"],
      });

      if (addLineNumbers) {
        html += `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${lineNumber}">${lineNumber}</span><span class="code-line">${lineContent || " "}</span>\n`;
      } else {
        html += `<span class="code-line">${lineContent || " "}</span>\n`;
      }
    }

    return html;
  }
};

// Check if highlighting is loaded
export const isHighlightingReady = (): boolean => {
  return isHighlightingLoaded;
};

// Get the highlight.js module directly (for advanced use cases)
export const getHighlightJs = () => {
  return hljsModule;
};
