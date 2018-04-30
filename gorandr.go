package gorandr

import (
	"hash"
	"hash/fnv"
	"sort"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/pkg/errors"
)

// Output represents the X output.
type Output struct {
	output randr.Output
	crtc   randr.Crtc
	mode   randr.Mode

	Name      string `json:"name"`
	MmWidth   uint32 `json:"mm_width"`
	MmHeight  uint32 `json:"mm_height"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Width     uint16 `json:"width"`
	Height    uint16 `json:"height"`
	Rotation  uint16 `json:"rotation"`
	Rotations uint16 `json:"rotations"`
	Length    uint32 `json:"length"`
	Area      uint64 `json:"area,omitempty"`
}

// ActiveOutputs returns the currently connected outputs.
func ActiveOutputs(conn *xgb.Conn, root xproto.Window) ([]*Output, error) {
	res, err := randr.GetScreenResources(conn, root).Reply()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get screen resources")
	}

	outs := make([]*Output, 0, len(res.Outputs))
	for _, o := range res.Outputs {
		oi, err := randr.GetOutputInfo(conn, o, 0).Reply()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get info of output %d", o)
		}

		if oi.Connection != randr.ConnectionConnected {
			continue
		}

		ci, err := randr.GetCrtcInfo(conn, oi.Crtc, 0).Reply()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get crtc info of output %d", o)
		}

		outs = append(outs, &Output{
			output: o,
			crtc:   oi.Crtc,
			mode:   ci.Mode,

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
	sort.SliceStable(outs, func(i, j int) bool {
		return outs[i].Name < outs[j].Name
	})
	return outs, nil
}

// Fingerprint identifies a connected output <-> EDID mapping.
// It assumes that randr.Init(conn) has been called and succeeded.
// If h is nil, 128-bit FNV-1a is used for hashing.
func Fingerprint(conn *xgb.Conn, root xproto.Window, outs []*Output, h hash.Hash) ([]byte, error) {
	type pair struct {
		name string
		EDID []byte
	}

	pairs := make([]pair, 0, len(outs))
	for _, o := range outs {
		props, err := randr.ListOutputProperties(conn, o.output).Reply()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get properties of output %s", o.Name)
		}

		for _, a := range props.Atoms {
			an, err := xproto.GetAtomName(conn, a).Reply()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get name of atom %d of output %s", a, o.Name)
			}

			if an.Name == "EDID" {
				p, err := randr.GetOutputProperty(conn, o.output, a, xproto.AtomAny, 0, 100, false, false).Reply()
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get property %d of output %s", a, o.Name)
				}

				pairs = append(pairs, pair{
					name: o.Name,
					EDID: p.Data,
				})
			}
		}
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].name < pairs[j].name
	})

	if h == nil {
		h = fnv.New128a()
	}

	b := make([]byte, 0, len(pairs)*32)
	for _, p := range pairs {
		b = append(append(b, h.Sum([]byte(p.name))...), h.Sum(p.EDID)[:]...)
	}
	return h.Sum(b), nil
}
