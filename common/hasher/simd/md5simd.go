package simd

import (
	"crypto/md5"
	"hash"
	"os"
	"sync"

	md5simd "github.com/minio/md5-simd"

	"github.com/pydio/cells-client/v4/common"
)

var (
	mServer     md5simd.Server
	mServerOnce sync.Once
)

func MD5() hash.Hash {
	if os.Getenv(common.EnvPrefix+"_ENABLE_SIMDMD5") != "" {
		mServerOnce.Do(func() {
			mServer = md5simd.NewServer()
		})
		return mServer.NewHash()
	}
	return md5.New()
}
