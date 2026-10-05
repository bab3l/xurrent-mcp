package livefind

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextSearchAfter_variants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		link string
		want string
	}{
		{
			"standard",
			`<https://api.xurrent.com/v1/teams?per_page=100&search_after=abc123>; rel="next"`,
			"abc123",
		},
		{
			"rel_unquoted",
			`<https://api.xurrent.com/v1/teams?per_page=100&search_after=xyz>; rel=next`,
			"xyz",
		},
		{
			"relative_path",
			`</v1/teams?per_page=100&search_after=rel1>; rel="next"`,
			"rel1",
		},
		{
			"encoded_cursor",
			`<https://api.xurrent.com/v1/teams?search_after=eyJpZCI6MX0%3D>; rel="next"`,
			"eyJpZCI6MX0=",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := &http.Response{Header: http.Header{"Link": []string{tc.link}}}
			require.Equal(t, tc.want, nextSearchAfter(resp))
		})
	}
}

func TestLinkHeader_multipleLines(t *testing.T) {
	t.Parallel()
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Add("Link", `<https://a.com/v1?search_after=first>; rel="first"`)
	resp.Header.Add("Link", `<https://a.com/v1?search_after=second>; rel="next"`)
	require.Equal(t, "second", nextSearchAfter(resp))
}

func TestListPageLimit_env(t *testing.T) {
	t.Setenv(envMaxPages, "")
	require.Equal(t, defaultListPageLimit, listPageLimit())

	t.Setenv(envMaxPages, "not-a-number")
	require.Equal(t, defaultListPageLimit, listPageLimit())

	t.Setenv(envMaxPages, "3")
	require.Equal(t, 3, listPageLimit())

	t.Setenv(envMaxPages, "99999")
	require.Equal(t, hardMaxListPages, listPageLimit())
}
