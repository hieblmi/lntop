package color

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

type Color = lipgloss.Color

var (
	yellow     = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))
	yellowBold = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true))
	green      = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("2")))
	greenBold  = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true))
	greenBg    = styleFunc(lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")))
	magentaBg  = styleFunc(lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("0")))
	red        = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))
	redBold    = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true))
	cyan       = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
	cyanBold   = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true))
	cyanBg     = styleFunc(lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0")))
	white      = styleFunc(lipgloss.NewStyle())
	whiteBold  = styleFunc(lipgloss.NewStyle().Bold(true))
	blackBg    = styleFunc(lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("7")))
	black      = styleFunc(lipgloss.NewStyle().Foreground(lipgloss.Color("0")))
)

func styleFunc(s lipgloss.Style) func(args ...interface{}) string {
	return func(args ...interface{}) string {
		return s.Render(fmt.Sprint(args...))
	}
}

type Option func(*options)

type options struct {
	bold bool
	bg   bool
}

func newOptions(opts []Option) options {
	options := options{}
	for i := range opts {
		if opts[i] == nil {
			continue
		}
		opts[i](&options)
	}
	return options
}

func Bold(o *options)       { o.bold = true }
func Background(o *options) { o.bg = true }

func Yellow(opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	if options.bold {
		return yellowBold
	}
	return yellow
}

func Green(opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	if options.bold {
		return greenBold
	}
	if options.bg {
		return greenBg
	}
	return green
}

func Red(opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	if options.bold {
		return redBold
	}
	return red
}

func White(opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	if options.bold {
		return whiteBold
	}
	return white
}

func Cyan(opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	if options.bold {
		return cyanBold
	}
	if options.bg {
		return cyanBg
	}
	return cyan
}

func Black(opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	if options.bg {
		return blackBg
	}
	return black
}

func Magenta(opts ...Option) func(a ...interface{}) string {
	return magentaBg
}

func HSL256(h, s, l float64, opts ...Option) func(a ...interface{}) string {
	options := newOptions(opts)
	// h is expected in [0,1], colorful.Hsl expects [0,360].
	c := colorful.Hsl(h*360, s, l)
	hex := c.Hex()

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
	if options.bg {
		fg := "#ffffff"
		if l > 0.5 {
			fg = "#000000"
		}
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color(fg)).
			Background(lipgloss.Color(hex))
	}
	if options.bold {
		style = style.Bold(true)
	}
	return func(a ...interface{}) string {
		return style.Render(fmt.Sprint(a...))
	}
}
