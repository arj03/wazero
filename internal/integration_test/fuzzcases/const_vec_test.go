package fuzzcases

import (
	"math/rand"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/testing/binaryencoding"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// TestConstVecLowering checks the compiler against the interpreter on vector shifts by a
// constant amount and on i8x16.shuffle, the lowerings that take an immediate or a single
// PSHUFB when the operands allow it.
func TestConstVecLowering(t *testing.T) {
	if !platform.CompilerSupported() {
		return
	}
	leb := func(b []byte, v uint32) []byte {
		for v >= 0x80 {
			b = append(b, byte(v)|0x80)
			v >>= 7
		}
		return append(b, byte(v))
	}
	sleb := func(b []byte, v int32) []byte {
		for {
			c := byte(v & 0x7f)
			v >>= 7
			if (v == 0 && c&0x40 == 0) || (v == -1 && c&0x40 != 0) {
				return append(b, c)
			}
			b = append(b, c|0x80)
		}
	}
	simd := func(b []byte, op uint32) []byte { return leb(append(b, wasm.OpcodeVecPrefix), op) }

	// Each body runs over v = (p0, p1) in local 4 and w = (p2, p3) in local 5, and leaves
	// one v128 that is returned as two i64s.
	var bodies [][]byte
	shifts := []uint32{107, 108, 109, 139, 140, 141, 171, 172, 173, 203, 204, 205} // shl, shr_s, shr_u per lane width
	amounts := []int32{0, 1, 2, 3, 4, 5, 7, 8, 9, 12, 15, 16, 17, 24, 25, 31, 32, 33, 63, 64, 65, -1}
	for _, op := range shifts {
		for _, amt := range amounts {
			b := []byte{wasm.OpcodeLocalGet, 4, wasm.OpcodeI32Const}
			bodies = append(bodies, simd(sleb(b, amt), op))
		}
	}
	rng := rand.New(rand.NewSource(1))
	lanes := func(f func(i int) byte) []byte {
		l := make([]byte, 16)
		for i := range l {
			l[i] = f(i)
		}
		return l
	}
	shuffles := []struct {
		y     byte
		lanes []byte
	}{
		{4, lanes(func(i int) byte { return byte(i&^3 | (i+2)&3) })},      // rotate each i32 by 16, x == y
		{4, lanes(func(i int) byte { return byte(i&^3|(i+3)&3) + 16 })},   // rotate by 8, x == y, lanes from y
		{5, lanes(func(i int) byte { return byte(15 - i) })},              // all from x
		{5, lanes(func(i int) byte { return byte(31 - i) })},              // all from y
		{5, lanes(func(i int) byte { return byte(rng.Intn(32)) })},        // both
		{4, lanes(func(i int) byte { return byte(rng.Intn(32)) })},        // both, x == y
		{5, lanes(func(i int) byte { return byte(i * 2) })},               // both, interleaved
		{5, lanes(func(i int) byte { return byte(rng.Intn(16) + 16) })},   // all from y, random
	}
	for _, s := range shuffles {
		b := simd([]byte{wasm.OpcodeLocalGet, 4, wasm.OpcodeLocalGet, s.y}, 13)
		bodies = append(bodies, append(b, s.lanes...))
	}

	i64 := wasm.ValueTypeI64
	m := &wasm.Module{
		TypeSection: []wasm.FunctionType{{Params: []wasm.ValueType{i64, i64, i64, i64}, Results: []wasm.ValueType{i64, i64}}},
	}
	for i, op := range bodies {
		var b []byte
		for _, l := range [][3]byte{{0, 1, 4}, {2, 3, 5}} {
			b = simd(append(b, wasm.OpcodeLocalGet, l[0]), 18) // i64x2.splat
			b = append(simd(append(b, wasm.OpcodeLocalGet, l[1]), 30), 1, wasm.OpcodeLocalSet, l[2])
		}
		b = append(append(b, op...), wasm.OpcodeLocalSet, 4)
		for lane := byte(0); lane < 2; lane++ {
			b = append(simd(append(b, wasm.OpcodeLocalGet, 4), 29), lane) // i64x2.extract_lane
		}
		m.FunctionSection = append(m.FunctionSection, 0)
		m.CodeSection = append(m.CodeSection, wasm.Code{
			LocalTypes: []wasm.ValueType{wasm.ValueTypeV128, wasm.ValueTypeV128},
			Body:       append(b, wasm.OpcodeEnd),
		})
		m.ExportSection = append(m.ExportSection, wasm.Export{Type: wasm.ExternTypeFunc, Name: string(rune('a' + i)), Index: wasm.Index(i)})
	}
	bin := binaryencoding.EncodeModule(m)

	interp := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	defer interp.Close(ctx)
	comp := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	defer comp.Close(ctx)
	want, err := interp.Instantiate(ctx, bin)
	require.NoError(t, err)
	got, err := comp.Instantiate(ctx, bin)
	require.NoError(t, err)

	for i := range bodies {
		name := string(rune('a' + i))
		for n := 0; n < 64; n++ {
			args := []uint64{rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()}
			w, err := want.ExportedFunction(name).Call(ctx, args...)
			require.NoError(t, err)
			g, err := got.ExportedFunction(name).Call(ctx, args...)
			require.NoError(t, err)
			require.Equal(t, w, g, "body %d (%x) on %x", i, bodies[i], args)
		}
	}
}
