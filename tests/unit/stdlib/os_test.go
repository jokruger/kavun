package stdlib

import (
	"os"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/tests/require"
)

func TestReadFile(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := os.CreateTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tf.Name()) }()

	_, err = tf.Write(content)
	require.NoError(t, err)
	_ = tf.Close()

	module(t, "os").call("read_file", tf.Name()).expect(alloc.NewBytesValue(content))
}

func TestReadFileArgs(t *testing.T) {
	module(t, "os").call("read_file").expectError()
}
func TestFileStatArgs(t *testing.T) {
	module(t, "os").call("stat").expectError()
}

func TestFileStatFile(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := os.CreateTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tf.Name()) }()

	_, err = tf.Write(content)
	require.NoError(t, err)
	_ = tf.Close()

	stat, err := os.Stat(tf.Name())
	if err != nil {
		t.Logf("could not get tmp file stat: %s", err)
		return
	}

	name := alloc.NewStringValue(stat.Name())
	mt := core.NewTimeValue(stat.ModTime())

	module(t, "os").call("stat", tf.Name()).expect(alloc.NewRecordValue(map[string]core.Value{
		"name":      name,
		"mtime":     mt,
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.False,
	}, true))
}

func TestFileStatDir(t *testing.T) {
	td, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(td) }()

	stat, err := os.Stat(td)
	require.NoError(t, err)

	name := alloc.NewStringValue(stat.Name())
	mt := core.NewTimeValue(stat.ModTime())

	module(t, "os").call("stat", td).expect(alloc.NewRecordValue(map[string]core.Value{
		"name":      name,
		"mtime":     mt,
		"size":      core.IntValue(stat.Size()),
		"mode":      core.IntValue(int64(stat.Mode())),
		"directory": core.True,
	}, true))
}

func TestOSExpandEnv(t *testing.T) {
	_ = os.Setenv("KAVUN", "FOO BAR")
	module(t, "os").call("expand_env", "$KAVUN").expect("FOO BAR")

	_ = os.Setenv("KAVUN", "FOO")
	module(t, "os").call("expand_env", "$KAVUN $KAVUN").expect("FOO FOO")

	_ = os.Setenv("KAVUN", "123456789012")
	module(t, "os").call("expand_env", "$KAVUN").expect("123456789012")

	_ = os.Setenv("KAVUN", "123456")
	module(t, "os").call("expand_env", "$KAVUN$KAVUN").expect("123456123456")

	_ = os.Setenv("KAVUN", "123456")
	module(t, "os").call("expand_env", "${KAVUN}${KAVUN}").expect("123456123456")
}
