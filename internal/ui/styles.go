package ui

import "github.com/charmbracelet/lipgloss"

const (
	artCols       = 24
	artRows       = 12
	listRows      = 10
	listWidth     = 43
	infoWidth     = 52
	barWidth      = 30
	wideLayoutMin = 130
	frameInset    = 8 // app border + padding
)

var (
	purple = lipgloss.Color("#9867C5")
	yellow = lipgloss.Color("#F1C40F")
	green  = lipgloss.Color("#2ECC71")
	gray   = lipgloss.Color("#444444")
	white  = lipgloss.Color("#EEEEEE")

	appStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 1).
			MarginBottom(1)

	playingStyle = lipgloss.NewStyle().Bold(true).Foreground(green)
	pausedStyle  = lipgloss.NewStyle().Bold(true).Foreground(gray)

	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Width(14)
	valueStyle = lipgloss.NewStyle().Foreground(yellow).Bold(true)

	listTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))

	barEmptyStyle  = lipgloss.NewStyle().Foreground(gray)
	barFullStyle   = lipgloss.NewStyle().Foreground(purple)
	barPausedStyle = lipgloss.NewStyle().Foreground(yellow)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			MarginTop(1)
)
