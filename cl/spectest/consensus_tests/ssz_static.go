package consensus_tests

import (
	"bytes"
	"io/fs"
	"testing"

	"github.com/tenderly/erigon/spectest"

	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/cltypes"
	"github.com/tenderly/erigon/cl/persistence/format/snapshot_format"
	"github.com/tenderly/erigon/cl/phase1/core/state"

	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/types/clonable"
	"github.com/tenderly/erigon/erigon-lib/types/ssz"

	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/cl/utils"
	"gopkg.in/yaml.v2"
)

type unmarshalerMarshalerHashable interface {
	ssz.EncodableSSZ
	ssz.HashableSSZ
	clonable.Clonable
}

type Root struct {
	Root string `yaml:"root"`
}

const rootsFile = "roots.yaml"
const serializedFile = "serialized.ssz_snappy"

func getSSZStaticConsensusTest[T unmarshalerMarshalerHashable](ref T) spectest.Handler {
	return spectest.HandlerFunc(func(t *testing.T, fsroot fs.FS, c spectest.TestCase) (err error) {
		rootBytes, err := fs.ReadFile(fsroot, rootsFile)
		require.NoError(t, err)
		root := Root{}
		err = yaml.Unmarshal(rootBytes, &root)
		require.NoError(t, err)
		expectedRoot := libcommon.HexToHash(root.Root)
		object := ref.Clone().(unmarshalerMarshalerHashable)
		_, isBeaconState := object.(*state.CachingBeaconState)

		snappyEncoded, err := fs.ReadFile(fsroot, serializedFile)
		require.NoError(t, err)
		encoded, err := utils.DecompressSnappy(snappyEncoded)
		require.NoError(t, err)

		if err := object.DecodeSSZ(encoded, int(c.Version())); err != nil && !isBeaconState {
			return err
		}
		haveRoot, err := object.HashSSZ()
		require.NoError(t, err)
		require.EqualValues(t, expectedRoot, haveRoot)
		// Cannot test it without a config.
		if isBeaconState {
			return nil
		}
		haveEncoded, err := object.EncodeSSZ(nil)
		require.NoError(t, err)
		require.EqualValues(t, haveEncoded, encoded)
		// Now let it do the encoding in snapshot format
		if blk, ok := object.(*cltypes.SignedBeaconBlock); ok {
			var b bytes.Buffer
			_, err := snapshot_format.WriteBlockForSnapshot(&b, blk, nil)
			require.NoError(t, err)
			var br snapshot_format.MockBlockReader
			if blk.Version() >= clparams.BellatrixVersion {
				br = snapshot_format.MockBlockReader{Block: blk.Block.Body.ExecutionPayload}

			}

			blk2, err := snapshot_format.ReadBlockFromSnapshot(&b, &br, &clparams.MainnetBeaconConfig)
			require.NoError(t, err)

			haveRoot, err := blk2.HashSSZ()
			require.NoError(t, err)
			require.EqualValues(t, expectedRoot, haveRoot)
		}

		return nil
	})
}
