package ocrcommon_test

import (
	"crypto/rand"
	"testing"

	cryptop2p "github.com/libp2p/go-libp2p-core/crypto"
	p2ppeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/pgtest"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocrcommon"
)

func Test_DiscovererDatabase(t *testing.T) {
	db := pgtest.NewSqlDB(t)

	localPeerID1 := mustRandomP2PPeerID(t)
	localPeerID2 := mustRandomP2PPeerID(t)

	dd1 := ocrcommon.NewDiscovererDatabase(db, localPeerID1)
	dd2 := ocrcommon.NewDiscovererDatabase(db, localPeerID2)

	ctx := testutils.Context(t)

	t.Run("StoreAnnouncement writes a value", func(t *testing.T) {
		ann := []byte{1, 2, 3}
		err := dd1.StoreAnnouncement(ctx, "remote1", ann)
		assert.NoError(t, err)

		// test upsert
		ann = []byte{4, 5, 6}
		err = dd1.StoreAnnouncement(ctx, "remote1", ann)
		assert.NoError(t, err)

		// write a different value
		ann = []byte{7, 8, 9}
		err = dd1.StoreAnnouncement(ctx, "remote2", ann)
		assert.NoError(t, err)
	})

	t.Run("ReadAnnouncements reads values filtered by given peerIDs", func(t *testing.T) {
		announcements, err := dd1.ReadAnnouncements(ctx, []string{"remote1", "remote2"})
		require.NoError(t, err)

		assert.Len(t, announcements, 2)
		assert.Equal(t, []byte{4, 5, 6}, announcements["remote1"])
		assert.Equal(t, []byte{7, 8, 9}, announcements["remote2"])

		announcements, err = dd1.ReadAnnouncements(ctx, []string{"remote1"})
		require.NoError(t, err)

		assert.Len(t, announcements, 1)
		assert.Equal(t, []byte{4, 5, 6}, announcements["remote1"])
	})

	t.Run("is scoped to local peer ID", func(t *testing.T) {
		ann := []byte{10, 11, 12}
		err := dd2.StoreAnnouncement(ctx, "remote1", ann)
		assert.NoError(t, err)

		announcements, err := dd2.ReadAnnouncements(ctx, []string{"remote1"})
		require.NoError(t, err)
		assert.Len(t, announcements, 1)
		assert.Equal(t, []byte{10, 11, 12}, announcements["remote1"])

		announcements, err = dd1.ReadAnnouncements(ctx, []string{"remote1"})
		require.NoError(t, err)
		assert.Len(t, announcements, 1)
		assert.Equal(t, []byte{4, 5, 6}, announcements["remote1"])
	})

	t.Run("persists data across restarts", func(t *testing.T) {
		dd3 := ocrcommon.NewDiscovererDatabase(db, localPeerID1)

		announcements, err := dd3.ReadAnnouncements(ctx, []string{"remote1"})
		require.NoError(t, err)
		assert.Len(t, announcements, 1)
		assert.Equal(t, []byte{4, 5, 6}, announcements["remote1"])

	})
}

func mustRandomP2PPeerID(t *testing.T) p2ppeer.ID {
	p2pPrivkey, _, err := cryptop2p.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	id, err := p2ppeer.IDFromPrivateKey(p2pPrivkey)
	require.NoError(t, err)
	return id
}
