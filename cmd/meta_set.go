package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client"
	"github.com/pydio/cells-sdk-go/v4/client/meta_service"
	"github.com/pydio/cells-sdk-go/v4/client/user_meta_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var (
	metaSetNodePath     string
	metaSetOperation    string
	metaSetNamespace    string
	metaSetJsonValue    string
	metaSetStringValue  string
	metaSetNumericValue int
	metaSetBooleanValue bool
)

const (
	emptyJson string = "\"\""
)

var metaSet = &cobra.Command{
	Use:   "set",
	Short: "Set specific metadata for node",
	Long: `
DESCRIPTION	

	Update or Delete metadata for given node.

EXAMPLE

# Update usermeta-tag-validation-status meta of node:

$` + os.Args[0] + ` meta set --path=personal/admin/test.txt --operation=update --meta-name=usermeta-tag-validation-status --string-value=Validated

`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := sdkClient.GetApiClient()

		if err := validateFilePath(); err != nil {
			cmd.PrintErr(err)
			return
		}

		node, err := validateFileExist(ctx)
		if err != nil {
			cmd.PrintErr(err)
			return
		}

		if err := validateMetaNamespace(); err != nil {
			cmd.PrintErr(err)
			return
		}

		ns, err := getMetaNameSpace(ctx)
		if err != nil {
			cmd.PrintErr(err)
			return
		}
		nsDef, err := getNamespaceDefinition(ns)
		if err != nil {
			cmd.PrintErr(err)
			return
		}

		uv, err := formatInputData(cmd, nsDef.Type)

		if err != nil {
			cmd.PrintErr(err)
			return
		}

		v := getFinalJsonValue(nsDef.Type, uv)
	
		do(ctx, client, node, v)
	},
}

func init() {
	flags := metaSet.PersistentFlags()
	flags.StringVarP(&metaSetNodePath, "path", "p", "", "Absolute path of node")
	flags.StringVarP(&metaSetOperation, "operation", "o", "", "Operation name: update|delete")
	flags.StringVarP(&metaSetNamespace, "namespace", "n", "", "Metadata namespace")
	flags.StringVarP(&metaSetJsonValue, "json-value", "j", "", "JSON-formated metadata value")
	flags.StringVarP(&metaSetStringValue, "string-value", "s", "", "String-formated metadata value")
	flags.IntVarP(&metaSetNumericValue, "numeric-value", "r", 0, "String-formated metadata value")
	flags.BoolVarP(&metaSetBooleanValue, "boolean-value", "b", false, "String-formated metadata value")
	metaCmd.AddCommand(metaSet)
}

type metaNsDef struct {
	Type  string
	Data  interface{} `json:"data,omitempty"`
	Steps bool        `json:"steps,omitempty"`
	Hide  bool        `json:"hide,omitempty"`
}

func validateMetaNamespace() error {
	if metaSetNamespace == "" {
		return fmt.Errorf("please provide --namespace value")
	}
	return nil
}

func getMetaNameSpace(ctx context.Context) (*models.IdmUserMetaNamespace, error) {	
	client := sdkClient.GetApiClient()

	params := &user_meta_service.ListUserMetaNamespaceParams{
		Context: ctx,
	}
	result, err := client.UserMetaService.ListUserMetaNamespace(params)

	if err != nil {
		return nil, err
	}

	for _, n := range result.Payload.Namespaces {
		if n.Namespace == metaSetNamespace {
			return n, nil
		}
	}
	return nil, fmt.Errorf("invalid namespace %s", metaSetNamespace)
}

func getNamespaceDefinition(n *models.IdmUserMetaNamespace) (*metaNsDef, error) {
	mtdef := metaNsDef{}
	if err := json.Unmarshal([]byte(n.JSONDefinition), &mtdef); err != nil {
		return nil, fmt.Errorf("malform JSONdefinition: namespace %s", metaSetNamespace)
	}
	return &mtdef, nil
}

func validateFilePath() error {
	if metaSetNodePath == "" {
		return fmt.Errorf("node path is not found")
	}
	return nil
}

func validateFileExist(ctx context.Context) (*models.TreeNode, error) {
	p := strings.Trim(metaSetNodePath, "/")
	node, exists := sdkClient.StatNode(ctx, p)

	if !exists && node == nil {
		return nil, fmt.Errorf("could not stat node, no folder/file found at %s", p)
	}
	return node, nil
}

func validateOperation() (models.UpdateUserMetaRequestUserMetaOp, error) {
	switch metaSetOperation {
	case "update":
		return models.UpdateUserMetaRequestUserMetaOpPUT, nil
	case "delete":
		return models.UpdateUserMetaRequestUserMetaOpDELETE, nil
	default:

	}
	return "", fmt.Errorf("--operation parameter is required, please provide either \"update\" or \"delete\"")
}

// Perform effective operation
func do(ctx context.Context, client *client.PydioCellsRestAPI, node *models.TreeNode, value string) error {
	opPut := models.UpdateUserMetaRequestUserMetaOpPUT
	params := &user_meta_service.UpdateUserMetaParams{
		Body: &models.IdmUpdateUserMetaRequest{
			MetaDatas: []*models.IdmUserMeta{
				{
					Namespace: metaSetNamespace,
					NodeUUID:  node.UUID,
					JSONValue: value,
				},
			},
			// Use PUT with empty value for DELETE
			Operation: &opPut,
		},
		Context: ctx,
	}
	_, err := client.UserMetaService.UpdateUserMeta(params)
	return err
}

func isString(t string) bool {
	return t == "textarea" ||
		t == "string" ||
		t == "choice" || // specific strategy validation: limited values
		t == "tags" || // specific strategy update
		t == "date" || // specific type: accept input as a string then convert to timestamp then convert to string
		t == "css_label" // specific strategy validation: limited values
}

func isJson(t string) bool {
	return t == "json"
}

func isNumeric(t string) bool {
	return t == "integer" ||
		t == "stars_rate" // specific strategy validation: limited values
}

func isBoolean(t string) bool {
	return t == "boolean"
}

// Make sure user use correctly parameter to update current meta namespace
func validateMetaFlagTypeMatch(cmd *cobra.Command, t string) error {
	o, _ := validateOperation()
	if o == models.UpdateUserMetaRequestUserMetaOpDELETE {		
		return nil
	}
	if isString(t) && !cmd.Flags().Changed("string-value") || 
		cmd.Flags().Changed("string-value") && metaSetStringValue == "" {
		return fmt.Errorf("--string-value is required")
	}

	if isBoolean(t) && !cmd.Flags().Changed("bollean-value") {
		return fmt.Errorf("--boolean-value is required")
	}

	if isJson(t) && !cmd.Flags().Changed("json-value") ||
		cmd.Flags().Changed("json-value") && metaSetJsonValue == "" {
		return fmt.Errorf("--json-value is required")
	}

	if isNumeric(t) && !cmd.Flags().Changed("numeric-value") {
		return fmt.Errorf("--numeric-value is required")
	}
	return nil
}

type userValues struct {
	StrValue     string
	JsonValue    json.RawMessage
	BooleanValue bool
	NumericValue int64
	StarValue    int
}

// formatInputData attempts to validate, format, and store different input types into the userValues struct.
func formatInputData(cmd *cobra.Command, metaType string) (*userValues, error) {
	uValues := userValues{}
	if err := validateMetaFlagTypeMatch(cmd, metaType); err != nil {
		return &uValues, err
	}
	
	metaNamespace, err := getMetaNameSpace(cmd.Context())
	if err != nil {
		return &uValues, err
	}
	nsDef, err := getNamespaceDefinition(metaNamespace)
	if err != nil {
		return &uValues, err
	}
	op, err := validateOperation()
	if err != nil {
		return &uValues, err
	}
	if isString(metaType) {
		switch metaType {
		case "choice":
			allowedValues, err := getChoiceValues(nsDef)
			if err != nil {
				return &uValues, err
			}
			if contains(allowedValues, metaSetStringValue) || len(allowedValues) == 0 {
				uValues.StrValue = metaSetStringValue
				return &uValues, nil
			}
			return &uValues, fmt.Errorf("value must be one of following: %s, given value: %s", strings.Join(allowedValues, ","), metaSetStringValue)
		case "tags":

			// TODO multiple values for tags
			if metaSetStringValue == "" {
				return &uValues, fmt.Errorf("delete operation on tags type require --string-value")
			}
			existValues, err := getCurrentTagsValues(cmd.Context())
			if err != nil {
				return &uValues, err
			}
		
			switch op {
			case models.UpdateUserMetaRequestUserMetaOpPUT:
				newTags := appendUnique(existValues, metaSetStringValue)
				uValues.StrValue = strings.Join(newTags, ",")
			case models.UpdateUserMetaRequestUserMetaOpDELETE:
				newTags := removeString(existValues, metaSetStringValue)
				uValues.StrValue = strings.Join(newTags, ",")
			}
			return &uValues, nil
		case "date":
			date, err := validateDate(metaSetStringValue)
			if err != nil {
				return &uValues, err
			}
			uValues.StrValue = fmt.Sprintf("%d", date.Unix())
		case "css_label":
			uValues.StrValue = metaSetStringValue
		default:
			uValues.StrValue = metaSetStringValue
		}
		return &uValues, nil
	}

	if isJson(metaType) {
		var js json.RawMessage
		err := json.Unmarshal([]byte(metaSetJsonValue), &js)
		if err != nil {
			return &uValues, err
		}
		uValues.JsonValue = js
		return &uValues, nil
	}

	if isBoolean(metaType) {
		uValues.BooleanValue = metaSetBooleanValue
	}

	if isNumeric(metaType) {
		switch metaType {
		case "stars_rate":
			if !isStarValue(metaSetNumericValue) {
				return &uValues, fmt.Errorf("stars_rate type accepts value from 0 to 5")
			}
		default:
		}
		uValues.NumericValue = int64(metaSetNumericValue)
		return &uValues, nil
	}
	return &uValues, nil
}

// getFinalJsonValue retrieves the formatted value from userValues and wraps it in a JSON string for the API call.
func getFinalJsonValue(t string, uv *userValues) string {
	o, _ := validateOperation()
	if o == models.UpdateUserMetaRequestUserMetaOpPUT {
		if isString(t) {
			return fmt.Sprintf("\"%s\"", uv.StrValue)
		}
		
		if isBoolean(t){		
			return fmt.Sprintf("\"%t\"", uv.BooleanValue)
		}

		if isNumeric(t) {
			return fmt.Sprintf("\"%d\"", uv.NumericValue)
		}

		if isJson(t) {
			return string(uv.JsonValue)
		}
	}

	// Delete
	if t == "tags" {
		return fmt.Sprintf("\"%s\"", uv.StrValue) 
	}

	return emptyJson
}

func appendUnique(slice []string, str string) []string {
	for _, s := range slice {
		if s == str {
			return slice // Already exists, return unchanged
		}
	}
	return append(slice, str) // Add since it's unique
}

func removeString(slice []string, str string) []string {
	var result []string
	for _, s := range slice {
		if s != str {
			result = append(result, s)
		}
	}
	return result
}

func isStarValue(starRate int) bool {
	return starRate >= 0 && starRate <= 5
}

func getChoiceValues(ndef *metaNsDef) ([]string, error) {
	type kv struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	type choiceData struct {
		Items []kv `json:"items"`
	}

	choiceValues := choiceData{}

	byteData, err := json.Marshal(ndef.Data)
	if err != nil {
		return []string{}, err
	}

	err = json.Unmarshal(byteData, &choiceValues)
	if err != nil {
		return []string{}, err
	}

	var keys []string
	for _, item := range choiceValues.Items {
		keys = append(keys, item.Key)
	}
	return keys, nil
}

func getCurrentTagsValues(ctx context.Context) ([]string, error) {
	client := sdkClient.GetApiClient()
	params := &meta_service.GetBulkMetaParams{
		Body: &models.RestGetBulkMetaRequest{
			NodePaths:        []string{metaSetNodePath},
			AllMetaProviders: true,
		},
		Context: ctx,
	}

	result, err := client.MetaService.GetBulkMeta(params)
	if err != nil {
		return []string{}, nil
	}
	node := result.Payload.Nodes[0]
	if tagsValue, ok := node.MetaStore[metaSetNamespace]; ok {
		return strings.Split(strings.Trim(tagsValue, "\""), ","), nil
	}
	return []string{}, nil
}

func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func validateDate(s string) (time.Time, error) {
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date-time format for %s. Please use the format: \"2006-01-02 15:04:05\"", s)
	}
	return t, nil
}
