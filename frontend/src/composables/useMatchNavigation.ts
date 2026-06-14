import { ref, watch, onMounted, onUnmounted, nextTick } from "vue";

export function useMatchNavigation(
  codeContainerRef: () => HTMLElement | null,
  fileContent: () => string,
  query: () => string,
) {
  const currentMatchIndex = ref(0);
  const observer = ref<IntersectionObserver | null>(null);
  const visibleMatches = ref<Set<Element>>(new Set());
  const matchElements = ref<Element[]>([]);

  const totalMatches = () => {
    const q = query();
    const content = fileContent();
    if (!q || !content) return 0;

    try {
      const regex = new RegExp(
        q.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
        "gi",
      );
      const matches = content.match(regex);
      return matches ? matches.length : 0;
    } catch {
      return 0;
    }
  };

  const initIntersectionObserver = () => {
    const container = codeContainerRef();
    if (!container) return;

    if (observer.value) {
      observer.value.disconnect();
    }

    observer.value = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            visibleMatches.value.add(entry.target);
          } else {
            visibleMatches.value.delete(entry.target);
          }
        });
      },
      {
        root: container,
        rootMargin: "100px",
        threshold: 0.1,
      },
    );
  };

  const refreshMatchObserver = async () => {
    const container = codeContainerRef();
    if (!container) return;
    await nextTick();

    const matches = container.querySelectorAll(".highlight-match");
    matchElements.value = Array.from(matches);

    initIntersectionObserver();

    matchElements.value.forEach((match) => {
      observer.value?.observe(match);
    });
  };

  const getAllMatchPositions = () => {
    const container = codeContainerRef();
    if (!container) return [];

    const matches = container.querySelectorAll(".highlight-match");
    matchElements.value = Array.from(matches);

    const positions: {
      element: Element;
      index: number;
      position: number;
    }[] = [];

    matchElements.value.forEach((element, i) => {
      const rect = element.getBoundingClientRect();
      const containerRect = container.getBoundingClientRect();
      const position =
        rect.top - containerRect.top + container.scrollTop;
      positions.push({ element, index: i, position });
    });

    positions.sort((a, b) => a.position - b.position);
    return positions;
  };

  const goToNextMatch = () => {
    if (!query() || !fileContent()) return;
    const container = codeContainerRef();
    if (!container) return;

    const matchPositions = getAllMatchPositions();
    if (matchPositions.length === 0) return;

    let nextIndex = 0;

    if (
      currentMatchIndex.value > 0 &&
      currentMatchIndex.value < matchPositions.length
    ) {
      nextIndex = currentMatchIndex.value;
    } else if (currentMatchIndex.value === matchPositions.length) {
      nextIndex = 0;
    } else {
      const currentScrollTop = container.scrollTop;
      for (let i = 0; i < matchPositions.length; i++) {
        if (matchPositions[i].position > currentScrollTop) {
          nextIndex = i;
          break;
        }
      }
    }

    const nextMatch = matchPositions[nextIndex].element;
    if (nextMatch) {
      nextMatch.scrollIntoView({ behavior: "smooth", block: "center" });
      currentMatchIndex.value = nextIndex + 1;
    }
  };

  const goToPreviousMatch = () => {
    if (!query() || !fileContent()) return;
    const container = codeContainerRef();
    if (!container) return;

    const matchPositions = getAllMatchPositions();
    if (matchPositions.length === 0) return;

    let prevIndex = 0;

    if (currentMatchIndex.value > 1) {
      prevIndex = currentMatchIndex.value - 2;
    } else {
      prevIndex = matchPositions.length - 1;
    }

    const prevMatch = matchPositions[prevIndex].element;
    if (prevMatch) {
      prevMatch.scrollIntoView({ behavior: "smooth", block: "center" });
      currentMatchIndex.value = prevIndex + 1;
    }
  };

  watch(
    () => [fileContent(), query()],
    () => {
      currentMatchIndex.value = 0;
      if (observer.value) {
        observer.value.disconnect();
        observer.value = null;
      }
      visibleMatches.value.clear();
      matchElements.value = [];
    },
  );

  onUnmounted(() => {
    if (observer.value) {
      observer.value.disconnect();
      observer.value = null;
    }
    visibleMatches.value.clear();
    matchElements.value = [];
  });

  return {
    currentMatchIndex,
    observer,
    visibleMatches,
    matchElements,
    totalMatches,
    refreshMatchObserver,
    goToNextMatch,
    goToPreviousMatch,
  };
}