package rest

import (
	"context"
	"fmt"

	"github.com/pydio/cells-sdk-go/v4/client/share_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

func (client *SdkClient) CreateSimpleFolderLink(ctx context.Context, targetNodeUuid, label string) (*models.RestShareLink, error) {

	perm := []*models.RestShareLinkAccessType{
		models.NewRestShareLinkAccessType(models.RestShareLinkAccessTypeDownload),
		models.NewRestShareLinkAccessType(models.RestShareLinkAccessTypePreview),
	}

	link := &models.RestShareLink{
		Label:                   label,
		RootNodes:               []*models.TreeNode{{UUID: targetNodeUuid}},
		Permissions:             perm,
		ViewTemplateName:        "pydio_shared_folder",
		PoliciesContextEditable: true,
	}

	params := (&share_service.PutShareLinkParams{}).WithContext(ctx).WithBody(&models.RestPutShareLinkRequest{
		ShareLink:       link,
		PasswordEnabled: false,
	})

	resp, err := client.GetApiClient().ShareService.PutShareLink(params)
	if err != nil {
		return nil, fmt.Errorf("call to PutShareLink for %s has failed, cause: %s", label, err.Error())
	}

	return resp.Payload, nil
}
