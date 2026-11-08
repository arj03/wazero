//go:build wazero_nodwarf

package wasmdebug

// DWARFLines is empty under wazero_nodwarf, which drops debug/dwarf from the binary:
// stack traces carry no source lines, as for a module without DWARF sections.
type DWARFLines struct{}

// ParseDWARFLines returns nil, the value for a module without DWARF sections.
func ParseDWARFLines(abbrev, info, line, ranges, str []byte) *DWARFLines { return nil }

// Line returns nil.
func (d *DWARFLines) Line(instructionOffset uint64) []string { return nil }
