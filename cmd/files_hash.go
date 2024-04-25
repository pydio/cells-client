package cmd

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common/hasher"
	"github.com/pydio/cells-client/v4/common/hasher/simd"
)

var (
	hashFilePath string
)

var hashFile = &cobra.Command{
	Use:   "hash",
	Short: "Compute Hash for a local file using the same algorithm as Cells Server",
	Long: `
DESCRIPTION

This command uses the same block-based algorithm as in the Cells server to verify a local file hash.
Output should be the same as the File Metadata > Internal Hash displayed on the web UX.

BlockHashing computes hashes for blocks of ` + humanize.Bytes(hasher.DefaultBlockSize) + ` using a specific hasher, then computes 
md5 of all these hashes joined together. Block-level hasing is done using the  standard golang md5 library. You can switch 
to SIMD implementation (it may be a bit faster) by exporting environment variable 'CELLS_ENABLE_SIMDMD5=true'. 

EXAMPLE

    $ ` + os.Args[0] + ` hash --file /path/to/file.ext

`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create a Reader on File & Use Hasher to compute hash
		if hashFilePath == "" {
			fmt.Println("Please provide a file to hash with --file or -f parameter")
			return
		}
		if st, er := os.Stat(hashFilePath); er != nil {
			fmt.Println("Cannot find file to hash: " + er.Error())
			return
		} else if st.IsDir() {
			fmt.Println("Please provide a file, not a folder!")
			return
		}
		fmt.Println("Starting hashing for file " + hashFilePath)
		t := time.Now()
		bH := hasher.NewBlockHash(simd.MD5(), hasher.DefaultBlockSize)
		file, e := os.Open(hashFilePath)
		if e != nil {
			fmt.Println("Cannot open file to hash: " + e.Error())
			return
		}
		defer file.Close()
		written, er := io.Copy(bH, file)
		if er != nil {
			fmt.Println("Could not copy file content to hash: " + e.Error())
			return
		}
		final := hex.EncodeToString(bH.Sum(nil))
		fmt.Printf("Final MD5 is '%s'.\nIt was computed in %s for %s.\n", final, time.Since(t), humanize.Bytes(uint64(written)))
	},
}

func init() {
	flags := hashFile.PersistentFlags()
	flags.StringVarP(&hashFilePath, "file", "f", "", "Path to file")
	RootCmd.AddCommand(hashFile)
}
