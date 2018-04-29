package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
)

func PrintOptimalLayout(conn *xgb.Conn, root xproto.Window) error {
	res, err := randr.GetScreenResources(conn, root).Reply()
	if err != nil {
		log.Fatalf("Failed to get screen resources: %s", err)
	}

	type info struct {
		Output *randr.GetOutputInfoReply
		Crtc   *randr.GetCrtcInfoReply
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

		ci, err := randr.GetCrtcInfo(conn, oi.Crtc, 0).Reply()
		if err != nil {
			log.Fatalf("Failed to get info of crtc %d associated with %s: %s", oi.Crtc, oi.Name, err)
		}
		is = append(is, info{
			Output: oi,
			Crtc:   ci,
		})
	}

	sort.Slice(is, func(i, j int) bool {
		switch vpi, vpj := uint64(is[i].Crtc.Width)*uint64(is[i].Crtc.Height), uint64(is[j].Crtc.Width)*uint64(is[j].Crtc.Height); {
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
		fmt.Printf("%s:%dx%d+%d+%d\n", i.Output.Name, i.Crtc.Width, i.Crtc.Height, x, i.Crtc.Y)
		x += i.Crtc.Width
	}
	return nil
}

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

	PrintOptimalLayout(conn, root)
	for {
		ev, err := conn.WaitForEvent()
		if err != nil {
			log.Fatalf("Failed to wait for output change event: %s", err)
		}

		switch ev := ev.(type) {
		case randr.NotifyEvent:
			if err := PrintOptimalLayout(conn, root); err != nil {
				log.Fatalf("Failed to rearrange screens: %s", err)
			}
		default:
			log.Fatalf("Unhandled event of type %T", ev)
		}
	}
}
