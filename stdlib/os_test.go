package stdlib_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/require"
)

func TestReadFile(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := ioutil.TempFile("", "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tf.Name()) }()

	_, err = tf.Write(content)
	require.NoError(t, err)
	_ = tf.Close()

	module(t, "os").call("read_file", tf.Name()).
		expect(&gs.Bytes{Value: content})
}

func TestReadFileArgs(t *testing.T) {
	module(t, "os").call("read_file").expectError()
}
func TestFileStatArgs(t *testing.T) {
	module(t, "os").call("stat").expectError()
}

func TestFileStatFile(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	tf, err := ioutil.TempFile("", "test")
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

	module(t, "os").call("stat", tf.Name()).expect(&gs.ImmutableMap{
		Value: map[string]gs.Object{
			"name":      &gs.String{Value: stat.Name()},
			"mtime":     &gs.Time{Value: stat.ModTime()},
			"size":      &gs.Int{Value: stat.Size()},
			"mode":      &gs.Int{Value: int64(stat.Mode())},
			"directory": gs.FalseValue,
		},
	})
}

func TestFileStatDir(t *testing.T) {
	td, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(td) }()

	stat, err := os.Stat(td)
	require.NoError(t, err)

	module(t, "os").call("stat", td).expect(&gs.ImmutableMap{
		Value: map[string]gs.Object{
			"name":      &gs.String{Value: stat.Name()},
			"mtime":     &gs.Time{Value: stat.ModTime()},
			"size":      &gs.Int{Value: stat.Size()},
			"mode":      &gs.Int{Value: int64(stat.Mode())},
			"directory": gs.TrueValue,
		},
	})
}

func TestOSExpandEnv(t *testing.T) {
	curMaxStringLen := gs.MaxStringLen
	defer func() { gs.MaxStringLen = curMaxStringLen }()
	gs.MaxStringLen = 12

	_ = os.Setenv("GS", "FOO BAR")
	module(t, "os").call("expand_env", "$GS").expect("FOO BAR")

	_ = os.Setenv("GS", "FOO")
	module(t, "os").call("expand_env", "$GS $GS").expect("FOO FOO")

	_ = os.Setenv("GS", "123456789012")
	module(t, "os").call("expand_env", "$GS").expect("123456789012")

	_ = os.Setenv("GS", "1234567890123")
	module(t, "os").call("expand_env", "$GS").expectError()

	_ = os.Setenv("GS", "123456")
	module(t, "os").call("expand_env", "$GS$GS").expect("123456123456")

	_ = os.Setenv("GS", "123456")
	module(t, "os").call("expand_env", "${GS}${GS}").
		expect("123456123456")

	_ = os.Setenv("GS", "123456")
	module(t, "os").call("expand_env", "$GS $GS").expectError()

	_ = os.Setenv("GS", "123456")
	module(t, "os").call("expand_env", "${GS} ${GS}").expectError()
}
