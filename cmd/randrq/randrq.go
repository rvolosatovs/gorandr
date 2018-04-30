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

const DefaultTemplate = "{{.Name}}:{{.Width}}x{{.Height}}+{{.X}}+{{.Y}}"

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

		outs := make([]*gorandr.Output, 0, len(res.Outputs))
		for _, o := range res.Outputs {
			oi, err := randr.GetOutputInfo(conn, o, 0).Reply()
			if err != nil {
				return errors.Wrapf(err, "failed to get info of output %d", o)
				continue
			}

			if oi.Connection != randr.ConnectionConnected {
				continue
			}

			ci, err := randr.GetCrtcInfo(conn, oi.Crtc, 0).Reply()
			if err != nil {
				log.Printf("Failed to get crtc info of output %d: %s", o, err)
				continue
			}

			outs = append(outs, &gorandr.Output{
				Name:      string(oi.Name),
				MmWidth:   oi.MmWidth,
				MmHeight:  oi.MmHeight,
				X:         ci.X,
				Y:         ci.Y,
				Width:     ci.Width,
				Height:    ci.Height,
				Rotation:  ci.Rotation,
				Rotations: ci.Rotations,
				Length:    ci.Length,
				Area:      uint64(ci.Width) * uint64(ci.Height),
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
