package main

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// ---------------------------------------------------------------------------
// .gitignore support
//
// When SearchRequest.RespectGitignore is true, collection honors the whole
// chain of .gitignore files from the search root down to each file's own
// directory, plus the root's .git/info/exclude. Pattern syntax stays entirely
// go-gitignore's job (negation "!", "**", anchoring, dir-only patterns);
// ignoreStack adds the cross-FILE rules a single matcher has no notion of:
// patterns resolve relative to the directory holding the .gitignore that
// declared them, a deeper file overrides a shallower one, a deeper "!"
// un-ignores what an ancestor ignored, and an ignored directory is pruned
// from the walk instead of descended.
//
// ponytail: only ignore files inside the search tree are read. A repository
// .gitignore ABOVE the search directory, the global core.excludesFile
// (~/.config/git/ignore), and a nested submodule's own .git/info/exclude are
// all skipped. Upgrade: walk up from the search root to the enclosing .git
// directory and prepend those levels to the stack, only if searching a
// subdirectory of a repo needs the repo's own rules.
// ---------------------------------------------------------------------------

// gitignoreFileName is the per-directory ignore file name. Declared here
// because gitignore.go owns ignore-file semantics; collection_index.go's
// fingerprint walk needs it to spot ignore files among the paths it hashes.
const gitignoreFileName = ".gitignore"

// ignoreLevel is one directory's compiled ignore rules.
//
// Patterns are matched against the path relative to the directory containing
// the .gitignore, not relative to the search root, so that directory travels
// with the matcher — as the separator-terminated prefix to strip, computed
// once here so per-file matching allocates nothing.
type ignoreLevel struct {
	prefix string // stripped from a walk path to get the pattern-relative path
	all    *gitignore.GitIgnore

	// positive holds the same rules with every leading "!" removed, so a
	// match here that `all` does not report means this file DID mention the
	// path and its last matching rule was a negation — an explicit
	// un-ignore, which must override an ancestor's ignore.
	//
	// Without it cross-file negation cannot work: a lone "!debug.log" in a
	// nested file leaves go-gitignore's MatchesPath false, indistinguishable
	// from "no rule mentions this path", because a negation there only flips
	// a match an earlier rule in the SAME file had already set. nil when the
	// file negates nothing, so the common case compiles once, not twice.
	positive *gitignore.GitIgnore
}

// ignoreStack resolves the effective gitignore verdict for a path from the
// chain of .gitignore files between the search root and the path's own
// directory.
//
// chains memoizes the resolved level list per directory, so each .gitignore
// is read and compiled exactly once per walk and a directory holding N files
// costs N map lookups rather than N reads. Lifetime is a single
// walkDirectoryTree call; filepath.WalkDir invokes its callback sequentially,
// so no lock is needed.
type ignoreStack struct {
	root       string // filepath.Clean of the search directory
	rootPrefix string // bounds the chainFor recursion to the search tree
	chains     map[string][]ignoreLevel
}

// newIgnoreStack prepares an empty stack rooted at the search directory.
// Nothing is read until a path is actually tested, so a walk that ends early
// (cancelled, or an empty tree) opens no ignore file at all.
func newIgnoreStack(directory string) *ignoreStack {
	root := filepath.Clean(directory)
	return &ignoreStack{
		root:       root,
		rootPrefix: dirPrefix(root),
		chains:     make(map[string][]ignoreLevel),
	}
}

// ignoresFile reports whether a walk path for a regular file is ignored.
func (s *ignoreStack) ignoresFile(path string) bool {
	return ignoredIn(s.chainFor(parentDir(path)), path)
}

// ignoresDir reports whether a walk path for a directory can be pruned.
//
// Git never descends into an ignored directory, so pruning is not only the
// saving (no ReadDir, no ignore-file reads for the whole subtree) but the
// correct semantics: a rule inside an excluded directory cannot re-include
// anything under it.
func (s *ignoreStack) ignoresDir(path string) bool {
	// The search root is never pruned — that would collect nothing. This is
	// also the only walk path that may carry a trailing separator (WalkDir
	// passes the root exactly as given and Joins every other entry), so
	// Clean here and plain slicing everywhere below.
	if filepath.Clean(path) == s.root {
		return false
	}

	// Only the PARENT's chain applies: a directory's own .gitignore cannot
	// ignore the directory itself, and not consulting it keeps a pruned
	// subtree's ignore file unread.
	chain := s.chainFor(parentDir(path))

	// Pruning is unsafe as soon as an applicable file negates anything.
	// "build/*" plus "!build/keep.txt" excludes the CONTENTS of build/ while
	// keeping one file, yet the regex go-gitignore builds for "build/*"
	// matches the bare "build/" too — pruning there would swallow the
	// re-included file. Per-file matching resolves that case exactly, so
	// give up the pruning and descend. Negation-free trees (the
	// node_modules/vendor case this exists for) still prune.
	for _, level := range chain {
		if level.positive != nil {
			return false
		}
	}

	// Tested in "build/" form: a dir-only pattern compiles to a regex that
	// requires the trailing separator, and patterns without one match that
	// form too. go-gitignore normalizes the OS separator before matching.
	return ignoredIn(chain, path+string(filepath.Separator))
}

// ignoredIn folds the chain shallowest level first, each level that has an
// opinion overriding the previous one. That is exactly git's rule — the
// applicable patterns ordered shallowest to deepest, last match wins — since
// a level reports ignore only when its own last matching rule was positive,
// and reports un-ignore only when that rule was a negation.
func ignoredIn(chain []ignoreLevel, target string) bool {
	verdict := false
	for _, level := range chain {
		rel := strings.TrimPrefix(target, level.prefix)
		if level.all.MatchesPath(rel) {
			verdict = true
			continue
		}
		if level.positive != nil && level.positive.MatchesPath(rel) {
			verdict = false
		}
	}
	return verdict
}

// chainFor returns the ignore levels applying inside dir, shallowest first,
// building and memoizing it on first use.
func (s *ignoreStack) chainFor(dir string) []ignoreLevel {
	if chain, ok := s.chains[dir]; ok {
		return chain
	}

	var chain []ignoreLevel
	// Recurse toward the search root so ancestors land first. Every walked
	// directory is the root or below it, so the recursion ends at the root;
	// the prefix test bounds a path from outside the tree to the search
	// scope, and parent != dir stops at the filesystem root either way.
	if parent := filepath.Dir(dir); parent != dir && dir != s.root && strings.HasPrefix(dir, s.rootPrefix) {
		chain = s.chainFor(parent)
	}

	if level, ok := s.loadLevel(dir); ok {
		// Only directories that carry rules allocate; every other directory
		// shares its parent's slice by reference.
		chain = append(append(make([]ignoreLevel, 0, len(chain)+1), chain...), level)
	}

	s.chains[dir] = chain
	return chain
}

// loadLevel reads and compiles dir's own ignore rules, reporting absent when
// it has none. One os.ReadFile per directory, never per file.
func (s *ignoreStack) loadLevel(dir string) (ignoreLevel, bool) {
	// .git/info/exclude belongs to the repository and git reads it once, not
	// once per directory, so only the search root contributes it.
	lines := loadIgnoreLines(dir, dir == s.root)
	if len(lines) == 0 {
		return ignoreLevel{}, false
	}
	return compileIgnoreLevel(dirPrefix(dir), lines), true
}

// compileIgnoreLevel builds the level's matchers from its raw rule lines.
func compileIgnoreLevel(prefix string, lines []string) ignoreLevel {
	level := ignoreLevel{
		prefix: prefix,
		all:    gitignore.CompileIgnoreLines(lines...),
	}

	negates := false
	for _, line := range lines {
		if _, negated := stripNegation(line); negated {
			negates = true
			break
		}
	}
	if !negates {
		return level
	}

	positives := make([]string, len(lines))
	for i, line := range lines {
		positives[i], _ = stripNegation(line)
	}
	level.positive = gitignore.CompileIgnoreLines(positives...)
	return level
}

// stripNegation removes a rule's leading "!" and reports whether it had one.
// It mirrors go-gitignore's own parsing step for step — comment first, then
// surrounding spaces, then the "!", then the single extra leading "#"/"!"
// that un-escapes "!#foo" and "!!foo" — so the positive form of a rule
// compiles to the same pattern the negated one did. Diverging here would make
// a rule vanish from `positive` (parsed as a comment) or stay negated in it.
func stripNegation(line string) (string, bool) {
	trimmed := strings.Trim(line, " ")
	if strings.HasPrefix(trimmed, "#") || !strings.HasPrefix(trimmed, "!") {
		return line, false
	}
	rest := trimmed[1:]
	if strings.HasPrefix(rest, "#") || strings.HasPrefix(rest, "!") {
		rest = rest[1:]
	}
	return rest, true
}

// parentDir returns path's parent directory without allocating: the walk
// paths WalkDir produces are already clean and carry no trailing separator,
// so slicing at the last separator is equivalent to filepath.Dir. Called once
// per walked entry, which is why it is not just filepath.Dir — that cleans,
// and cleaning allocates.
func parentDir(path string) string {
	i := strings.LastIndexByte(path, filepath.Separator)
	switch i {
	case -1:
		return "." // relative entry directly under a relative search root
	case 0:
		return path[:1] // filesystem root, e.g. "/etc" -> "/"
	default:
		return path[:i]
	}
}

// dirPrefix returns the string to strip from a walk path to make it relative
// to directory. WalkDir cleans "./a" to "a", so a "." root has no prefix to
// strip, and a root that already ends in a separator ("/") must not gain a
// second one.
func dirPrefix(directory string) string {
	switch {
	case directory == ".":
		return ""
	case strings.HasSuffix(directory, string(filepath.Separator)):
		return directory
	default:
		return directory + string(filepath.Separator)
	}
}

// loadIgnoreLines reads directory's ignore rules as raw lines: its own
// .gitignore, plus .git/info/exclude when includeInfoExclude is set. Missing
// files contribute nothing.
func loadIgnoreLines(directory string, includeInfoExclude bool) []string {
	var lines []string

	if bs, err := os.ReadFile(filepath.Join(directory, gitignoreFileName)); err == nil {
		lines = append(lines, splitIgnoreLines(bs)...)
	}

	if includeInfoExclude {
		if bs, err := os.ReadFile(filepath.Join(directory, ".git", "info", "exclude")); err == nil {
			lines = append(lines, splitIgnoreLines(bs)...)
		}
	}

	return lines
}

// splitIgnoreLines splits ignore-file content on "\n", trimming trailing "\r"
// so CRLF ignore files parse identically to LF ones.
func splitIgnoreLines(bs []byte) []string {
	raw := string(bs)
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

// loadGitignoreMatcher builds a single matcher from the directory's own
// .gitignore and .git/info/exclude. Missing files contribute no lines.
// Returns nil when no ignore source exists.
//
// Flat, single-directory form: correct only for paths matched relative to
// `directory` itself. Collection goes through ignoreStack instead, which adds
// nested files and cross-file precedence.
func loadGitignoreMatcher(directory string) *gitignore.GitIgnore {
	lines := loadIgnoreLines(directory, true)
	if len(lines) == 0 {
		return nil
	}
	return gitignore.CompileIgnoreLines(lines...)
}

// filterByGitignore drops files whose path relative to `directory` is matched
// by a single matcher loaded from that same directory. Returns the input
// slice if nothing is dropped (no allocation).
//
// Flat counterpart to ignoreStack, for a caller that already holds exactly
// one matcher and one directory.
func filterByGitignore(files []fileMeta, directory string, matcher *gitignore.GitIgnore) []fileMeta {
	if matcher == nil {
		return files
	}
	out := make([]fileMeta, 0, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(directory, f.absPath)
		if err != nil {
			continue // can't resolve — drop, same safe default as the walk
		}
		if matcher.MatchesPath(rel) {
			continue
		}
		out = append(out, f)
	}
	return out
}
