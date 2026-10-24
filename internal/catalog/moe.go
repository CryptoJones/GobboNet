package catalog

// borrowMoE gives each entry in c that carries no Mixture-of-Experts allowance
// the allowance of the entry in bundled with the same file, and reports how
// many it filled (issue #65).
//
// Same reasoning as borrowBuckets: the live catalogue is published separately
// and does not carry moe_min_vram_gb / min_ram_gb yet, so without this a newer
// build would describe the MoE models by their whole-model VRAM only while the
// bundled models.ini beside it knows better. A live entry that states either
// figure keeps its own -- the live list wins wherever it has an opinion.
func borrowMoE(c, bundled *Catalog) int {
	if c == nil || bundled == nil {
		return 0
	}
	type allowance struct{ vram, ram int }
	byFile := map[string]allowance{}
	for _, e := range bundled.Entries {
		if e.MoEMinVRAM > 0 && e.MinRAM > 0 {
			byFile[e.File] = allowance{e.MoEMinVRAM, e.MinRAM}
		}
	}
	filled := 0
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.MoEMinVRAM > 0 || e.MinRAM > 0 {
			continue
		}
		if a, ok := byFile[e.File]; ok {
			e.MoEMinVRAM, e.MinRAM = a.vram, a.ram
			filled++
		}
	}
	return filled
}
