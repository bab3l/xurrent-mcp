package livefind

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapStr(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{"id": float64(42), "name": "x"}
	require.Equal(t, "42", mapStr(m, "id"))
	require.Equal(t, "x", mapStr(m, "name"))
	require.Equal(t, "", mapStr(m, "missing"))
}

func TestMatchFold(t *testing.T) {
	t.Parallel()
	require.True(t, matchFold(" Foo ", "foo"))
	require.False(t, matchFold("a", "b"))
}

func TestNextSearchAfter_nilEmpty(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", nextSearchAfter(nil))
	require.Equal(t, "", nextSearchAfter(&http.Response{Header: http.Header{}}))
}
