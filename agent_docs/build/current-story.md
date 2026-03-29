# Current Story: M2-S1

## Bubbletea app scaffold + tree data model

**Acceptance Criteria:**
- [ ] Bubbletea app initializes and renders to terminal
- [ ] Tree node model: symbol name, kind icon, file path, line number, stale flag, children (lazy-loadable)
- [ ] Root node loaded from graph store query
- [ ] `q` quits the app cleanly
- [ ] App handles terminal resize gracefully

**Relevant System Design:**
- TUI framework: charmbracelet/bubbletea
- Styling: charmbracelet/lipgloss
- Components: charmbracelet/bubbles
- Package: internal/tui/ (app.go, tree.go, preview.go, keymap.go, styles.go)
- Tree node shows: symbol kind prefix (fn/class/type/iface/var/exp), file path, line number
- Key bindings: q quits

**Dependencies needed:**
- github.com/charmbracelet/bubbletea
- github.com/charmbracelet/lipgloss
- github.com/charmbracelet/bubbles
