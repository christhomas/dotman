package diffview

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	FgReset      = "39"
	FgLight      = "37"
	BgReset      = "49"
	FgLightGrey  = "38;5;252"
	BgLightGrey  = "48;5;250"
	BgLightGreyName = "grey"
	BgDarkRed    = "48;5;52"
	BgDarkGreen  = "48;5;22"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleLen(s string) int {
	return len(ansiRe.ReplaceAllString(s, ""))
}

// Colors holds ANSI fragments used when serializing styles.
type Colors struct {
	BackgroundReset string
	ForegroundReset string
	DiffLeftBg      string
	DiffRightBg     string
	NeutralBg       string
	NeutralFg       string
	AccentFg        string
}

// Theme holds all visual configuration for rendering.
type Theme struct {
	Colors             Colors
	Border             bool
	BorderColor        tcell.Color
	BorderPadding      int
	PanelGap           int
	MinTotalWidth      int
	BorderHeight       int
	MinPanelHeight     int
	DiffLeftTagFormat  string
	DiffRightTagFormat string
	UnchangedTagFormat string
	LeftTitle          string
	RightTitle         string
}

var defaultTheme = Theme{
	Colors: Colors{
		BackgroundReset: BgReset,
		ForegroundReset: FgReset,
		DiffLeftBg:      BgDarkRed,
		DiffRightBg:     BgDarkGreen,
		NeutralBg:       BgLightGrey,
		NeutralFg:       FgLightGrey,
		AccentFg:        FgLight,
	},
	BorderPadding:      2,  // border adds 2 columns (left/right)
	PanelGap:           0,  // gap between panes
	MinTotalWidth:      20, // minimal total width to avoid too-narrow layout
	BorderHeight:       2,  // top + bottom
	MinPanelHeight:     3,  // ensure a usable panel even when empty
	Border:             true,
	BorderColor:        tcell.ColorWhite,
	DiffLeftTagFormat:  "[white:red]%s[-]",
	DiffRightTagFormat: "[white:green]%s[-]",
	UnchangedTagFormat: "[white:" + BgLightGreyName + "]%s[-]",
	LeftTitle:          "repo",
	RightTitle:         "dotfiles",
}

// FilePair represents a left/right file to render.
type FilePair struct {
	Label     string
	LeftPath  string
	RightPath string
}

// Renderer renders side-by-side panels to plain text with ANSI styling.
type Renderer struct {
	Theme Theme
}

// NewRenderer returns a Renderer with default colors.
func NewRenderer() *Renderer {
	return &Renderer{
		Theme: defaultTheme,
	}
}

func (r *Renderer) readOrMsg(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("<<unreadable or missing>>\n%s", err.Error())
	}
	return string(content)
}

func (r *Renderer) ansiForStyle(s tcell.Style) string {
	fg, bg, _ := s.Decompose()
	if fg == tcell.ColorDefault && bg == tcell.ColorDefault {
		return ""
	}

	toCode := func(c tcell.Color, isBg bool) string {
		if c == tcell.ColorDefault {
			if isBg {
				return r.Theme.Colors.BackgroundReset
			}
			return r.Theme.Colors.ForegroundReset
		}
		switch c {
		case tcell.ColorRed:
			if isBg {
				return r.Theme.Colors.DiffLeftBg
			}
			return r.Theme.Colors.AccentFg
		case tcell.ColorGreen:
			if isBg {
				return r.Theme.Colors.DiffRightBg
			}
			return r.Theme.Colors.AccentFg
		case tcell.ColorWhite:
			if isBg {
				return r.Theme.Colors.NeutralBg
			}
			return r.Theme.Colors.NeutralFg
		default:
			if isBg {
				return r.Theme.Colors.BackgroundReset
			}
			return r.Theme.Colors.ForegroundReset
		}
	}

	fgCode := toCode(fg, false)
	bgCode := toCode(bg, true)
	if fgCode == "" && bgCode == "" {
		return ""
	}
	return fmt.Sprintf("\x1b[%s;%sm", fgCode, bgCode)
}

// renderPanel returns the rendered ANSI string for a single pair of texts.
func (r *Renderer) renderPanel(label, leftText, rightText string, idx, total int) (string, error) {
	leftLines := strings.Split(leftText, "\n")
	rightLines := strings.Split(rightText, "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	height := maxLines + r.Theme.BorderHeight
	if height < r.Theme.MinPanelHeight {
		height = r.Theme.MinPanelHeight
	}

	// Compute dynamic width based on content and titles to avoid wrapping/truncation.
	leftTitle := fmt.Sprintf(" %s | %s (%d/%d) ", r.Theme.LeftTitle, label, idx, total)
	rightTitle := fmt.Sprintf(" %s | %s (%d/%d) ", r.Theme.RightTitle, label, idx, total)
	leftMaxLen := len(leftTitle)
	for _, l := range leftLines {
		if len(l) > leftMaxLen {
			leftMaxLen = len(l)
		}
	}
	rightMaxLen := len(rightTitle)
	for _, r := range rightLines {
		if len(r) > rightMaxLen {
			rightMaxLen = len(r)
		}
	}
	// Borders add padding on each pane plus a gap between panes.
	totalWidth := (leftMaxLen + r.Theme.BorderPadding) + (rightMaxLen + r.Theme.BorderPadding) + r.Theme.PanelGap
	if totalWidth < r.Theme.MinTotalWidth {
		totalWidth = r.Theme.MinTotalWidth
	}

	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		return "", err
	}
	screen.SetSize(totalWidth, height)
	screen.Clear()

	left := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetText(leftText)
	left.SetBorder(r.Theme.Border).
		SetBorderColor(r.Theme.BorderColor).
		SetTitle(fmt.Sprintf(" %s | %s (%d/%d) ", r.Theme.LeftTitle, label, idx, total))

	right := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetText(rightText)
	right.SetBorder(r.Theme.Border).
		SetBorderColor(r.Theme.BorderColor).
		SetTitle(fmt.Sprintf(" %s | %s (%d/%d) ", r.Theme.RightTitle, label, idx, total))

	layout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(left, 0, 1, false).
		AddItem(right, 0, 1, false)

	width, height := screen.Size()
	layout.SetRect(0, 0, width, height)
	layout.Draw(screen)
	screen.Show()

	// Capture just the bounding box of drawn content to avoid leading/trailing blank rows/cols.
	minX, maxX := width, 0
	minY, maxY := height, 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, _, _, _ := screen.GetContent(x, y)
			if r == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if minX > maxX { // nothing drawn
		screen.Fini()
		return "", nil
	}

	var lines []string
	for y := minY; y <= maxY; y++ {
		var row strings.Builder
		var curStyle tcell.Style
		hasStyle := false
		for x := minX; x <= maxX; x++ {
			rn, _, style, _ := screen.GetContent(x, y)
			if rn == 0 {
				rn = ' '
			}
			if !hasStyle || style != curStyle {
				row.WriteString(r.ansiForStyle(style))
				curStyle = style
				hasStyle = true
			}
			row.WriteRune(rn)
		}
		if hasStyle {
			row.WriteString("\x1b[0m")
		}
		lines = append(lines, strings.TrimRight(row.String(), " "))
	}
	screen.Fini()
	return strings.Join(lines, "\n"), nil
}

// RenderFiles renders each file pair and returns the ANSI strings.
// If highlightDiffLines is true, differing lines are given red/green backgrounds.
func (r *Renderer) RenderFiles(pairs []FilePair, highlightDiffLines bool) ([]string, error) {
	var results []string
	for i, pair := range pairs {
		leftRaw := strings.ReplaceAll(r.readOrMsg(pair.LeftPath), "\r\n", "\n")
		rightRaw := strings.ReplaceAll(r.readOrMsg(pair.RightPath), "\r\n", "\n")

		leftLines := strings.Split(leftRaw, "\n")
		rightLines := strings.Split(rightRaw, "\n")
		maxLines := len(leftLines)
		if len(rightLines) > maxLines {
			maxLines = len(rightLines)
		}

		if highlightDiffLines {
			styledLeft := make([]string, maxLines)
			styledRight := make([]string, maxLines)
			for idx := 0; idx < maxLines; idx++ {
				var l, rline string
				if idx < len(leftLines) {
					l = leftLines[idx]
				}
				if idx < len(rightLines) {
					rline = rightLines[idx]
				}
				if l != rline {
					styledLeft[idx] = fmt.Sprintf(r.Theme.DiffLeftTagFormat, l)
					styledRight[idx] = fmt.Sprintf(r.Theme.DiffRightTagFormat, rline)
				} else {
					styledLeft[idx] = fmt.Sprintf(r.Theme.UnchangedTagFormat, l)
					styledRight[idx] = fmt.Sprintf(r.Theme.UnchangedTagFormat, rline)
				}
			}
			leftRaw = strings.Join(styledLeft, "\n")
			rightRaw = strings.Join(styledRight, "\n")
		}

		panel, err := r.renderPanel(pair.Label, leftRaw, rightRaw, i+1, len(pairs))
		if err != nil {
			return nil, err
		}
		results = append(results, panel)
	}
	return results, nil
}
