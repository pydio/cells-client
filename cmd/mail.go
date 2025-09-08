package cmd

import (
	"fmt"
	"log"
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/pydio/cells-sdk-go/v4/client/mailer_service"
	"github.com/pydio/cells-sdk-go/v4/client/user_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

func NewCellInvitation(ctx context.Context, targetUserLogin string, cellName string, cellPath string) {
	apiClient := sdkClient.GetApiClient()
	config := sdkClient.GetConfig()
	sdkClient.GetConfig()

	// Get current user record
	currentUser, err := apiClient.UserService.GetUser(&user_service.GetUserParams{
		Login:   config.User,
		Context: ctx,
	})
	if err != nil {
		log.Fatalf("Error retrieving current user: %v", err)
	}

	// Get target user record
	targetUser, err := apiClient.UserService.GetUser(&user_service.GetUserParams{
		Login:   targetUserLogin,
		Context: ctx,
	})
	if err != nil {
		log.Fatalf("Error retrieving targetuser: %v", err)
	}

	// Create mailer service client
	mailer := mailer_service.New(apiClient.Transport, strfmt.Default)

	// Prepare mail params
	params := &mailer_service.SendParams{
		Context: ctx,
		Body: &models.MailerMail{
			From: &models.MailerUser{
				Address: currentUser.Payload.Attributes["email"],
				Name: currentUser.Payload.Attributes["displayName"],
				//UUID: currentUser.Payload.UUID,
			},
			To: []*models.MailerUser{
				{
					Address: targetUser.Payload.Attributes["email"],
					Name:    targetUser.Payload.Attributes["displayName"],
					//UUID:    targetUser.Payload.UUID,
				},
			},
			TemplateID: "Cell",
			TemplateData: map[string]string{
				"Cell":     cellName,
				"LinkPath": "/ws-" + cellPath,
				"Inviter":  "Admin",
			},
		},
	}

	// Send the email
	_, err = mailer.Send(params)
	if err != nil {
		log.Fatalf("Failed to send mail: %v", err)
	}
	fmt.Println("Mail sent successfully!")
}