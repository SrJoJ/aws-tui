package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/SrJoJ/aws-tui/pkg/awsclient"
	"github.com/SrJoJ/aws-tui/pkg/config"
	"github.com/SrJoJ/aws-tui/pkg/provider"
)

// AutocompleteInputField wraps tview.InputField to draw a dimmed inline suggestion.
type AutocompleteInputField struct {
	*tview.InputField
	suggestion string
}

// NewAutocompleteInputField initializes a new autocomplete-enabled input field.
func NewAutocompleteInputField() *AutocompleteInputField {
	return &AutocompleteInputField{
		InputField: tview.NewInputField(),
	}
}

// Draw overrides the default Draw method to display the inline autocomplete suggestion.
func (a *AutocompleteInputField) Draw(screen tcell.Screen) {
	a.InputField.Draw(screen)

	if a.suggestion == "" || !a.HasFocus() {
		return
	}

	x, y, width, _ := a.InputField.GetInnerRect()
	text := a.GetText()
	label := a.GetLabel()

	startX := x + len(label) + len(text)
	style := tcell.StyleDefault.Foreground(tcell.ColorGray)

	for i, r := range a.suggestion {
		if startX+i >= x+width {
			break
		}
		screen.SetContent(startX+i, y, r, nil, style)
	}
}

// App manages the main terminal interface and its lifecycle.
type App struct {
	app      *tview.Application
	registry *provider.Registry
	config   *config.Config

	// Layout grid and panels
	grid            *tview.Grid
	headerPanel     *tview.TextView
	mainPanel       *tview.Pages // Allows switching between Table, Describe, Log, and Profile views
	cheatSheetPanel *tview.TextView

	// Navigation & Command state
	cmdInput        *AutocompleteInputField
	activeProvider  provider.ResourceProvider
	activeView      string // "table", "describe", etc.
	selectedProfile string
	selectedRegion  string
	searchQuery     string

	// Table & Text components
	table        *tview.Table
	resources    []provider.Resource
	textView     *tview.TextView
	originalText string

	// Profile & Region components
	profileList *tview.List
	regionList  *tview.List
}

// NewApp initializes the TUI layout and sets up components.
func NewApp(registry *provider.Registry, cfg *config.Config) *App {
	tuiApp := &App{
		app:      tview.NewApplication(),
		registry: registry,
		config:   cfg,
		table:    tview.NewTable(),
	}

	tuiApp.setupUI()
	return tuiApp
}

// Run starts the main application loop.
func (a *App) Run() error {
	a.showProfileSelection()
	return a.app.SetRoot(a.grid, true).EnableMouse(true).Run()
}

func (a *App) getMainFocusable() tview.Primitive {
	frontPage, _ := a.mainPanel.GetFrontPage()
	switch frontPage {
	case "profile_select":
		return a.profileList
	case "region_select":
		return a.regionList
	case "text_view":
		return a.textView
	default:
		return a.table
	}
}

func (a *App) setupUI() {
	// 1. Command Input Bar
	a.cmdInput = NewAutocompleteInputField()
	a.cmdInput.InputField.
		SetPlaceholder("Type : for commands, / to search").
		SetPlaceholderTextColor(tcell.ColorGray).
		SetFieldTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorYellow).
		SetFieldBackgroundColor(tcell.ColorDefault)

	a.cmdInput.SetChangedFunc(func(text string) {
		a.cmdInput.suggestion = a.getAutocompleteSuggestion(text)
	})

	// 2. Header panel
	a.headerPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.updateHeader("None", "None")

	// 3. Main content pages
	a.mainPanel = tview.NewPages()
	a.setupTable()
	a.setupProfileSelection()
	a.setupRegionSelection()

	a.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorWhite).
		SetScrollable(true)

	a.mainPanel.AddPage("table", a.table, true, true)
	a.mainPanel.AddPage("profile_select", a.profileList, true, true)
	a.mainPanel.AddPage("region_select", a.regionList, true, true)
	a.mainPanel.AddPage("text_view", a.textView, true, true)

	// 4. Cheat Sheet panel
	a.cheatSheetPanel = tview.NewTextView().
		SetDynamicColors(true)
	a.updateCheatSheetForViews("profile_select")

	// 5. Organize overall layout using Grid (Header, Cheat Sheet, CmdInput, Main Content)
	a.grid = tview.NewGrid().
		SetRows(3, 1, 1, 0). // Header (3), Cheat Sheet (1), CmdInput (1), Main Content (0)
		SetColumns(0).
		SetBorders(false).
		AddItem(a.headerPanel, 0, 0, 1, 1, 0, 0, false).
		AddItem(a.cheatSheetPanel, 1, 0, 1, 1, 0, 0, false).
		AddItem(a.cmdInput, 2, 0, 1, 1, 0, 0, false).
		AddItem(a.mainPanel, 3, 0, 1, 1, 0, 0, true)

	// Global Key handlers
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Inside command input mode
		if a.app.GetFocus() == a.cmdInput {
			switch event.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				a.cmdInput.SetText("")
				a.cmdInput.SetLabel("")
				a.cmdInput.suggestion = ""
				frontPage, _ := a.mainPanel.GetFrontPage()
				if frontPage == "text_view" {
					a.highlightText("")
					a.app.SetFocus(a.textView)
				} else {
					if a.searchQuery != "" {
						a.cmdInput.SetLabel("/")
						a.cmdInput.SetText(a.searchQuery)
					}
					a.app.SetFocus(a.getMainFocusable())
				}
				return nil
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if a.cmdInput.GetText() == "" {
					a.cmdInput.SetLabel("")
					a.cmdInput.suggestion = ""
					a.app.SetFocus(a.getMainFocusable())
					return nil
				}
			case tcell.KeyTab:
				if a.cmdInput.suggestion != "" {
					a.cmdInput.SetText(a.cmdInput.GetText() + a.cmdInput.suggestion)
					a.cmdInput.suggestion = ""
				}
				return nil
			case tcell.KeyEnter:
				text := a.cmdInput.GetText()
				a.cmdInput.suggestion = ""
				label := a.cmdInput.GetLabel()
				a.cmdInput.SetText("")
				a.cmdInput.SetLabel("")

				if label == ":" {
					a.handleCommand(text)
				} else if label == "/" {
					frontPage, _ := a.mainPanel.GetFrontPage()
					if frontPage == "text_view" {
						a.searchQuery = text
						a.highlightText(text)
						a.app.SetFocus(a.textView)
					} else {
						a.searchQuery = text
						if a.searchQuery != "" {
							a.cmdInput.SetLabel("/")
							a.cmdInput.SetText(a.searchQuery)
						}
						a.refreshData()
						a.app.SetFocus(a.getMainFocusable())
					}
				} else {
					a.app.SetFocus(a.getMainFocusable())
				}
				return nil
			}
			return event
		}

		// Handle key capture on text_view directly
		if a.app.GetFocus() == a.textView {
			if event.Key() == tcell.KeyEscape {
				a.mainPanel.SwitchToPage("table")
				a.updateCheatSheetForViews("table")
				a.app.SetFocus(a.table)
				return nil
			}
			if event.Key() == tcell.KeyRune && event.Rune() == '/' {
				a.app.SetFocus(a.cmdInput)
				a.cmdInput.SetLabel("/")
				a.cmdInput.SetText("")
				return nil
			}
		}

		// Main navigation key handlers (when not focused on input)
		cmdPaletteShortcut := a.config.Shortcuts.Global["command_palette"]
		profileSelectShortcut := a.config.Shortcuts.Global["profile_select"]

		if a.matchShortcut(event, cmdPaletteShortcut) || (event.Key() == tcell.KeyRune && event.Rune() == ':') {
			a.app.SetFocus(a.cmdInput)
			a.cmdInput.SetLabel(":")
			a.cmdInput.SetText("")
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			a.app.SetFocus(a.cmdInput)
			a.cmdInput.SetLabel("/")
			a.cmdInput.SetText("")
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			a.app.Stop()
			return nil
		}
		if a.matchShortcut(event, profileSelectShortcut) {
			a.showProfileSelection()
			return nil
		}

		// Handle Describe view with 'd' key in table
		if a.app.GetFocus() == a.table && a.activeProvider != nil {
			if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
				row, _ := a.table.GetSelection()
				if row > 0 && row-1 < len(a.resources) {
					res := a.resources[row-1]
					go func() {
						desc, err := a.activeProvider.Describe(context.Background(), res.ID)
						a.app.QueueUpdateDraw(func() {
							if err != nil {
								a.cheatSheetPanel.SetText(fmt.Sprintf(" [red]Error describing resource: %v", err))
							} else {
								escaped := escapeTviewTags(desc)
								a.originalText = escaped
								a.textView.SetText(escaped)
								a.mainPanel.SwitchToPage("text_view")
								a.updateCheatSheetForViews("text_view")
								a.app.SetFocus(a.textView)
							}
						})
					}()
					return nil
				}
			}
		}

		// Handle custom provider-specific hotkeys if table view is active
		if a.app.GetFocus() == a.table && a.activeProvider != nil {
			providerName := strings.ToLower(a.activeProvider.GetResourceType())
			for _, action := range a.activeProvider.GetCustomActions() {
				hotkey := action.Hotkey
				// Check if overridden in config
				if providerShortcuts, ok := a.config.Shortcuts.Providers[providerName]; ok {
					if customHotkey, exists := providerShortcuts[action.Name]; exists {
						hotkey = customHotkey
					}
				}

				if a.matchShortcut(event, hotkey) {
					row, _ := a.table.GetSelection()
					if row > 0 && row-1 < len(a.resources) {
						res := a.resources[row-1]
						go func() {
							content, err := action.ActionFunc(context.Background(), res)
							a.app.QueueUpdateDraw(func() {
								if err != nil {
									a.cheatSheetPanel.SetText(fmt.Sprintf(" [red]Error executing action '%s': %v", action.Name, err))
								} else {
									if action.Type == "text" {
										escaped := escapeTviewTags(content)
										a.originalText = escaped
										a.textView.SetText(escaped)
										a.mainPanel.SwitchToPage("text_view")
										a.updateCheatSheetForViews("text_view")
										a.app.SetFocus(a.textView)
									} else {
										a.cheatSheetPanel.SetText(fmt.Sprintf(" [green]Executed action '%s' successfully", action.Name))
									}
								}
							})
						}()
					}
					return nil
				}
			}
		}

		return event
	})
}

func (a *App) setupTable() {
	a.table.SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
}

func (a *App) setupProfileSelection() {
	a.profileList = tview.NewList().
		ShowSecondaryText(false).
		SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(tcell.ColorYellow)
	a.profileList.SetBorder(true).
		SetTitle(" Select AWS Profile ").
		SetTitleColor(tcell.ColorYellow)

	profiles := awsclient.GetAWSProfiles()
	for i, profile := range profiles {
		var shortcut rune
		if i < 9 {
			shortcut = rune('1' + i)
		}
		a.profileList.AddItem(profile, "", shortcut, nil)
	}

	a.profileList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		a.selectedProfile = mainText
		a.selectedRegion = "us-east-1"

		// Set default view to EC2 if it exists
		if p, ok := a.registry.Get("ec2"); ok {
			a.switchProvider(p)
		} else {
			a.updateHeader(a.selectedProfile, a.selectedRegion)
			a.updateCheatSheetForViews("table")
			a.mainPanel.SwitchToPage("table")
		}

		a.app.SetFocus(a.table)
	})
}

func (a *App) setupRegionSelection() {
	a.regionList = tview.NewList().
		ShowSecondaryText(false).
		SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(tcell.ColorYellow)
	a.regionList.SetBorder(true).
		SetTitle(" Select AWS Region ").
		SetTitleColor(tcell.ColorYellow)

	regions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-central-1", "ap-northeast-1",
		"ap-southeast-1", "ap-southeast-2", "sa-east-1",
	}
	for i, region := range regions {
		var shortcut rune
		if i < 9 {
			shortcut = rune('1' + i)
		}
		a.regionList.AddItem(region, "", shortcut, nil)
	}

	a.regionList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		a.selectedRegion = mainText

		if p, ok := a.registry.Get("ec2"); ok {
			a.switchProvider(p)
		} else {
			a.updateHeader(a.selectedProfile, a.selectedRegion)
			a.updateCheatSheetForViews("table")
			a.mainPanel.SwitchToPage("table")
		}

		a.app.SetFocus(a.table)
	})
}

func (a *App) showProfileSelection() {
	a.activeProvider = nil
	a.updateHeader("None", "None")
	a.updateCheatSheetForViews("profile_select")
	a.mainPanel.SwitchToPage("profile_select")
	a.app.SetFocus(a.profileList)
}

func (a *App) showRegionSelection() {
	a.activeProvider = nil
	a.updateHeader(a.selectedProfile, "None")
	a.updateCheatSheetForViews("region_select")
	a.mainPanel.SwitchToPage("region_select")
	a.app.SetFocus(a.regionList)
}

func (a *App) updateHeader(profile, region string) {
	a.headerPanel.SetText(fmt.Sprintf(
		" [green]AWS-TUI[white] | Profile: [yellow]%s[white] | Region: [yellow]%s[white] | Active: [cyan]%s[white]",
		profile, region, a.getActiveResourceName(),
	))
}

func (a *App) updateCheatSheet() {
	var sb strings.Builder
	sb.WriteString(" [yellow]Global Keys:[white] :profile (Select Profile) | :region (Select Region) | / (Filter) | :q (Quit)")

	if a.activeProvider != nil {
		providerName := a.activeProvider.GetResourceType()
		sb.WriteString(fmt.Sprintf("  [cyan]%s Actions:[white] d (Describe) |", providerName))

		actions := a.activeProvider.GetCustomActions()
		for _, action := range actions {
			hotkey := action.Hotkey
			// Check for config overrides
			if providerShortcuts, ok := a.config.Shortcuts.Providers[strings.ToLower(providerName)]; ok {
				if customHotkey, exists := providerShortcuts[action.Name]; exists {
					hotkey = customHotkey
				}
			}
			sb.WriteString(fmt.Sprintf(" %s (%s) |", hotkey, action.Name))
		}
		text := sb.String()
		if strings.HasSuffix(text, "|") {
			text = text[:len(text)-1]
		}
		a.cheatSheetPanel.SetText(text)
		return
	}
	a.cheatSheetPanel.SetText(sb.String())
}

func (a *App) updateCheatSheetForViews(viewName string) {
	if viewName == "profile_select" {
		a.cheatSheetPanel.SetText(" [yellow]Profile Select:[white] Enter/1-9 (Select Profile) | / (Filter) | :q (Quit)")
	} else if viewName == "region_select" {
		a.cheatSheetPanel.SetText(" [yellow]Region Select:[white] Enter/1-9 (Select Region) | / (Filter) | :q (Quit)")
	} else if viewName == "text_view" {
		a.cheatSheetPanel.SetText(" [yellow]Text View:[white] / (Search) | Esc (Back to Table) | :q (Quit)")
	} else {
		a.updateCheatSheet()
	}
}

func (a *App) getActiveResourceName() string {
	if a.activeProvider == nil {
		return "None"
	}
	return a.activeProvider.GetResourceType()
}

func (a *App) handleCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	// Exit commands
	if cmd == "q" || cmd == "quit" {
		a.app.Stop()
		return
	}

	// Profile command
	if cmd == "profile" || cmd == "profiles" {
		a.showProfileSelection()
		return
	}

	// Region view commands
	if cmd == "reg" || cmd == "region" || cmd == "regions" {
		a.showRegionSelection()
		return
	}

	// Direct region switch commands (e.g. reg us-west-2)
	if strings.HasPrefix(cmd, "reg ") || strings.HasPrefix(cmd, "region ") {
		parts := strings.Fields(cmd)
		if len(parts) == 2 {
			a.selectedRegion = parts[1]
			if p, ok := a.registry.Get("ec2"); ok {
				a.switchProvider(p)
			} else {
				a.updateHeader(a.selectedProfile, a.selectedRegion)
				a.updateCheatSheetForViews("table")
				a.mainPanel.SwitchToPage("table")
			}
			a.app.SetFocus(a.table)
			return
		}
	}

	// Resource commands
	if p, found := a.registry.Get(cmd); found {
		a.switchProvider(p)
	} else {
		a.cheatSheetPanel.SetText(fmt.Sprintf(" [red]Unknown resource alias or command: %s", cmd))
	}
}

func (a *App) switchProvider(p provider.ResourceProvider) {
	a.activeProvider = p
	a.updateHeader(a.selectedProfile, a.selectedRegion)
	a.updateCheatSheetForViews("table")
	a.refreshData()
}

func (a *App) refreshData() {
	if a.activeProvider == nil {
		return
	}

	a.table.Clear()

	// 1. Draw Headers
	cols := a.activeProvider.GetColumns()
	for colIdx, colDef := range cols {
		cell := tview.NewTableCell(colDef.Header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false)
		a.table.SetCell(0, colIdx, cell)
	}

	// 2. Fetch Data from Provider
	go func() {
		filters := make(map[string]string)
		if a.searchQuery != "" {
			filters["search"] = a.searchQuery
		}

		resources, err := a.activeProvider.List(context.Background(), filters)
		if err != nil {
			a.app.QueueUpdateDraw(func() {
				cell := tview.NewTableCell(fmt.Sprintf("Error: %v", err)).
					SetTextColor(tcell.ColorRed)
				a.table.SetCell(1, 0, cell)
			})
			return
		}

		a.app.QueueUpdateDraw(func() {
			a.resources = resources
			for rowIdx, res := range resources {
				for colIdx, colDef := range cols {
					val := colDef.ValueFunc(res)
					cell := tview.NewTableCell(val).
						SetTextColor(tcell.ColorWhite)

					// Highlight running/status values
					if colDef.Header == "STATUS" {
						if val == "running" || val == "active" {
							cell.SetTextColor(tcell.ColorGreen)
						} else if val == "stopped" || val == "terminated" {
							cell.SetTextColor(tcell.ColorRed)
						}
					}

					a.table.SetCell(rowIdx+1, colIdx, cell)
				}
			}
		})
	}()
}

// getAutocompleteSuggestion returns the autocomplete suggestion based on input prefix.
func (a *App) getAutocompleteSuggestion(text string) string {
	if a.cmdInput.GetLabel() != ":" {
		return ""
	}

	// 1. Gather all built-in/registered commands
	commands := []string{
		"profile",
		"profiles",
		"reg",
		"region",
		"regions",
	}

	// Add all short names of registered providers
	for _, p := range a.registry.ListProviders() {
		commands = append(commands, strings.ToLower(p.GetResourceType()))
		commands = append(commands, p.GetShortNames()...)
	}

	// Sort commands by length to prioritize shorter matching aliases
	sort.Slice(commands, func(i, j int) bool {
		return len(commands[i]) < len(commands[j])
	})

	// 2. Handle region sub-command auto-completion (e.g. reg us-east-1)
	if strings.HasPrefix(text, "reg ") || strings.HasPrefix(text, "region ") {
		parts := strings.Fields(text)
		if len(parts) == 1 {
			return "us-east-1"
		}
		if len(parts) == 2 {
			arg := parts[1]
			regions := []string{
				"us-east-1", "us-east-2", "us-west-1", "us-west-2",
				"eu-west-1", "eu-central-1", "ap-northeast-1",
				"ap-southeast-1", "ap-southeast-2", "sa-east-1",
			}
			for _, r := range regions {
				if strings.HasPrefix(r, arg) && r != arg {
					return r[len(arg):]
				}
			}
		}
		return ""
	}

	// 3. Find the best match for the main command
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, text) && cmd != text {
			return cmd[len(text):]
		}
	}

	return ""
}

// matchShortcut matches tcell.EventKey with user-configured string shortcuts.
func (a *App) matchShortcut(event *tcell.EventKey, shortcutStr string) bool {
	shortcutStr = strings.ToLower(strings.TrimSpace(shortcutStr))
	if shortcutStr == "" {
		return false
	}

	// Handle special keys
	switch shortcutStr {
	case "enter":
		return event.Key() == tcell.KeyEnter
	case "esc":
		return event.Key() == tcell.KeyEscape
	case "backspace":
		return event.Key() == tcell.KeyBackspace
	case "tab":
		return event.Key() == tcell.KeyTab
	}

	// Handle ctrl-X shortcuts dynamically
	if strings.HasPrefix(shortcutStr, "ctrl-") && len(shortcutStr) == 6 {
		char := shortcutStr[5]
		if char >= 'a' && char <= 'z' {
			expectedKey := tcell.Key(char - 'a' + 1)
			return event.Key() == expectedKey
		}
	}

	// Handle normal runes
	if len(shortcutStr) == 1 && event.Key() == tcell.KeyRune {
		return string(event.Rune()) == shortcutStr
	}

	return false
}

// escapeTviewTags escapes open bracket characters so they are not parsed as dynamic color tags by tview.
func escapeTviewTags(text string) string {
	return strings.ReplaceAll(text, "[", "[[")
}

// highlightText highlights occurrences of the search query inside the text view.
func (a *App) highlightText(query string) {
	if query == "" {
		a.textView.SetText(a.originalText)
		return
	}

	escapedQuery := escapeTviewTags(query)
	lowerOrig := strings.ToLower(a.originalText)
	lowerQuery := strings.ToLower(escapedQuery)

	var sb strings.Builder
	lastIdx := 0

	for {
		idx := strings.Index(lowerOrig[lastIdx:], lowerQuery)
		if idx == -1 {
			sb.WriteString(a.originalText[lastIdx:])
			break
		}
		matchStart := lastIdx + idx
		matchEnd := matchStart + len(escapedQuery)

		sb.WriteString(a.originalText[lastIdx:matchStart])
		sb.WriteString(fmt.Sprintf("[yellow:black]%s[white:default]", a.originalText[matchStart:matchEnd]))

		lastIdx = matchEnd
	}

	a.textView.SetText(sb.String())
}
