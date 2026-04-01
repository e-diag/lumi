package bot

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReferralUserID(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	got := ParseReferralUserID("/start ref_" + id.String())
	require.NotNil(t, got)
	assert.Equal(t, id, *got)

	assert.Nil(t, ParseReferralUserID("/start"))
	assert.Nil(t, ParseReferralUserID("/start ref_not-a-uuid"))
	assert.Nil(t, ParseReferralUserID("/start invite_123"))
}

func TestBuildReferralLink(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	assert.Equal(t, "https://t.me/MyBot?start=ref_"+id.String(), BuildReferralLink("MyBot", id))
	assert.Equal(t, "https://t.me/MyBot?start=ref_"+id.String(), BuildReferralLink("@MyBot", id))
	assert.Equal(t, "", BuildReferralLink("", id))
}
