package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"text/template"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/pkg/errors"
	"github.com/rvolosatovs/gorandr"
)

const DefaultTemplate = "{{.Name}}:{{.Width}}x{{.Height}}"

var (
	printJSON = flag.Bool("json", false, "Print data as JSON")
	tmpl      = flag.String("f", DefaultTemplate, "Go template to use (has no effect for JSON)")
)

func main() {
	if err := func() error {
		flag.Parse()

		conn, err := xgb.NewConn()
		if err != nil {
			return errors.Wrap(err, "failed to connect to X")
		}

		if err := randr.Init(conn); err != nil {
			return errors.Wrap(err, "failed to init RandR")
		}

		root := xproto.Setup(conn).DefaultScreen(conn).Root

		res, err := randr.GetScreenResources(conn, root).Reply()
		if err != nil {
			return errors.Wrap(err, "failed to get screen resources")
		}

		modes := make(map[randr.Mode]randr.ModeInfo, len(res.Modes))
		for _, mi := range res.Modes {
			modes[randr.Mode(mi.Id)] = mi
		}

		outs := make([]*gorandr.Output, 0, len(res.Outputs))
		for _, o := range res.Outputs {
			oi, err := randr.GetOutputInfo(conn, o, 0).Reply()
			if err != nil {
				return errors.Wrapf(err, "failed to get info of output %s", oi.Name)
				continue
			}

			if oi.Connection != randr.ConnectionConnected {
				continue
			}

			if len(oi.Modes) < 1 {
				log.Printf("Output %s has no modes defined", oi.Name)
				continue
			}

			mi := modes[oi.Modes[0]]
			for _, m := range oi.Modes {
				info := modes[m]
				if uint64(info.Width)*uint64(info.Height) > uint64(mi.Width)*uint64(mi.Height) {
					mi = info
					continue
				}
			}

			outs = append(outs, &gorandr.Output{
				Name:     string(oi.Name),
				MmWidth:  oi.MmWidth,
				MmHeight: oi.MmHeight,
				Length:   oi.Length,
				Width:    mi.Width,
				Height:   mi.Height,
				Area:     uint64(mi.Width) * uint64(mi.Height),
			})
		}

		sort.Slice(outs, func(i, j int) bool {
			switch {
			case outs[i].Area > outs[j].Area:
				return true
			case outs[i].Area == outs[j].Area:
				return sort.StringsAreSorted([]string{outs[i].Name, outs[j].Name})
			default:
				return false
			}
		})

		if *printJSON {
			if err := json.NewEncoder(os.Stdout).Encode(outs); err != nil {
				return errors.Wrap(err, "failed to write JSON to stdout")
			}
			return nil
		}

		t, err := template.New("format").Parse(*tmpl)
		if err != nil {
			return errors.Wrap(err, "failed to parse template")
		}

		for _, out := range outs {
			if err := t.Execute(os.Stdout, out); err != nil {
				return errors.Wrap(err, "failed to execute template")
			}
			fmt.Print("\n")
		}
		return nil
	}(); err != nil {
		log.Fatal(err)
	}
}
