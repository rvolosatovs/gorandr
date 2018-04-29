package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	conn, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Failed to connect to X: %s", err)
	}

	if err := randr.Init(conn); err != nil {
		log.Fatalf("Failed to init RandR: %s", err)
	}

	root := xproto.Setup(conn).DefaultScreen(conn).Root

	if err = randr.SelectInputChecked(conn, root, randr.NotifyMaskOutputChange).Check(); err != nil {
		log.Fatalf("Failed to check for output changes: %s", err)
	}

	res, err := randr.GetScreenResources(conn, root).Reply()
	if err != nil {
		log.Fatalf("Failed to get screen resources: %s", err)
	}

	type info struct {
		Output *randr.GetOutputInfoReply
		Mode   randr.ModeInfo
	}

	is := make([]info, 0, len(res.Outputs))
	for _, o := range res.Outputs {
		oi, err := randr.GetOutputInfo(conn, o, 0).Reply()
		if err != nil {
			log.Fatalf("Failed to get info of output %d: %s", o, err)
		}

		if oi.Connection != randr.ConnectionConnected {
			continue
		}

		if len(oi.Modes) == 0 {
			log.Fatalf("No modes found for output %s", oi.Name)
		}

		mID := oi.Modes[0]
		for _, m := range res.Modes {
			if m.Id != uint32(mID) {
				continue
			}
			is = append(is, info{
				Output: oi,
				Mode:   m,
			})
		}
	}

	sort.Slice(is, func(i, j int) bool {
		switch vpi, vpj := uint64(is[i].Mode.Width)*uint64(is[i].Mode.Height), uint64(is[j].Mode.Width)*uint64(is[j].Mode.Height); {
		case vpi > vpj:
			return true
		case vpi == vpj:
			return sort.StringsAreSorted([]string{string(is[i].Output.Name), string(is[j].Output.Name)})
		default:
			return false
		}
	})

	x := uint16(0)
	for _, i := range is {
		fmt.Printf("%s:%dx%d+%d+%d\n", i.Output.Name, i.Mode.Width, i.Mode.Height, x, 0)
		x += i.Mode.Width
	}
}
