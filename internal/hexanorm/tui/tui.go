package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/analysis"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/domain"
	"github.com/modelcontextprotocol/go-sdk/examples/server/hexanorm/internal/hexanorm/graph"
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	normalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	violationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5555")).
			Bold(true)
)

type item struct {
	title, desc string
	node        *domain.Node
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type Model struct {
	graph    *graph.Graph
	analyzer *analysis.Analyzer

	lists    []list.Model
	focused  int
	viewport viewport.Model

	ready bool
	width int
	height int
}

func NewModel(g *graph.Graph, a *analysis.Analyzer) Model {
	// Initialize lists for each layer
	layers := []string{"Domain", "Application", "Infrastructure", "Interface"}
	lists := make([]list.Model, len(layers))

	nodes := g.GetAllNodes()
	
	// Group nodes
	grouped := make(map[string][]list.Item)
	for _, n := range nodes {
		layer := "Other"
		if l, ok := n.Metadata["layer"].(string); ok {
			layer = strings.Title(l)
		} else if n.Kind == domain.NodeKindRequirement {
			layer = "Domain" // Put reqs in Domain for now
		}

		// Normalize layer names to match our columns
		switch layer {
		case "Domain", "Application", "Infrastructure", "Interface":
			// ok
		default:
			// maybe put in Interface or a separate Misc?
			// For simplicity, let's just skip "Other" or put in Interface
			// Actually, let's map unknown to Interface for now to see them
			layer = "Interface"
		}

		grouped[layer] = append(grouped[layer], item{
			title: shortID(n.ID),
			desc:  string(n.Kind),
			node:  n,
		})
	}

	for i, l := range layers {
		items := grouped[l]
		// Sort items
		sort.Slice(items, func(a, b int) bool {
			return items[a].(item).title < items[b].(item).title
		})

		lists[i] = list.New(items, list.NewDefaultDelegate(), 0, 0)
		lists[i].Title = l
		lists[i].SetShowHelp(false)
	}

	return Model{
		graph:    g,
		analyzer: a,
		lists:    lists,
		focused:  0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			m.focused--
			if m.focused < 0 {
				m.focused = len(m.lists) - 1
			}
		case "right", "l":
			m.focused++
			if m.focused >= len(m.lists) {
				m.focused = 0
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height/3)
			m.viewport.YPosition = msg.Height - msg.Height/3
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height / 3
		}

		// Resize lists
		// 4 columns
		colWidth := msg.Width / 4
		listHeight := msg.Height - m.viewport.Height - 5 // padding

		for i := range m.lists {
			m.lists[i].SetSize(colWidth-2, listHeight)
		}
	}

	// Update focused list
	m.lists[m.focused], cmd = m.lists[m.focused].Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport content based on selection
	selectedItem := m.lists[m.focused].SelectedItem()
	if selectedItem != nil {
		it := selectedItem.(item)
		m.viewport.SetContent(m.renderDetails(it.node))
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render columns
	cols := make([]string, len(m.lists))
	for i, l := range m.lists {
		style := normalStyle
		if i == m.focused {
			style = focusedStyle
		}
		cols[i] = style.Render(l.View())
	}

	// Join columns
	board := lipgloss.JoinHorizontal(lipgloss.Left, cols...)

	// Render details
	details := detailStyle.Width(m.width - 4).Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left, board, details)
}

func (m Model) renderDetails(n *domain.Node) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ID: %s\n", n.ID))
	sb.WriteString(fmt.Sprintf("Kind: %s\n", n.Kind))
	if layer, ok := n.Metadata["layer"]; ok {
		sb.WriteString(fmt.Sprintf("Layer: %s\n", layer))
	}

	sb.WriteString("\nViolations:\n")
	violations := m.analyzer.FindViolations() // This scans all, inefficient but ok for now
	found := false
	for _, v := range violations {
		if v.File == n.ID { // Assuming File matches ID for code nodes
			sb.WriteString(violationStyle.Render(fmt.Sprintf("- [%s] %s", v.Severity, v.Message)) + "\n")
			found = true
		}
	}
	if !found {
		sb.WriteString("No violations found.\n")
	}

	sb.WriteString("\nOutgoing Edges:\n")
	edges := m.graph.GetEdgesFrom(n.ID)
	for _, e := range edges {
		sb.WriteString(fmt.Sprintf("-> %s (%s)\n", shortID(e.TargetID), e.Type))
	}

	sb.WriteString("\nIncoming Edges:\n")
	inEdges := m.graph.GetEdgesTo(n.ID)
	for _, e := range inEdges {
		sb.WriteString(fmt.Sprintf("<- %s (%s)\n", shortID(e.SourceID), e.Type))
	}

	return sb.String()
}

func shortID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return id
}
