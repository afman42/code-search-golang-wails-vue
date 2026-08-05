import { describe, test, expect, vi, beforeEach } from "vitest";
import { ref, nextTick } from "vue";
import { useMatchNavigation } from "../../../src/composables/useMatchNavigation";

// jsdom doesn't do layout, so getBoundingClientRect returns zeros by default.
// We stub each element's rect explicitly to control the vertical positions used
// by getAllMatchPositions for sorting and scroll comparisons. Positions are
// kept strictly positive so the scroll-based branch of goToNextMatch picks the
// first match when scrollTop is 0.

interface ContainerResult {
  container: HTMLElement;
  matchEls: HTMLElement[];
}

function createContainer(
  matchCount: number,
  positions: number[] = [],
): ContainerResult {
  const container = document.createElement("div");
  container.scrollTop = 0;

  const matchEls: HTMLElement[] = [];
  for (let i = 0; i < matchCount; i++) {
    const el = document.createElement("span");
    el.classList.add("highlight-match");
    const top = positions[i] ?? (i + 1) * 100;
    el.getBoundingClientRect = () =>
      ({
        top,
        bottom: top + 10,
        left: 0,
        right: 0,
        width: 10,
        height: 10,
      }) as DOMRect;
    // Per-instance scrollIntoView mock so we can assert exactly which element
    // was scrolled without relying on prototype-spy instance tracking.
    el.scrollIntoView = vi.fn();
    container.appendChild(el);
    matchEls.push(el);
  }

  container.getBoundingClientRect = () =>
    ({
      top: 0,
      bottom: 0,
      left: 0,
      right: 0,
      width: 0,
      height: 0,
    }) as DOMRect;

  return { container, matchEls };
}

// Assert none of the per-instance scrollIntoView mocks were called.
function expectNoScroll(matchEls: HTMLElement[]) {
  for (const el of matchEls) {
    expect(el.scrollIntoView).not.toHaveBeenCalled();
  }
}

describe("useMatchNavigation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("totalMatches", () => {
    test("returns 0 when query is empty", () => {
      const content = ref("hello world");
      const query = ref("");
      const { totalMatches } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      expect(totalMatches()).toBe(0);
    });

    test("returns 0 when content is empty", () => {
      const content = ref("");
      const query = ref("hello");
      const { totalMatches } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      expect(totalMatches()).toBe(0);
    });

    test("returns count of case-insensitive matches", () => {
      const content = ref("Hello hello HELLO world");
      const query = ref("hello");
      const { totalMatches } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      expect(totalMatches()).toBe(3);
    });

    test("escapes regex special characters in query (literal dot)", () => {
      const content = ref("a.b.c (test) [bracket] $end");
      const query = ref("a.b");
      const { totalMatches } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      // If special chars weren't escaped, `.` would match any char and yield
      // more hits. With escaping, "a.b" is treated literally → 1 match.
      expect(totalMatches()).toBe(1);
    });

    test("escapes parentheses and brackets literally", () => {
      const content = ref("(test) [x] (test)");
      const query = ref("(test)");
      const { totalMatches } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      expect(totalMatches()).toBe(2);
    });

    test("returns 0 on invalid regex (no throw)", () => {
      // `(` is escaped to `\(` before reaching RegExp, so no throw; the
      // literal "(*invalid" is not present in content → 0 matches.
      const content = ref("some content");
      const query = ref("(*invalid");
      const { totalMatches } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      expect(() => totalMatches()).not.toThrow();
      expect(totalMatches()).toBe(0);
    });
  });

  describe("goToNextMatch", () => {
    test("no-op when no matches in content", () => {
      const content = ref("nothing here");
      const query = ref("zzz");
      const { container, matchEls } = createContainer(0);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch();

      expectNoScroll(matchEls);
      expect(currentMatchIndex.value).toBe(0);
    });

    test("no-op when query is empty", () => {
      const content = ref("hello hello");
      const query = ref("");
      const { container, matchEls } = createContainer(2, [10, 110]);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch();

      expectNoScroll(matchEls);
      expect(currentMatchIndex.value).toBe(0);
    });

    test("no-op when container is null", () => {
      const content = ref("hello hello");
      const query = ref("hello");
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      goToNextMatch();

      expect(currentMatchIndex.value).toBe(0);
    });

    test("scrolls first match into view on initial next", () => {
      const content = ref("hello hello hello");
      const query = ref("hello");
      const { container, matchEls } = createContainer(3, [10, 110, 210]);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch();

      // scrollTop is 0; first position (10) > 0 → nextIndex 0 → match[0].
      expect(matchEls[0].scrollIntoView).toHaveBeenCalledTimes(1);
      expect(matchEls[0].scrollIntoView).toHaveBeenCalledWith({
        behavior: "smooth",
        block: "center",
      });
      expect(matchEls[1].scrollIntoView).not.toHaveBeenCalled();
      expect(matchEls[2].scrollIntoView).not.toHaveBeenCalled();
      expect(currentMatchIndex.value).toBe(1);
    });

    test("advances to next match and scrolls it into view", () => {
      const content = ref("hello hello hello");
      const query = ref("hello");
      const { container, matchEls } = createContainer(3, [10, 110, 210]);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch(); // → currentMatchIndex 1 (match[0])
      expect(matchEls[1].scrollIntoView).not.toHaveBeenCalled();
      goToNextMatch(); // currentMatchIndex 1 > 0 and < 3 → nextIndex 1 (match[1])

      expect(matchEls[1].scrollIntoView).toHaveBeenCalledTimes(1);
      expect(matchEls[1].scrollIntoView).toHaveBeenCalledWith({
        behavior: "smooth",
        block: "center",
      });
      expect(currentMatchIndex.value).toBe(2);
    });

    test("wraps to first match (index 0) at end", () => {
      const content = ref("hello hello hello");
      const query = ref("hello");
      const { container, matchEls } = createContainer(3, [10, 110, 210]);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch(); // → 1 (match[0])
      goToNextMatch(); // → 2 (match[1])
      goToNextMatch(); // currentMatchIndex 2 > 0 and < 3 → nextIndex 2 (match[2]) → 3
      expect(currentMatchIndex.value).toBe(3);

      // Reset per-instance mocks to isolate the wrap call.
      matchEls.forEach((el) => (el.scrollIntoView as ReturnType<typeof vi.fn>).mockClear());

      goToNextMatch(); // currentMatchIndex 3 === length 3 → wrap to nextIndex 0

      expect(matchEls[0].scrollIntoView).toHaveBeenCalledTimes(1);
      expect(matchEls[1].scrollIntoView).not.toHaveBeenCalled();
      expect(matchEls[2].scrollIntoView).not.toHaveBeenCalled();
      expect(currentMatchIndex.value).toBe(1);
    });
  });

  describe("goToPreviousMatch", () => {
    test("wraps to last match when currentMatchIndex is 0 (<= 1)", () => {
      const content = ref("hello hello hello");
      const query = ref("hello");
      const { container, matchEls } = createContainer(3, [10, 110, 210]);
      const { goToPreviousMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      // currentMatchIndex starts at 0 (<= 1) → wraps to last match.
      goToPreviousMatch();

      expect(matchEls[2].scrollIntoView).toHaveBeenCalledTimes(1);
      expect(matchEls[2].scrollIntoView).toHaveBeenCalledWith({
        behavior: "smooth",
        block: "center",
      });
      // prevIndex = length - 1 = 2; currentMatchIndex becomes prevIndex + 1 = 3.
      expect(currentMatchIndex.value).toBe(3);
    });

    test("wraps to last match when currentMatchIndex === 1", () => {
      const content = ref("hello hello hello");
      const query = ref("hello");
      const { container, matchEls } = createContainer(3, [10, 110, 210]);
      const { goToPreviousMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      currentMatchIndex.value = 1;
      goToPreviousMatch();

      expect(matchEls[2].scrollIntoView).toHaveBeenCalledTimes(1);
      expect(currentMatchIndex.value).toBe(3);
    });

    test("moves to previous match when currentMatchIndex > 1", () => {
      const content = ref("hello hello hello");
      const query = ref("hello");
      const { container, matchEls } = createContainer(3, [10, 110, 210]);
      const { goToPreviousMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      // Simulate being on the 3rd match (currentMatchIndex = 3).
      currentMatchIndex.value = 3;
      goToPreviousMatch();

      // prevIndex = currentMatchIndex - 2 = 1 (the 2nd match).
      expect(matchEls[1].scrollIntoView).toHaveBeenCalledTimes(1);
      // currentMatchIndex becomes prevIndex + 1 = 2.
      expect(currentMatchIndex.value).toBe(2);
    });

    test("no-op when no matches", () => {
      const content = ref("nothing here");
      const query = ref("zzz");
      const { container, matchEls } = createContainer(0);
      const { goToPreviousMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToPreviousMatch();

      expectNoScroll(matchEls);
      expect(currentMatchIndex.value).toBe(0);
    });

    test("no-op when container is null", () => {
      const content = ref("hello hello");
      const query = ref("hello");
      const { goToPreviousMatch, currentMatchIndex } = useMatchNavigation(
        () => null,
        () => content.value,
        () => query.value,
      );

      goToPreviousMatch();

      expect(currentMatchIndex.value).toBe(0);
    });
  });

  describe("watch([fileContent, query])", () => {
    test("resets currentMatchIndex to 0 when content changes", async () => {
      const content = ref("hello hello");
      const query = ref("hello");
      const { container } = createContainer(2, [10, 110]);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch();
      expect(currentMatchIndex.value).toBe(1);

      content.value = "changed content";
      await nextTick();

      expect(currentMatchIndex.value).toBe(0);
    });

    test("resets currentMatchIndex to 0 when query changes", async () => {
      const content = ref("hello hello");
      const query = ref("hello");
      const { container } = createContainer(2, [10, 110]);
      const { goToNextMatch, currentMatchIndex } = useMatchNavigation(
        () => container,
        () => content.value,
        () => query.value,
      );

      goToNextMatch();
      expect(currentMatchIndex.value).toBe(1);

      query.value = "world";
      await nextTick();

      expect(currentMatchIndex.value).toBe(0);
    });

    test("clears observer and match state on content change", async () => {
      const content = ref("hello hello");
      const query = ref("hello");
      const { container } = createContainer(2, [10, 110]);
      const { observer, visibleMatches, matchElements, refreshMatchObserver } =
        useMatchNavigation(
          () => container,
          () => content.value,
          () => query.value,
        );

      await refreshMatchObserver();
      expect(observer.value).not.toBeNull();
      expect(matchElements.value).toHaveLength(2);

      content.value = "changed";
      await nextTick();

      expect(observer.value).toBeNull();
      expect(visibleMatches.value.size).toBe(0);
      expect(matchElements.value).toEqual([]);
    });

    test("clears observer and match state on query change", async () => {
      const content = ref("hello hello");
      const query = ref("hello");
      const { container } = createContainer(2, [10, 110]);
      const { observer, visibleMatches, matchElements, refreshMatchObserver } =
        useMatchNavigation(
          () => container,
          () => content.value,
          () => query.value,
        );

      await refreshMatchObserver();
      expect(observer.value).not.toBeNull();

      query.value = "world";
      await nextTick();

      expect(observer.value).toBeNull();
      expect(visibleMatches.value.size).toBe(0);
      expect(matchElements.value).toEqual([]);
    });
  });
});
