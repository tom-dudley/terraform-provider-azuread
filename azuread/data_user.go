package azuread

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azuread/azuread/helpers/ar"
	"github.com/terraform-providers/terraform-provider-azuread/azuread/helpers/validate"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceUserRead,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"user_principal_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validate.NoEmptyStrings,
			},

			"account_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mail": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mail_nickname": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceUserRead(d *schema.ResourceData, meta interface{}) (err error) {
	queryString := d.Get("user_principal_name").(string)

	user, err := userRead(queryString, meta)

	d.SetId(*user.ObjectID)
	d.Set("user_principal_name", user.UserPrincipalName)
	d.Set("account_enabled", user.AccountEnabled)
	d.Set("display_name", user.DisplayName)
	d.Set("mail", user.Mail)
	d.Set("mail_nickname", user.MailNickname)

	return
}

func userRead(queryString string, meta interface{}) (user graphrbac.User, err error) {
	client := meta.(*ArmClient).usersClient
	ctx := meta.(*ArmClient).StopContext

	log.Printf("[DEBUG] Using Get with the following query string: %q", queryString)
	user, err = client.Get(ctx, queryString)
	if err != nil {
		if ar.ResponseWasNotFound(user.Response) {
			err = fmt.Errorf("Error: No AzureAD User found with the following query string: %q", queryString)
			return
		}
		err = fmt.Errorf("Error making Read request on AzureAD User the following query string: %q: %+v", queryString, err)
		return
	}

	if user.ObjectID == nil {
		err = fmt.Errorf("User objectId is nil")
		return
	}

	return user, nil
}
