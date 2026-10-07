// Package tui is the Bubble Tea interface: replay list, match detail,
// trends of several matches and launching the coaching sessions.
package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"replayer/internal/analysis"
	"replayer/internal/coach"
	"replayer/internal/replay"
	"replayer/internal/trends"
)

var (
	keyCoach  = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "coach"))
	keyDetail = key.NewBinding(key.WithKeys("enter"), key.WithHelp("entrée", "détail"))
	keyMark   = key.NewBinding(key.WithKeys(" "), key.WithHelp("espace", "cocher"))
	keyUnmark = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "tout décocher"))
	keyTrends = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tendances"))
	keyBack   = key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc", "retour"))
)

// Number of replays analysed in parallel (each one runs rrrocket).
const analysisWorkers = 4

type item struct {
	r       *replay.Replay
	players []string
	marked  bool
}

func (i item) ourTeam() int { return i.r.OurTeam(i.players) }

func (i item) scores() (us, them int) { return i.r.Scores(i.players) }

func (i item) Title() string {
	us, them := i.scores()
	res := "D"
	if us > them {
		res = "V"
	}
	mark := "  "
	if i.marked {
		mark = "● "
	}
	return fmt.Sprintf("%s%s  %s %d-%d  %s  %s %dv%d", mark, i.r.Date.Format("02/01/2006 15:04"), res, us, them,
		i.r.Map, i.r.MatchType, i.r.TeamSize, i.r.TeamSize)
}

func (i item) Description() string {
	coached := i.r.Coached(i.players)
	var mates, opps []string
	for _, p := range i.r.Players {
		switch {
		case p.Team != i.ourTeam():
			opps = append(opps, p.Name)
		case len(coached) == 0 || p.Name != coached[0]:
			mates = append(mates, p.Name)
		}
	}
	s := "contre " + strings.Join(opps, ", ")
	if len(mates) > 0 {
		s = "avec " + strings.Join(mates, ", ") + " · " + s
	}
	return "  " + s
}

func (i item) FilterValue() string {
	var names []string
	for _, p := range i.r.Players {
		names = append(names, p.Name)
	}
	return i.Title() + " " + strings.Join(names, " ")
}

type page int

const (
	pageList page = iota
	pageDetail
	pageTrends
)

// reportsMsg brings the analyses computed in the background; then runs on
// the model once they are stored.
type reportsMsg struct {
	reports map[string]*analysis.Report
	errs    map[string]error
	then    func(*Model) tea.Cmd
}

type coachDoneMsg struct{ err error }

type Model struct {
	list    list.Model
	vp      viewport.Model
	players []string
	page    page

	detail       *item
	trends       *trends.Trends
	trendReplays []*replay.Replay

	// Analysis cache, by replay ID.
	reports    map[string]*analysis.Report
	reportErrs map[string]error

	status    string
	statusErr bool
	width     int
}

func New(replays []*replay.Replay, players []string, warnings []error) Model {
	items := make([]list.Item, len(replays))
	for i, r := range replays {
		items[i] = item{r: r, players: players}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = fmt.Sprintf("Replays Rocket League (%d)", len(replays))
	l.AdditionalShortHelpKeys = func() []key.Binding { return []key.Binding{keyDetail, keyCoach, keyMark, keyTrends} }
	l.AdditionalFullHelpKeys = func() []key.Binding { return []key.Binding{keyDetail, keyCoach, keyMark, keyUnmark, keyTrends} }
	m := Model{list: l, vp: viewport.New(0, 0), players: players,
		reports: map[string]*analysis.Report{}, reportErrs: map[string]error{}}
	if len(warnings) > 0 {
		m.setStatus(fmt.Sprintf("%d replay(s) illisible(s), ex : %v", len(warnings), warnings[0]), true)
	}
	return m
}

func (m *Model) setStatus(s string, isErr bool) { m.status, m.statusErr = s, isErr }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.list.SetSize(msg.Width, msg.Height-1)
		m.vp.Width, m.vp.Height = msg.Width, msg.Height-2
		m.refresh()
		return m, nil
	case reportsMsg:
		for id, rep := range msg.reports {
			m.reports[id] = rep
		}
		for id, err := range msg.errs {
			m.reportErrs[id] = err
		}
		cmd := msg.then(&m)
		m.refresh()
		return m, cmd
	case coachDoneMsg:
		m.setStatus("", false)
		if msg.err != nil {
			m.setStatus("session coach terminée avec une erreur : "+msg.err.Error(), true)
		}
		return m, nil
	case tea.KeyMsg:
		if m.page != pageList {
			switch {
			case key.Matches(msg, keyBack):
				m.page = pageList
				return m, nil
			case key.Matches(msg, keyCoach):
				if m.page == pageDetail {
					return m, m.startCoach(*m.detail)
				}
				return m, m.startTrendsCoach()
			case msg.String() == "ctrl+c":
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
		if m.list.FilterState() != list.Filtering {
			if it, ok := m.list.SelectedItem().(item); ok {
				switch {
				case key.Matches(msg, keyDetail):
					m.page, m.detail = pageDetail, &it
					m.refresh()
					m.vp.GotoTop()
					return m, m.analyze([]*replay.Replay{it.r}, func(*Model) tea.Cmd { return nil })
				case key.Matches(msg, keyCoach):
					return m, m.startCoach(it)
				case key.Matches(msg, keyMark):
					it.marked = !it.marked
					return m, m.list.SetItem(m.list.GlobalIndex(), it)
				}
			}
			switch {
			case key.Matches(msg, keyUnmark):
				var cmds []tea.Cmd
				for i, li := range m.list.Items() {
					if it := li.(item); it.marked {
						it.marked = false
						cmds = append(cmds, m.list.SetItem(i, it))
					}
				}
				return m, tea.Batch(cmds...)
			case key.Matches(msg, keyTrends):
				return m, m.startTrends()
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// analyze computes the missing reports of rs in the background, then calls
// then on the model.
func (m *Model) analyze(rs []*replay.Replay, then func(*Model) tea.Cmd) tea.Cmd {
	var missing []*replay.Replay
	for _, r := range rs {
		if m.reports[r.ID] == nil && m.reportErrs[r.ID] == nil {
			missing = append(missing, r)
		}
	}
	return func() tea.Msg {
		msg := reportsMsg{reports: map[string]*analysis.Report{}, errs: map[string]error{}, then: then}
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, analysisWorkers)
		for _, r := range missing {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				rep, err := analysis.File(r)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					msg.errs[r.ID] = err
				} else {
					msg.reports[r.ID] = rep
				}
			}()
		}
		wg.Wait()
		return msg
	}
}

// startCoach analyses the replay if needed, then suspends the TUI and hands
// the terminal to an interactive claude session; the TUI comes back when
// claude exits.
func (m *Model) startCoach(it item) tea.Cmd {
	if !claudeAvailable(m) {
		return nil
	}
	m.setStatus("analyse du replay…", false)
	return m.analyze([]*replay.Replay{it.r}, func(m *Model) tea.Cmd {
		m.setStatus("", false)
		rep := m.reports[it.r.ID]
		if rep == nil {
			m.setStatus(fmt.Sprintf("analyse des frames impossible, coaching sur les stats de fin de match : %v", m.reportErrs[it.r.ID]), true)
		}
		cmd, err := coach.Command(it.r, rep, m.players)
		if err != nil {
			m.setStatus("préparation du coaching impossible : "+err.Error(), true)
			return nil
		}
		return tea.ExecProcess(cmd, func(err error) tea.Msg { return coachDoneMsg{err} })
	})
}

// startTrends analyses the marked replays and opens the trends page.
func (m *Model) startTrends() tea.Cmd {
	var rs []*replay.Replay
	for _, li := range m.list.Items() {
		if it := li.(item); it.marked {
			rs = append(rs, it.r)
		}
	}
	if len(rs) < 2 {
		m.setStatus("coche au moins 2 replays avec espace pour voir les tendances", true)
		return nil
	}
	m.setStatus(fmt.Sprintf("analyse de %d replays…", len(rs)), false)
	return m.analyze(rs, func(m *Model) tea.Cmd {
		var failed []string
		for _, r := range rs {
			if m.reports[r.ID] == nil {
				failed = append(failed, fmt.Sprintf("%s (%v)", r.Date.Format("02/01"), m.reportErrs[r.ID]))
			}
		}
		t := trends.Build(rs, m.reports, m.players)
		if len(t.Matches) < 2 {
			m.setStatus("pas assez de replays analysables : "+strings.Join(failed, ", "), true)
			return nil
		}
		m.setStatus("", false)
		if len(failed) > 0 {
			m.setStatus("ignorés (analyse impossible) : "+strings.Join(failed, ", "), true)
		}
		m.trends, m.trendReplays, m.page = t, rs, pageTrends
		m.refresh()
		m.vp.GotoTop()
		return nil
	})
}

func (m *Model) startTrendsCoach() tea.Cmd {
	if !claudeAvailable(m) {
		return nil
	}
	cmd, err := coach.TrendsCommand(m.trends, m.trendReplays, m.reports)
	if err != nil {
		m.setStatus("préparation du coaching impossible : "+err.Error(), true)
		return nil
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return coachDoneMsg{err} })
}

func claudeAvailable(m *Model) bool {
	if _, err := exec.LookPath("claude"); err != nil {
		m.setStatus("commande claude introuvable dans le PATH", true)
		return false
	}
	return true
}

// refresh re-renders the content of the scrollable page.
func (m *Model) refresh() {
	switch m.page {
	case pageDetail:
		m.vp.SetContent(m.detailContent(*m.detail))
	case pageTrends:
		m.vp.SetContent(m.trendsContent())
	}
}

func (m Model) View() string {
	var v string
	switch m.page {
	case pageList:
		v = m.list.View()
	default:
		v = m.vp.View() + "\n" + dimStyle.Render("  ↑/↓ défiler · c coach · esc retour · ctrl+c quitter")
	}
	if m.status != "" {
		style := infoStyle
		if m.statusErr {
			style = errStyle
		}
		v += "\n" + style.Render(m.status)
	}
	return v
}
