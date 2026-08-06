import { mount } from "@vue/test-utils";
import { describe, it, expect } from "vitest";
import { TreeViewPanel } from '@/components/ui';

const FILES = [
  "/project/README.md",
  "/project/src/index.ts",
  "/project/src/utils/helper.ts",
];

describe("TreeViewPanel.vue", () => {
  const mountPanel = (props = {}) =>
    mount(TreeViewPanel, {
      props: {
        isVisible: true,
        currentFilePath: null,
        ...props,
      },
    });

  // Item counts render as "name (N)" on folders; extract the bare name.
  const names = (wrapper: ReturnType<typeof mountPanel>): string[] =>
    wrapper
      .findAll(".tree-item-name")
      .map((el) => el.text().replace(/\s*\(\d+\)\s*$/, "").trim());

  describe("rendering", () => {
    it("shows an empty state when there are no files", () => {
      const wrapper = mountPanel({ files: [] });
      expect(wrapper.find(".tree-empty").exists()).toBe(true);
      expect(wrapper.text()).toContain("No files found for this search.");
    });

    it("builds a directory tree from file paths", () => {
      const wrapper = mountPanel({ files: FILES });
      const rendered = names(wrapper);
      expect(rendered).toContain("project");
      expect(rendered).toContain("src");
      expect(rendered).toContain("utils");
      expect(rendered).toContain("index.ts");
      expect(rendered).toContain("helper.ts");
      expect(rendered).toContain("README.md");
    });

    it("renders directories before files and sorts alphabetically", () => {
      const wrapper = mountPanel({
        files: ["/b.txt", "/a_dir/c.txt", "/a.txt"],
      });
      const rendered = names(wrapper);
      // Depth-first render: the a_dir folder expands inline before siblings.
      expect(rendered[0]).toBe("a_dir");
      expect(rendered[1]).toBe("c.txt");
      expect(rendered[2]).toBe("a.txt");
      expect(rendered[3]).toBe("b.txt");
    });

    it("shows file and folder counts in the summary", () => {
      const wrapper = mountPanel({ files: FILES });
      const summary = wrapper.find(".tree-summary").text();
      expect(summary).toContain("3 files");
      expect(summary).toContain("3 folders");
    });

    it("highlights the current file", () => {
      const wrapper = mountPanel({
        files: FILES,
        currentFilePath: "/project/src/index.ts",
      });
      const active = wrapper.find(".tree-item-header.current-file");
      expect(active.exists()).toBe(true);
      expect(active.text()).toContain("index.ts");
    });
  });

  describe("file selection", () => {
    it("emits file-click with the file path when a file row is clicked", async () => {
      const wrapper = mountPanel({ files: FILES });
      const firstFile = wrapper.findAll(".tree-item-header.is-file")[0];
      await firstFile.trigger("click");
      expect(wrapper.emitted("fileClick")).toBeTruthy();
      expect(wrapper.emitted("fileClick")[0]).toEqual([
        "/project/src/utils/helper.ts",
      ]);
    });

    it("does not emit file-click when a folder is clicked", async () => {
      const wrapper = mountPanel({ files: FILES });
      const folder = wrapper.find(".tree-item-header.is-folder");
      await folder.trigger("click");
      expect(wrapper.emitted("fileClick")).toBeUndefined();
    });
  });
});
