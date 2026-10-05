package livefind

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xurrent/go-xurrent/pkg/collisionhints"
)

// TestLive_Search_teamName hits the real API when XURRENT_TOKEN and XURRENT_ACCOUNT are set.
func TestLive_Search_teamName(t *testing.T) {
	if os.Getenv("XURRENT_TOKEN") == "" || os.Getenv("XURRENT_ACCOUNT") == "" {
		t.Skip("set XURRENT_TOKEN and XURRENT_ACCOUNT for live API test")
	}
	client, err := ClientFromEnv()
	require.NoError(t, err)
	ctx := context.Background()
	needle := "___mcp_livefind_unlikely_" + time.Now().Format("150405")
	res, err := Search(ctx, client, collisionhints.KindTeam, "name", needle, "")
	if err != nil && strings.Contains(err.Error(), "429") {
		t.Skip("rate limited (HTTP 429)")
	}
	require.NoError(t, err)
	require.NotNil(t, res)
	t.Logf("matches=%d notes=%v", len(res.Matches), res.Notes)
}
