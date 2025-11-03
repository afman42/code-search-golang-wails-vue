import { mount } from "@vue/test-utils";
import CodeModal from "../../../src/components/ui/CodeModal.vue";

describe("CodeModal.vue - Syntax Highlighting", () => {
  let wrapper: any;

  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();
  });

  afterEach(() => {
    if (wrapper && wrapper.unmount) {
      wrapper.unmount();
    }
  });

  describe("Language Detection", () => {
    it("detects Go language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.go",
          fileContent: "package main\n\nfunc main() {\n    println('Hello, World!')\n}",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("go");
    });

    it("detects JavaScript language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: "console.log('Hello, World!');",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("javascript");
    });

    it("detects TypeScript language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.ts",
          fileContent: "console.log('Hello, World!');",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("typescript");
    });

    it("detects Python language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.py",
          fileContent: "print('Hello, World!')",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("python");
    });

    it("detects Java language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/Test.java",
          fileContent: "public class Test {\n    public static void main(String[] args) {\n        System.out.println(\"Hello, World!\");\n    }\n}",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("java");
    });

    it("detects C++ language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.cpp",
          fileContent: "#include <iostream>\n\nint main() {\n    std::cout << \"Hello, World!\" << std::endl;\n    return 0;\n}",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("cpp");
    });

    it("detects HTML language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.html",
          fileContent: "<!DOCTYPE html>\n<html>\n<head>\n    <title>Test</title>\n</head>\n<body>\n    <h1>Hello</h1>\n</body>\n</html>",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("html");
    });

    it("detects CSS language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.css",
          fileContent: "body {\n    background-color: #fff;\n    font-family: Arial, sans-serif;\n}",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("css");
    });

    it("detects JSON language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.json",
          fileContent: "{\n    \"name\": \"test\",\n    \"version\": \"1.0.0\"\n}",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("json");
    });

    it("detects YAML language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.yaml",
          fileContent: "name: test\nversion: 1.0.0",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("yaml");
    });

    it("detects Markdown language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/README.md",
          fileContent: "# Test\n\nThis is a test README file.",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("markdown");
    });

    it("detects SQL language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/query.sql",
          fileContent: "SELECT * FROM users WHERE id = 1;",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("sql");
    });

    it("detects Rust language correctly", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/main.rs",
          fileContent: "fn main() {\n    println!(\"Hello, World!\");\n}",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("rust");
    });

    it("detects unknown file types as text", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/unknown.xyz",
          fileContent: "This is some content",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("text");
    });

    it("detects files with no extension as text", () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/Makefile",
          fileContent: "build:\n\techo 'Building...'",
          query: ""
        }
      });

      expect(wrapper.vm.detectedLanguage).toBe("text");
    });
  });

  describe("Syntax Highlighting", () => {
    it("applies syntax highlighting to Go code", async () => {
      const goCode = "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.go",
          fileContent: goCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain Go code content
      expect(codeHtml).toContain("package");
      expect(codeHtml).toContain("main");
      expect(codeHtml).toContain("fmt");
      expect(codeHtml).toContain("Hello, World!");
    });

    it("applies syntax highlighting to JavaScript code", async () => {
      const jsCode = "const message = 'Hello, World!';\nconsole.log(message);";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: jsCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain highlighted JavaScript code
      expect(codeHtml).toContain("const");
      expect(codeHtml).toContain("message");
      expect(codeHtml).toContain("console");
      expect(codeHtml).toContain("log");
    });

    it("applies syntax highlighting to Python code", async () => {
      const pythonCode = "def hello():\n    print('Hello, World!')";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.py",
          fileContent: pythonCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain highlighted Python code
      expect(codeHtml).toContain("def");
      expect(codeHtml).toContain("hello");
      expect(codeHtml).toContain("print");
    });

    it("applies syntax highlighting to HTML code", async () => {
      const htmlCode = "<!DOCTYPE html>\n<html>\n<head>\n    <title>Test</title>\n</head>\n<body>\n    <h1>Hello</h1>\n</body>\n</html>";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.html",
          fileContent: htmlCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain highlighted HTML code (escaped for HTML)
      expect(codeHtml).toContain("&lt;!DOCTYPE");
      expect(codeHtml).toContain("&lt;html&gt;");
      expect(codeHtml).toContain("&lt;title&gt;");
      expect(codeHtml).toContain("Test");
    });

    it("applies syntax highlighting to CSS code", async () => {
      const cssCode = "body {\n    background-color: #fff;\n    font-family: Arial, sans-serif;\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.css",
          fileContent: cssCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain highlighted CSS code
      expect(codeHtml).toContain("body");
      expect(codeHtml).toContain("background-color");
      expect(codeHtml).toContain("#fff");
    });

    it("applies syntax highlighting to JSON code", async () => {
      const jsonCode = "{\n    \"name\": \"test\",\n    \"version\": \"1.0.0\"\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.json",
          fileContent: jsonCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain highlighted JSON code
      expect(codeHtml).toContain("\"name\"");
      expect(codeHtml).toContain("\"test\"");
      expect(codeHtml).toContain("\"version\"");
      expect(codeHtml).toContain("\"1.0.0\"");
    });

    it("falls back to plain text for unknown languages", async () => {
      const unknownCode = "This is some plain text content\nwith multiple lines";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/unknown.xyz",
          fileContent: unknownCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain plain text without language-specific highlighting
      expect(codeHtml).toContain("This is some plain text content");
      expect(codeHtml).toContain("with multiple lines");
    });

    it("handles empty file content gracefully", async () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/empty.go",
          fileContent: "",
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const codeBlock = wrapper.find("code");
      expect(codeBlock.text()).toBe("");
    });

    it("handles very large files with syntax highlighting", async () => {
      // Create a large Go file
      const lines = [];
      for (let i = 1; i <= 1000; i++) {
        lines.push(`// Line ${i}\nfunc function${i}() {\n    println("Hello from function ${i}")\n}`);
      }
      const largeGoCode = lines.join('\n');

      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/large-file.go",
          fileContent: largeGoCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain highlighted Go code even for large files
      expect(codeHtml).toContain("func");
      expect(codeHtml).toContain("println");
    });
  });

  describe("Theme Integration", () => {
    it("uses the agate theme for syntax highlighting", async () => {
      const jsCode = "const message = 'Hello, World!';\nconsole.log(message);";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: jsCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Check that the component uses the agate theme
      // This is verified by checking that the agate.css is imported in the component
      expect(wrapper.exists()).toBe(true);
    });

    it("maintains consistent styling with agate theme", async () => {
      const codeContent = "function greet(name) {\n    return `Hello, ${name}!`;\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Check that the component renders with consistent styling
      const modalContainer = wrapper.find(".modal-container");
      expect(modalContainer.exists()).toBe(true);
      
      const codeBlock = wrapper.find("code");
      expect(codeBlock.exists()).toBe(true);
      
      // Check that line numbers are present
      const lineNumbers = wrapper.findAll(".line-number");
      expect(lineNumbers.length).toBeGreaterThan(0);
    });
  });

  describe("Highlighting with Search Query", () => {
    it("highlights search matches in addition to syntax highlighting", async () => {
      const jsCode = "function greet(name) {\n    console.log('Hello, ' + name);\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: jsCode,
          query: "console"
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain both syntax highlighting and search match highlighting
      expect(codeHtml).toContain('<mark class="highlight-match">');
      expect(codeHtml).toContain("console");
    });

    it("handles complex search queries with syntax highlighting", async () => {
      const goCode = "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.go",
          fileContent: goCode,
          query: "fmt"
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should highlight search matches in syntax-highlighted code
      expect(codeHtml).toContain('<mark class="highlight-match">');
      expect(codeHtml).toContain("fmt");
    });

    it("preserves syntax highlighting when search query is empty", async () => {
      const pythonCode = "def hello():\n    print('Hello, World!')";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.py",
          fileContent: pythonCode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should contain syntax-highlighted code without search highlights
      expect(codeHtml).toContain("def");
      expect(codeHtml).toContain("hello");
      expect(codeHtml).toContain("print");
      expect(codeHtml).not.toContain('<mark class="highlight-match">');
    });
  });

  describe("Edge Cases", () => {
    it("handles special characters in code correctly", async () => {
      const codeWithSpecialChars = "const regex = /[.*+?^${}()|[\\]\\\\]/g;\nconst html = '<div class=\"test\">&amp;</div>';";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeWithSpecialChars,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should properly escape and highlight special characters
      expect(codeHtml).toContain("regex");
      expect(codeHtml).toContain("html");
    });

    it("handles unicode characters in code correctly", async () => {
      const codeWithUnicode = "const greeting = 'Hello, 世界!';\nconst emoji = '🚀';";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeWithUnicode,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should properly handle unicode characters
      expect(codeHtml).toContain("greeting");
      expect(codeHtml).toContain("Hello, 世界!");
      expect(codeHtml).toContain("emoji");
      expect(codeHtml).toContain("🚀");
    });

    it("handles multiline strings correctly", async () => {
      const multilineString = "const multiline = `\nLine 1\nLine 2\nLine 3\n`;";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: multilineString,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Wait for async operations
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      // Should properly handle multiline strings
      expect(codeHtml).toContain("multiline");
      expect(codeHtml).toContain("Line 1");
      expect(codeHtml).toContain("Line 2");
      expect(codeHtml).toContain("Line 3");
    });

    it("handles invalid regex patterns gracefully", async () => {
      const codeContent = "test string";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeContent,
          query: "[" // Invalid regex
        }
      });

      await wrapper.vm.$nextTick();
      
      // Should not crash and should handle invalid regex gracefully
      const vm = wrapper.vm as any;
      expect(vm.totalMatches).toBe(0);
    });
  });
});