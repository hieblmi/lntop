package views

import (
	"fmt"
	"regexp"

	"github.com/awesome-gocui/gocui"
	"github.com/edouardparis/lntop/ui/color"
	"github.com/edouardparis/lntop/ui/models"
)

const (
	HEADER = "myheader"
)

var versionReg = regexp.MustCompile(`(\d+\.)?(\d+\.)?(\*|\d+)`)

type Header struct {
	Info *models.Info
}

func (h *Header) Set(g *gocui.Gui, x0, y0, x1, y1 int) error {
	v, err := g.SetView(HEADER, x0, y0, x1, y0+2, 0)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}
	v.Frame = false

	version := h.Info.Version
	matches := versionReg.FindStringSubmatch(h.Info.Version)
	if len(matches) > 0 {
		version = matches[0]
	}

	chain := ""
	if len(h.Info.Chains) > 0 {
		chain = h.Info.Chains[0]
	}

	network := h.Info.Network
	if network == "" {
		network = "mainnet"
	}

	sync := color.Yellow()("[syncing]")
	if h.Info.Synced {
		sync = color.Green()("[synced]")
	}

	v.Clear()
	cyan := color.Cyan()
	_, _ = fmt.Fprintf(v, "%s %s %s %s %s %s %d %s %d\n",
		color.Cyan(color.Background)(h.Info.Alias),
		cyan("lnd-v"+version),
		chain, network,
		sync,
		cyan("height:"), h.Info.BlockHeight,
		cyan("peers:"), h.Info.NumPeers,
	)
	return nil
}

func NewHeader(info *models.Info) *Header {
	return &Header{Info: info}
}
