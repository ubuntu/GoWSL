package wsl

import "errors"

type ImportType int8

const (
	Vhd ImportType = iota
	TarGz
)

// Export copies the distro's filesystem to a given file. This copy can
// later be imported.
func (d Distro) Export(where string, format ImportType) error {
	switch format {
	case Vhd:
		return export(d.Name, where, true)
	case TarGz:
		return export(d.Name, where, false)
	}
	return errors.New("unrecognized format. Use the exported values")
}

type importOpts struct {
	source      string
	destination string
	format      ImportType
	inPlace     bool
	wslVersion  int
}

// Import registers a new distro with the provided settings. Use these functions to
// provide the import settings:
// - FromTarball
// - FromVhd
// - InPlace
func (d *Distro) Import(o importOpts) error {
	switch {
	case o.inPlace:
		return importInPlace(d.Name, o.destination)
	case o.format == Vhd:
		return importCopy(d.Name, o.destination, o.source, true, o.wslVersion)
	case o.format == TarGz:
		return importCopy(d.Name, o.destination, o.source, false, o.wslVersion)
	}
	return errors.New("unrecognized importOpts parameter combination. Use the exported functions to fill them")
}

// FromVhd generates options for (*Distro).Import to import a distro copying a Vhd.
// Source is the location of the virtual harddrive, and destination is where the
// distro's filesystem vhd will live. wslVersion must be 1 or 2.
func FromVhd(source string, destination string, wslVersion int) importOpts {
	return importOpts{
		source:      source,
		destination: destination,
		format:      Vhd,
		inPlace:     false,
		wslVersion:  wslVersion,
	}
}

// FromTarball generates options for (*Distro).Import to import a distro copying a tarball.
// Source is the location of the tarball, and destination is where the distro's filesystem vhd
// will live. wslVersion must be 1 or 2.
func FromTarball(source string, destination string, wslVersion int) importOpts {
	return importOpts{
		source:      source,
		destination: destination,
		format:      TarGz,
		inPlace:     false,
		wslVersion:  wslVersion,
	}
}

// InPlace generates options for (*Distro).Import to import a distro using an existing Vhd.
// vhd is the virtual harddrive to use.
func InPlace(vhd string) importOpts {
	return importOpts{
		source:      vhd,
		destination: vhd,
		format:      Vhd,
		inPlace:     true,
		wslVersion:  2,
	}
}
