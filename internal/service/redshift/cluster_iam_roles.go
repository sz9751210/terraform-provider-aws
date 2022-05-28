package redshift

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceClusterIamRoles() *schema.Resource {
	return &schema.Resource{
		Create: resourceClusterIamRolesCreate,
		Read:   resourceClusterIamRolesRead,
		Update: resourceClusterIamRolesUpdate,
		Delete: resourceClusterIamRolesDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(75 * time.Minute),
			Update: schema.DefaultTimeout(75 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"default_iam_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"iam_roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: verify.ValidARN,
				},
			},
		},
	}
}

func resourceClusterIamRolesCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).RedshiftConn

	input := &redshift.ModifyClusterIamRolesInput{
		ClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
	}

	if v, ok := d.GetOk("iam_roles"); ok && v.(*schema.Set).Len() > 0 {
		input.AddIamRoles = flex.ExpandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("default_iam_role_arn"); ok {
		input.DefaultIamRoleArn = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Adding Redshift Cluster Iam Roles IAM Roles: %s", input)
	out, err := conn.ModifyClusterIamRoles(input)

	if err != nil {
		return fmt.Errorf("error adding Redshift Cluster Iam Roles (%s) IAM roles: %w", d.Id(), err)
	}

	d.SetId(aws.StringValue(out.Cluster.ClusterIdentifier))

	if _, err := waitClusterUpdated(conn, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
		return fmt.Errorf("error waiting for Redshift Cluster Iam Roles (%s) update: %w", d.Id(), err)
	}

	return resourceClusterIamRolesRead(d, meta)
}

func resourceClusterIamRolesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).RedshiftConn

	rsc, err := FindClusterByID(conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Redshift Cluster Iam Roles (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading Redshift Cluster Iam Roles (%s): %w", d.Id(), err)
	}

	var apiList []*string

	for _, iamRole := range rsc.IamRoles {
		apiList = append(apiList, iamRole.IamRoleArn)
	}
	d.Set("iam_roles", aws.StringValueSlice(apiList))
	d.Set("default_iam_role_arn", rsc.DefaultIamRoleArn)
	d.Set("cluster_identifier", rsc.ClusterIdentifier)

	return nil
}

func resourceClusterIamRolesUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).RedshiftConn

	o, n := d.GetChange("iam_roles")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)
	add := ns.Difference(os)
	del := os.Difference(ns)

	input := &redshift.ModifyClusterIamRolesInput{
		AddIamRoles:       flex.ExpandStringSet(add),
		ClusterIdentifier: aws.String(d.Id()),
		RemoveIamRoles:    flex.ExpandStringSet(del),
		DefaultIamRoleArn: aws.String(d.Get("default_iam_role_arn").(string)),
	}

	log.Printf("[DEBUG] Modifying Redshift Cluster Iam Roles IAM Roles: %s", input)
	_, err := conn.ModifyClusterIamRoles(input)

	if err != nil {
		return fmt.Errorf("error modifying Redshift Cluster Iam Roles (%s) IAM roles: %w", d.Id(), err)
	}

	if _, err := waitClusterUpdated(conn, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
		return fmt.Errorf("error waiting for Redshift Cluster Iam Roles (%s) update: %w", d.Id(), err)
	}

	return resourceClusterIamRolesRead(d, meta)
}

func resourceClusterIamRolesDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).RedshiftConn

	input := &redshift.ModifyClusterIamRolesInput{
		ClusterIdentifier: aws.String(d.Id()),
		RemoveIamRoles:    flex.ExpandStringSet(d.Get("iam_roles").(*schema.Set)),
		DefaultIamRoleArn: aws.String(d.Get("default_iam_role_arn").(string)),
	}

	log.Printf("[DEBUG] Removing Redshift Cluster Iam Roles IAM Roles: %s", input)
	_, err := conn.ModifyClusterIamRoles(input)

	if err != nil {
		return fmt.Errorf("error removing Redshift Cluster Iam Roles (%s) IAM roles: %w", d.Id(), err)
	}

	if _, err := waitClusterUpdated(conn, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
		return fmt.Errorf("error waiting for Redshift Cluster Iam Roles (%s) removal: %w", d.Id(), err)
	}

	return nil
}
