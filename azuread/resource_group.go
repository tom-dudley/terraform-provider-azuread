package azuread

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-azuread/azuread/helpers/ar"
	"github.com/terraform-providers/terraform-provider-azuread/azuread/helpers/p"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceGroupCreate,
		Read:   resourceGroupRead,
		Update: resourceGroupUpdate,
		Delete: resourceGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"members": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).groupsClient
	ctx := meta.(*ArmClient).StopContext

	name := d.Get("name").(string)

	properties := graphrbac.GroupCreateParameters{
		DisplayName:     &name,
		MailEnabled:     p.Bool(false), //we're defaulting to false, as the API currently only supports the creation of non-mail enabled security groups.
		MailNickname:    &name,
		SecurityEnabled: p.Bool(true), //we're defaulting to true, as the API currently only supports the creation of non-mail enabled security groups.
	}

	group, err := client.Create(ctx, properties)
	if err != nil {
		return err
	}

	d.SetId(*group.ObjectID)

	members := d.Get("members").(*schema.Set)
	err = addMembers(*group.ObjectID, members, meta)

	if err != nil {
		return err
	}

	return resourceGroupRead(d, meta)
}

func addMembers(groupID string, members *schema.Set, meta interface{}) error {
	client := meta.(*ArmClient).groupsClient
	ctx := meta.(*ArmClient).StopContext

	for _, member := range members.List() {
		user, err := userRead(member.(string), meta)
		if err != nil {
			return err
		}

		tenantID := meta.(*ArmClient).tenantID
		memberURL := "https://graph.windows.net/" + tenantID + "/directoryObjects/" + *user.ObjectID
		parameters := graphrbac.GroupAddMemberParameters{URL: &memberURL}
		client.AddMember(ctx, groupID, parameters)
	}

	return nil
}

func removeMembers(groupID string, members *schema.Set, meta interface{}) error {
	client := meta.(*ArmClient).groupsClient
	ctx := meta.(*ArmClient).StopContext

	for _, member := range members.List() {
		user, err := userRead(member.(string), meta)
		if err != nil {
			return err
		}

		client.RemoveMember(ctx, groupID, *user.ObjectID)
	}

	return nil
}

func resourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).groupsClient
	ctx := meta.(*ArmClient).StopContext

	resp, err := client.Get(ctx, d.Id())
	if err != nil {
		if ar.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Azure AD group with id %q was not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Azure AD Group with ID %q: %+v", d.Id(), err)
	}

	d.Set("name", resp.DisplayName)

	return nil
}

func resourceGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("members") {
		o, n := d.GetChange("members")
		oldMembers := o.(*schema.Set)
		newMembers := n.(*schema.Set)

		removedMembers := oldMembers.Difference(newMembers)
		addedMembers := newMembers.Difference(oldMembers)

		removeMembers(d.Id(), removedMembers, meta)
		addMembers(d.Id(), addedMembers, meta)
	}

	return nil
}

func resourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).groupsClient
	ctx := meta.(*ArmClient).StopContext

	if resp, err := client.Delete(ctx, d.Id()); err != nil {
		if !ar.ResponseWasNotFound(resp) {
			return fmt.Errorf("Error Deleting Azure AD Group with ID %q: %+v", d.Id(), err)
		}
	}

	return nil
}
