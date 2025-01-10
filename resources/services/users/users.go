package users

import (
	"context"
	"encoding/json"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/cloudquery/plugin-sdk/v4/types"
	"github.com/jsifuentes/cq-source-googleworkspace/client"
	directory "google.golang.org/api/admin/directory/v1"
)

func UsersTable() *schema.Table {
	return &schema.Table{
		Name:        "googleworkspace_users",
		Description: "Google Workspace Users",
		Transform:   transformers.TransformWithStruct(&directory.User{}, transformers.WithPrimaryKeys("Id")),
		Columns: []schema.Column{
			client.CustomerIDColumn,
			{
				Name: "first_name",
				Type: arrow.BinaryTypes.String,
				Resolver: func(_ context.Context, meta schema.ClientMeta, r *schema.Resource, c schema.Column) error {
					return r.Set(c.Name, r.Item.(*directory.User).Name.GivenName)
				},
			},
			{
				Name: "last_name",
				Type: arrow.BinaryTypes.String,
				Resolver: func(_ context.Context, meta schema.ClientMeta, r *schema.Resource, c schema.Column) error {
					return r.Set(c.Name, r.Item.(*directory.User).Name.FamilyName)
				},
			},
			{
				Name: "organizations",
				Type: types.NewJSONType(),
				Resolver: func(_ context.Context, meta schema.ClientMeta, r *schema.Resource, c schema.Column) error {
					user := r.Item.(*directory.User)
					if orgs, ok := user.Organizations.([]interface{}); ok {
						orgsJSON, err := json.Marshal(orgs)
						if err != nil {
							return err
						}
						return r.Set(c.Name, string(orgsJSON))
					}
					return nil
				},
			},
		},
		Relations: []*schema.Table{
			UserAliasesTable(),
		},
		Resolver: fetchUsers,
	}
}

func fetchUsers(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)
	return c.DirectoryService.Users.List().Customer(c.Spec.CustomerID).Projection("full").Pages(ctx, func(users *directory.Users) error {
		for _, u := range users.Users {
			res <- u
		}
		return nil
	})
}
