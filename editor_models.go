package main

// EditorAvailability holds information about which editors are available on the system
type EditorAvailability struct {
	VSCode          bool `json:"vscode"`
	VSCodium        bool `json:"vscodium"`
	Sublime         bool `json:"sublime"`
	JetBrains       bool `json:"jetbrains"`
	Geany           bool `json:"geany"`
	Neovim          bool `json:"neovim"`
	Vim             bool `json:"vim"`
	Nano            bool `json:"nano"`
	Micro           bool `json:"micro"`
	Helix           bool `json:"helix"`
	GoLand          bool `json:"goland"`
	PyCharm         bool `json:"pycharm"`
	IntelliJ        bool `json:"intellij"`
	WebStorm        bool `json:"webstorm"`
	PhpStorm        bool `json:"phpstorm"`
	CLion           bool `json:"clion"`
	Rider           bool `json:"rider"`
	AndroidStudio   bool `json:"androidstudio"`
	SystemDefault   bool `json:"systemdefault"`
	Emacs           bool `json:"emacs"`
	Neovide         bool `json:"neovide"`
	CodeBlocks      bool `json:"codeblocks"`
	DevCpp          bool `json:"devcpp"`
	NotepadPlusPlus bool `json:"notepadplusplus"`
	VisualStudio    bool `json:"visualstudio"`
	Eclipse         bool `json:"eclipse"`
	NetBeans        bool `json:"netbeans"`
}
