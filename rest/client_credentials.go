package rest

// "context"
// "encoding/json"
// "io/ioutil"
// "log"
// "os"
// "path/filepath"
// "runtime"
// "strconv"

// "github.com/go-openapi/strfmt"
// "github.com/shibukawa/configdir"

// "github.com/pydio/cells-client/common"
// cells_sdk "github.com/pydio/cells-sdk-go"
// "github.com/pydio/cells-sdk-go/client"
// "github.com/pydio/cells-sdk-go/transport"

// Keys to retrieve configuration via environment variables
const (
	KeyURL          = "CELLS_CLIENT_TARGET_URL"
	KeyClientKey    = "CELLS_CLIENT_TARGET_CLIENT_KEY"
	KeyClientSecret = "CELLS_CLIENT_TARGET_CLIENT_SECRET"
	KeyUser         = "CELLS_CLIENT_TARGET_USER"
	KeyPassword     = "CELLS_CLIENT_TARGET_PWD"
	KeySkipVerify   = "CELLS_CLIENT_TARGET_SKIP_VERIFY"
)
