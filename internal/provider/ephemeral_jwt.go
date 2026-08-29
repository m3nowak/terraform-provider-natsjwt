package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
)

var _ ephemeral.EphemeralResource = &jwtEphemeralResource{}

type jwtEphemeralResource struct {
	typeName    string
	description string
	attributes  func() map[string]schema.Attribute
	open        func(context.Context, ephemeral.OpenRequest, *ephemeral.OpenResponse)
}

func NewOperatorEphemeralResource() ephemeral.EphemeralResource {
	return &jwtEphemeralResource{
		typeName:    "operator",
		description: "Generates an ephemeral signed NATS operator JWT without persisting its seed or result in Terraform artifacts.",
		attributes: func() map[string]schema.Attribute {
			var response datasource.SchemaResponse
			(&OperatorDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &response)
			return response.Schema.Attributes
		},
		open: openOperatorEphemeralResource,
	}
}

func NewAccountEphemeralResource() ephemeral.EphemeralResource {
	return newAccountEphemeralResource("account", false)
}

func NewSystemAccountEphemeralResource() ephemeral.EphemeralResource {
	return newAccountEphemeralResource("system_account", true)
}

func newAccountEphemeralResource(typeName string, system bool) ephemeral.EphemeralResource {
	return &jwtEphemeralResource{
		typeName:    typeName,
		description: "Generates an ephemeral signed NATS " + typeName + " JWT without persisting its seeds or result in Terraform artifacts.",
		attributes:  accountSchemaAttributes,
		open: func(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
			openAccountEphemeralResource(ctx, req, resp, system)
		},
	}
}

func NewUserEphemeralResource() ephemeral.EphemeralResource {
	return &jwtEphemeralResource{
		typeName:    "user",
		description: "Generates ephemeral NATS user credentials without persisting seeds, JWTs, or credentials in Terraform artifacts.",
		attributes: func() map[string]schema.Attribute {
			var response datasource.SchemaResponse
			(&UserDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &response)
			return response.Schema.Attributes
		},
		open: openUserEphemeralResource,
	}
}

func (r *jwtEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.typeName
}

func (r *jwtEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	attributes, err := ephemeralSchemaAttributes(r.attributes())
	if err != nil {
		resp.Diagnostics.AddError("Ephemeral Schema Error", err.Error())
		return
	}
	resp.Schema = ephemeralschema.Schema{
		Description: r.description,
		Attributes:  attributes,
	}
}

func (r *jwtEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	r.open(ctx, req, resp)
}

func openOperatorEphemeralResource(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data OperatorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	generateOperator(ctx, &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	}
}

func openAccountEphemeralResource(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse, system bool) {
	var data AccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	generateAccount(ctx, &data, system, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	}
}

func openUserEphemeralResource(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	generateUser(ctx, &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	}
}

func ephemeralSchemaAttributes(attributes map[string]schema.Attribute) (map[string]ephemeralschema.Attribute, error) {
	result := make(map[string]ephemeralschema.Attribute, len(attributes))
	for name, attribute := range attributes {
		switch value := attribute.(type) {
		case schema.StringAttribute:
			result[name] = ephemeralschema.StringAttribute{Required: value.Required, Optional: value.Optional, Computed: value.Computed, Sensitive: value.Sensitive, Description: value.Description, Validators: value.Validators}
		case schema.Int64Attribute:
			result[name] = ephemeralschema.Int64Attribute{Required: value.Required, Optional: value.Optional, Computed: value.Computed, Sensitive: value.Sensitive, Description: value.Description, Validators: value.Validators}
		case schema.BoolAttribute:
			result[name] = ephemeralschema.BoolAttribute{Required: value.Required, Optional: value.Optional, Computed: value.Computed, Sensitive: value.Sensitive, Description: value.Description, Validators: value.Validators}
		case schema.ListAttribute:
			result[name] = ephemeralschema.ListAttribute{ElementType: value.ElementType, Required: value.Required, Optional: value.Optional, Computed: value.Computed, Sensitive: value.Sensitive, Description: value.Description, Validators: value.Validators}
		case schema.SingleNestedAttribute:
			nested, err := ephemeralSchemaAttributes(value.Attributes)
			if err != nil {
				return nil, fmt.Errorf("attribute %q: %w", name, err)
			}
			result[name] = ephemeralschema.SingleNestedAttribute{Attributes: nested, Required: value.Required, Optional: value.Optional, Computed: value.Computed, Sensitive: value.Sensitive, Description: value.Description, Validators: value.Validators}
		case schema.ListNestedAttribute:
			nested, err := ephemeralSchemaAttributes(value.NestedObject.Attributes)
			if err != nil {
				return nil, fmt.Errorf("attribute %q: %w", name, err)
			}
			result[name] = ephemeralschema.ListNestedAttribute{NestedObject: ephemeralschema.NestedAttributeObject{Attributes: nested}, Required: value.Required, Optional: value.Optional, Computed: value.Computed, Sensitive: value.Sensitive, Description: value.Description, Validators: value.Validators}
		default:
			return nil, fmt.Errorf("attribute %q uses unsupported type %T", name, attribute)
		}
	}
	return result, nil
}
