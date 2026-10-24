package supervisor

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/ElodineOfficial/GobboNet/internal/models"
)

// writeHeaderOnlyGGUF writes a GGUF v3 file with the given metadata and no
// tensors. That is all ReadGGUFMeta looks at: it parses the header and never
// touches weights, so a few hundred bytes stand in for a 20 GB model.
func writeHeaderOnlyGGUF(t *testing.T, path string, strs map[string]string, u32s map[string]uint32) {
	t.Helper()
	const (
		ggufTypeUint32 = 4
		ggufTypeString = 8
	)
	var b bytes.Buffer
	le := binary.LittleEndian
	str := func(s string) {
		binary.Write(&b, le, uint64(len(s)))
		b.WriteString(s)
	}
	b.WriteString("GGUF")
	binary.Write(&b, le, uint32(3))                   // version
	binary.Write(&b, le, uint64(0))                   // tensor count
	binary.Write(&b, le, uint64(len(strs)+len(u32s))) // metadata count
	for k, v := range strs {
		str(k)
		binary.Write(&b, le, uint32(ggufTypeString))
		str(v)
	}
	for k, v := range u32s {
		str(k)
		binary.Write(&b, le, uint32(ggufTypeUint32))
		binary.Write(&b, le, v)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The chain start() runs -- IdentifyFile on the model on disk, then BuildArgs
// -- against real GGUF headers, so the expert count is shown to arrive from
// the file and not only from a hand-built Record (issue #65, PR #66).
func TestMoEHeadroomFromRealGGUFHeader(t *testing.T) {
	dir := t.TempDir()
	moe := filepath.Join(dir, "Qwen_Qwen3.6-35B-A3B-Q4_K_M.gguf")
	dense := filepath.Join(dir, "Qwen_Qwen3.5-9B-Q4_K_M.gguf")
	writeHeaderOnlyGGUF(t, moe,
		map[string]string{"general.architecture": "qwen3moe", "general.name": "test moe"},
		map[string]uint32{"qwen3moe.expert_count": 128, "qwen3moe.expert_used_count": 8, "qwen3moe.context_length": 40960})
	writeHeaderOnlyGGUF(t, dense,
		map[string]string{"general.architecture": "qwen3", "general.name": "test dense"},
		map[string]uint32{"qwen3.context_length": 40960})

	meta, err := models.ReadGGUFMeta(moe)
	if err != nil {
		t.Fatalf("the test GGUF did not parse: %v", err)
	}
	if meta.ExpertCount != 128 {
		t.Fatalf("ReadGGUFMeta ExpertCount = %d, want 128 from qwen3moe.expert_count", meta.ExpertCount)
	}

	s, err := New(Options{LLMURL: "http://127.0.0.1:1", GPUReserveMiB: 1024,
		Tuning: Tuning{CtxSize: 16384, GPULayers: -1, KVCacheType: "q8_0"}})
	if err != nil {
		t.Fatal(err)
	}
	fitTarget := func(path string) string {
		args := s.BuildArgs(models.IdentifyFile(path), path)
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--fit-target" {
				return args[i+1]
			}
		}
		return ""
	}
	if got := fitTarget(moe); got != "2048" {
		t.Errorf("MoE GGUF on disk got --fit-target %q, want 2048", got)
	}
	if got := fitTarget(dense); got != "1024" {
		t.Errorf("dense GGUF on disk got --fit-target %q, want the configured 1024", got)
	}
}
