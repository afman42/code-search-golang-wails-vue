import { mount } from "@vue/test-utils";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { SearchSuggestions } from '@/components/ui';

const STORAGE_KEY = "codeSearchRecentSearches";

const seedStorage = () => {
  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify([
      { query: "hello", extension: "go", directory: "/mock/project" },
      { query: "world", extension: "", directory: "/mock/project" },
    ])
  );
};

describe("SearchSuggestions.vue", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
    seedStorage();
  });

  afterEach(() => {
    document.body.innerHTML = "";
    localStorage.clear();
    vi.restoreAllMocks();
  });

  const mountSuggestions = (show = true) =>
    mount(SearchSuggestions, {
      props: { show },
    });

  it("renders suggestions loaded from localStorage", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    expect(wrapper.findAll(".suggestion-item")).toHaveLength(2);
    expect(wrapper.text()).toContain("hello");
  });

  it("does not render anything when there are no suggestions", async () => {
    localStorage.clear();
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    expect(wrapper.find(".search-suggestions").exists()).toBe(false);
  });

  it("shows the extension and directory for entries that carry them", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    expect(wrapper.text()).toContain("go");
    expect(wrapper.text()).toContain("project");
  });

  it("emits select with the full search entry when a suggestion is picked", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    await wrapper.findAll(".suggestion-item")[0].trigger("mousedown");
    expect(wrapper.emitted("select")).toBeTruthy();
    expect(wrapper.emitted("select")[0]).toEqual([
      { query: "hello", extension: "go", directory: "/mock/project" },
    ]);
  });

  it("emits remove and updates localStorage on delete", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    await wrapper.find(".suggestion-delete").trigger("mousedown");
    expect(wrapper.emitted("remove")).toBeTruthy();
    expect(wrapper.emitted("remove")[0]).toEqual(["hello"]);
    expect(localStorage.getItem(STORAGE_KEY)).not.toContain("hello");
    expect(wrapper.findAll(".suggestion-item")).toHaveLength(1);
  });

  it("removes only the matching entry, keeping other directories intact", async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify([
        { query: "hello", extension: "go", directory: "/a" },
        { query: "hello", extension: "go", directory: "/b" },
      ])
    );
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    await wrapper.findAll(".suggestion-item")[0].trigger("mousedown", { button: 0 });
    await wrapper.find(".suggestion-delete").trigger("mousedown");
    const remaining = JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]");
    expect(remaining).toEqual([{ query: "hello", extension: "go", directory: "/b" }]);
  });

  it("emits close when clicking outside the dropdown", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    document.body.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true }));
    expect(wrapper.emitted("close")).toBeTruthy();
  });

  it("does not emit close when clicking inside the dropdown", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    await wrapper.find(".suggestion-item").trigger("pointerdown");
    expect(wrapper.emitted("close")).toBeUndefined();
  });

  it("emits close on Escape key", async () => {
    const wrapper = mountSuggestions();
    await wrapper.vm.$nextTick();
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    expect(wrapper.emitted("close")).toBeTruthy();
  });

  it("removes document listeners on unmount", () => {
    const spy = vi.spyOn(document, "removeEventListener");
    const wrapper = mountSuggestions();
    wrapper.unmount();
    expect(spy).toHaveBeenCalledWith("pointerdown", expect.any(Function));
    expect(spy).toHaveBeenCalledWith("keydown", expect.any(Function));
  });
});
