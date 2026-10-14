package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/vm"
)

// -genkey writes one line of hex, owner-only, and never overwrites; -key (or $KAVUN_KEY_FILE) loads it.
func TestKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.key")
	require.NoError(t, GenerateKey(path))
	st, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, "-rw-------", st.Mode().Perm().String())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, 2*vm.MinHMACKeyLen+1, len(data))
	require.True(t, strings.HasSuffix(string(data), "\n"))

	err = GenerateKey(path)
	require.True(t, errors.Is(err, os.ErrExist), "%v", err)
	again, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(data), string(again))

	signer, err := loadSigner(path)
	require.NoError(t, err)
	require.True(t, signer != nil)

	t.Setenv(keyFileEnv, "")
	signer, err = loadSigner("")
	require.NoError(t, err)
	require.True(t, signer == nil)
	t.Setenv(keyFileEnv, path)
	signer, err = loadSigner("")
	require.NoError(t, err)
	require.True(t, signer != nil)

	for content, want := range map[string]string{
		"zz\n":   "not one line of hex",
		"abcd\n": "key too short: 2 bytes (min: 32)",
	} {
		bad := filepath.Join(dir, "bad.key")
		require.NoError(t, os.WriteFile(bad, []byte(content), 0o600))
		_, err := loadSigner(bad)
		require.True(t, err != nil && strings.Contains(err.Error(), want), "%v", err)
	}
	_, err = loadSigner(filepath.Join(dir, "missing.key"))
	require.Error(t, err)
}

// A file compiled with a key runs with that key and without one; an unsigned file, or one signed with another key,
// is refused when a key is given. -info reports instead of refusing.
func TestSignedCompile(t *testing.T) {
	dir := t.TempDir()
	keyPath, otherPath := filepath.Join(dir, "app.key"), filepath.Join(dir, "other.key")
	require.NoError(t, GenerateKey(keyPath))
	require.NoError(t, GenerateKey(otherPath))
	key, err := loadSigner(keyPath)
	require.NoError(t, err)
	other, err := loadSigner(otherPath)
	require.NoError(t, err)

	src := []byte(`x := import("math").hypot(3, 4)`)
	in := filepath.Join(dir, "app.kvn")
	signed, plain := filepath.Join(dir, "app.kvb"), filepath.Join(dir, "plain.kvb")
	require.NoError(t, os.WriteFile(signed, bytes.Repeat([]byte{'x'}, 4096), 0o600)) // longer than the output: truncated
	require.NoError(t, CompileOnly(src, in, signed, compiler.O0(), key))
	require.NoError(t, CompileOnly(src, in, plain, compiler.O0(), nil))
	signedData, err := os.ReadFile(signed)
	require.NoError(t, err)
	plainData, err := os.ReadFile(plain)
	require.NoError(t, err)

	require.NoError(t, RunCompiled(signedData, key))
	require.NoError(t, RunCompiled(signedData, nil))
	require.NoError(t, RunCompiled(plainData, nil))
	require.True(t, errors.Is(RunCompiled(plainData, key), vm.ErrUntrusted))
	require.True(t, errors.Is(RunCompiled(signedData, other), vm.ErrUntrusted))

	info := func(data []byte, signer vm.Signer) string {
		var out bytes.Buffer
		require.NoError(t, PrintInfo(&out, bytes.NewReader(data), signer))
		return out.String()
	}
	core := "core version:  1.0 (this engine: 1.0)\nhost version:  0.0 (this process: 0.0)\n"
	rest := "imports:       math\nbindings:      (none)\n"
	require.Equal(t, core+"signed:        yes (not checked: no key)\n"+rest, info(signedData, nil))
	require.Equal(t, core+"signed:        yes, signature valid\n"+rest, info(signedData, key))
	require.Equal(t, core+"signed:        yes, signature INVALID\n"+rest, info(signedData, other))
	require.Equal(t, core+"signed:        no\n"+rest, info(plainData, nil))
	require.Equal(t, core+"signed:        no (refused with this key)\n"+rest, info(plainData, key))

	err = PrintInfo(&bytes.Buffer{}, bytes.NewReader(src), nil)
	require.True(t, errors.Is(err, vm.ErrNotBytecode), "%v", err)
}
