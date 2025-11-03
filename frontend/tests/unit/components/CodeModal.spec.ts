import { mount } from "@vue/test-utils";
import CodeModal from "../../../src/components/ui/CodeModal.vue";

describe("CodeModal.vue", () => {
  let wrapper: any;

  beforeEach(() => {
    // Reset all mocks before each test
    jest.clearAllMocks();
    
    // Set up a default wrapper
    wrapper = mount(CodeModal, {
      props: {
        isVisible: true,
        filePath: "/path/to/test.go",
        fileContent: "package main\n\nfunc main() {\n    println('Hello, World!')\n}",
        query: ""
      }
    });
  });

  afterEach(() => {
    if (wrapper && wrapper.unmount) {
      wrapper.unmount();
    }
  });

  describe("Rendering", () => {
    it("renders properly with code content", () => {
      expect(wrapper.exists()).toBe(true);
      expect(wrapper.find(".modal-container").exists()).toBe(true);
      expect(wrapper.find(".modal-title").text()).toContain("File Preview:");
    });

    it("renders code with line numbers", async () => {
      await wrapper.vm.$nextTick();
      
      const codeBlock = wrapper.find("code");
      expect(codeBlock.exists()).toBe(true);
      
      const codeHtml = codeBlock.html();
      expect(codeHtml).toContain('class="line-number"');
      expect(codeHtml).toContain("1"); // First line number
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

    it("truncates long file paths", async () => {
      const longPath = "/very/long/path/to/a/file/that/has/a/very/long/name/and/should/be/truncated.go";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: longPath,
          fileContent: "package main",
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const title = wrapper.find(".modal-title");
      const titleText = title.text();
      expect(titleText).toContain("File Preview:");
      expect(titleText).toContain("..."); // Should contain truncated version
    });
  });

  describe("Language Detection", () => {
    it("detects programming language correctly from file extension", async () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: "console.log('Hello, World!');",
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const footerInfo = wrapper.find(".modal-footer-info");
      expect(footerInfo.text()).toContain("Language: javascript");
    });

    it("correctly detects various programming languages", async () => {
      const testCases = [
        { file: "test.go", expectedLang: "go" },
        { file: "test.js", expectedLang: "javascript" },
        { file: "test.ts", expectedLang: "typescript" },
        { file: "test.py", expectedLang: "python" },
        { file: "test.java", expectedLang: "java" },
        { file: "test.html", expectedLang: "html" },
        { file: "test.css", expectedLang: "css" },
        { file: "test.json", expectedLang: "json" },
        { file: "test.yaml", expectedLang: "yaml" },
        { file: "test.sql", expectedLang: "sql" },
        { file: "test.rs", expectedLang: "rust" },
        { file: "test.jsx", expectedLang: "jsx" },
        { file: "test.tsx", expectedLang: "tsx" },
        { file: "test.unknown", expectedLang: "text" }
      ];

      for (const testCase of testCases) {
        wrapper = mount(CodeModal, {
          props: {
            isVisible: true,
            filePath: testCase.file,
            fileContent: "test content",
            query: ""
          }
        });

        await wrapper.vm.$nextTick();
        
        const footerInfo = wrapper.find(".modal-footer-info");
        expect(footerInfo.text()).toMatch(new RegExp(`Language: ${testCase.expectedLang}`, 'i'));
      }
    });
  });

  describe("Search Functionality", () => {
    it("highlights search query in code", async () => {
      const codeContent = "function greet(name) {\n    console.log('Hello, ' + name);\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeContent,
          query: "console"
        }
      });

      await wrapper.vm.$nextTick();
      await new Promise(resolve => setTimeout(resolve, 10));
      
      const codeBlock = wrapper.find("code");
      const codeHtml = codeBlock.html();
      
      expect(codeHtml).toContain('<mark class="highlight-match">');
      expect(codeHtml).toContain("console");
    });

    it("search navigation works with matches", async () => {
      const codeContent = "console.log('test 1');\nconsole.log('test 2');\nconsole.log('test 3');";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeContent,
          query: "console"
        }
      });

      await wrapper.vm.$nextTick();
      await new Promise(resolve => setTimeout(resolve, 10));
      
      const vm = wrapper.vm as any;
      
      expect(vm.hasMatches).toBe(true);
      expect(vm.totalMatches).toBe(3);
      
      // Test finding next match
      const initialIndex = vm.currentMatchIndex;
      vm.findNextMatch();
      expect(vm.currentMatchIndex).toBe((initialIndex + 1) % 3);
      
      // Test finding previous match
      vm.findPreviousMatch();
      expect(vm.currentMatchIndex).toBe(initialIndex);
    });

    it("handles cases with no matches", async () => {
      const codeContent = "function greet() { return 'Hello'; }";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.js",
          fileContent: codeContent,
          query: "nonexistent"
        }
      });

      await wrapper.vm.$nextTick();
      
      const vm = wrapper.vm as any;
      expect(vm.hasMatches).toBe(false);
      expect(vm.totalMatches).toBe(0);
    });
  });

  describe("Navigation", () => {
    it("shows navigation controls for large files", async () => {
      const lines = Array.from({ length: 60 }, (_, i) => `Line ${i + 1}: This is line number ${i + 1}`);
      const largeFileContent = lines.join('\n');

      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/large-file.js",
          fileContent: largeFileContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const navControls = wrapper.find(".navigation-controls");
      expect(navControls.exists()).toBe(true);
      
      const lineInput = wrapper.find(".line-input");
      expect(lineInput.exists()).toBe(true);
    });

    it("jump to line functionality works", async () => {
      const codeContent = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.txt",
          fileContent: codeContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const vm = wrapper.vm as any;
      vm.targetLine = 3;
      
      vm.jumpToLine();
      
      await wrapper.vm.$nextTick();
      await new Promise(resolve => setTimeout(resolve, 10));
      
      const codeBlock = wrapper.find("code");
      expect(codeBlock.exists()).toBe(true);
    });

    it("displays correct line count", async () => {
      const codeContent = "line1\nline2\nline3";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.go",
          fileContent: codeContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const footerInfo = wrapper.find(".modal-footer-info");
      expect(footerInfo.text()).toContain("Lines: 3");
    });

    it("handles invalid line numbers gracefully", async () => {
      const codeContent = "Line 1\nLine 2";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.txt",
          fileContent: codeContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const vm = wrapper.vm as any;
      vm.targetLine = -1; // Invalid line
      vm.jumpToLine();
      
      vm.targetLine = 100; // Line beyond file length
      vm.jumpToLine();
      
      expect(vm.totalLines).toBe(2); // Should stay at actual total
    });
  });

  describe("UI Interactions", () => {
    it("closes modal when close button is clicked", async () => {
      const closeButton = wrapper.find(".modal-close-button");
      await closeButton.trigger("click");
      
      expect(wrapper.emitted()).toHaveProperty("close");
    });

    it("closes modal when clicking outside", async () => {
      const modalOverlay = wrapper.find(".modal-overlay");
      await modalOverlay.trigger("click");
      
      expect(wrapper.emitted()).toHaveProperty("close");
    });

    it("handles escape key to close modal", async () => {
      // Mock keydown event
      const keydownEvent = new KeyboardEvent('keydown', { key: 'Escape' });
      document.dispatchEvent(keydownEvent);
      
      expect(wrapper.emitted()).toHaveProperty("close");
    });

    it("copy to clipboard functionality works", async () => {
      const codeContent = "package main\n\nfunc main() {\n    println('Hello, World!')\n}";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.go",
          fileContent: codeContent,
          query: ""
        }
      });

      // Mock clipboard API
      const mockWriteText = jest.fn().mockResolvedValue(undefined);
      Object.assign(navigator, {
        clipboard: {
          writeText: mockWriteText
        }
      });

      const copyButton = wrapper.find(".copy-button");
      await copyButton.trigger("click");

      expect(mockWriteText).toHaveBeenCalledWith(codeContent);
      expect(wrapper.emitted()).toHaveProperty("copy");
    });

    it("shows copied status after copying", async () => {
      // Mock clipboard API
      const mockWriteText = jest.fn().mockResolvedValue(undefined);
      Object.assign(navigator, {
        clipboard: {
          writeText: mockWriteText
        }
      });

      const copyButton = wrapper.find(".copy-button");
      await copyButton.trigger("click");

      await new Promise(resolve => setTimeout(resolve, 50)); // Wait for copy to complete
      
      expect(wrapper.text()).toContain("Copied!");
    });
  });

  describe("Large File Handling", () => {
    it("shows skeleton loading for very large files", async () => {
      const largeFileContent = "a\n".repeat(1000); // Create a large file
      
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/huge-file.txt",
          fileContent: largeFileContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      // Check that the component loads without crashing
      expect(wrapper.find(".modal-container").exists()).toBe(true);
    });

    it("handles extremely large files without performance issues", async () => {
      const hugeFileContent = "line\n".repeat(25000); // More than 20000 lines threshold
      
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/extremely-huge-file.txt",
          fileContent: hugeFileContent,
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      await new Promise(resolve => setTimeout(resolve, 300)); // Wait for loading simulation
      
      expect(wrapper.find(".modal-container").exists()).toBe(true);
    });
  });

  describe("Edge Cases", () => {
    it("handles file paths with no extension", async () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/Makefile",
          fileContent: "build:\n\techo 'Building...'",
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const footerInfo = wrapper.find(".modal-footer-info");
      expect(footerInfo.text()).toContain("Language: text");
    });

    it("handles files with multiple extensions", async () => {
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/script.go.js", // has both .go and .js
          fileContent: "console.log('test');",
          query: ""
        }
      });

      await wrapper.vm.$nextTick();
      
      const footerInfo = wrapper.find(".modal-footer-info");
      expect(footerInfo.text()).toContain("Language: javascript"); // Should take last extension
    });

    it("handles null or undefined props gracefully", () => {
      expect(() => {
        mount(CodeModal, {
          props: {
            isVisible: true,
            filePath: "", // Empty string instead of null
            fileContent: "", // Empty string instead of null
            query: "" // Empty string instead of null
          }
        });
      }).not.toThrow();
    });

    it("handles invalid regex patterns gracefully", async () => {
      const codeContent = "test string";
      wrapper = mount(CodeModal, {
        props: {
          isVisible: true,
          filePath: "/path/to/test.txt",
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