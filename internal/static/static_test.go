package static_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kube-adguard-ing/internal/static"
)

func TestStatic_Load(t *testing.T) {
	fileName := saveTextFile(t, `
- domain: hello-world
  address:
  - 1.2.3.4
  - 6.7.8.9
- domain: example
  address:
  - 9.8.7.6
`)
	defer os.Remove(fileName)

	s := static.New(static.Config{
		Path: fileName,
		TTL:  time.Minute,
	})
	list, err := s.Load()
	require.NoError(t, err)
	require.Len(t, list, 2)

	assert.Equal(t, "hello-world", list[0].Domain)
	assert.Equal(t, "example", list[1].Domain)

	assert.Equal(t, []string{"1.2.3.4", "6.7.8.9"}, list[0].Address)
	assert.Equal(t, []string{"9.8.7.6"}, list[1].Address)
}

func saveTextFile(t *testing.T, content string) string {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString(content)
	require.NoError(t, err)

	require.NoError(t, f.Close())

	return f.Name()
}
