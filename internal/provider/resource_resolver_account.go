package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

var _ resource.Resource = &ResolverAccountResource{}

// NatsRequester abstracts the NATS connection methods we need for testability.
type NatsRequester interface {
	Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error)
	Close()
}

type ResolverAccountResource struct {
	connectFunc      func(url, creds string) (*nats.Conn, error)
	providerData     *NatsjwtProviderData
	testConnOverride NatsRequester // test-only: overrides getConnection when set
}

type ResolverAccountResourceModel struct {
	JWT          types.String `tfsdk:"jwt"`
	OperatorSeed types.String `tfsdk:"operator_seed"`
}

func NewResolverAccountResource() resource.Resource {
	return &ResolverAccountResource{
		connectFunc: connectNATS,
	}
}

func (r *ResolverAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resolver_account"
}

func (r *ResolverAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Uploads or updates an account JWT on a NATS server using the NATS-based resolver.",
		Attributes: map[string]schema.Attribute{
			"jwt": schema.StringAttribute{
				Required:    true,
				Description: "The signed account JWT to push to the NATS resolver.",
			},
			"operator_seed": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Operator seed used to sign deletion requests. If omitted, terraform destroy will only remove the resource from state and will NOT delete the account from the server.",
				Validators:  []validator.String{SeedTypeValidator(nkeys.PrefixByteOperator)},
			},
		},
	}
}

func (r *ResolverAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*NatsjwtProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *NatsjwtProviderData, got %T", req.ProviderData),
		)
		return
	}
	r.providerData = config
}

func (r *ResolverAccountResource) getConnection() (NatsRequester, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r.testConnOverride != nil {
		return r.testConnOverride, diags
	}
	if r.providerData == nil {
		diags.AddError("Provider Not Configured", "Provider data is missing. Ensure 'nats_url' is set in the provider configuration.")
		return nil, diags
	}
	if r.providerData.NatsUrl.IsNull() || r.providerData.NatsUrl.ValueString() == "" {
		diags.AddError(
			"Missing Provider Configuration",
			"The provider attribute 'nats_url' is required when using natsjwt_resolver_account resources.",
		)
		return nil, diags
	}
	nc, err := r.connectFunc(r.providerData.NatsUrl.ValueString(), r.providerData.Creds.ValueString())
	if err != nil {
		diags.AddError("NATS Connection Error", err.Error())
		return nil, diags
	}
	return nc, diags
}

// updateResponse matches the JSON structure returned by NATS on $SYS.REQ.CLAIMS.UPDATE.
type updateResponse struct {
	Server map[string]interface{} `json:"server,omitempty"`
	Data   *updateData            `json:"data,omitempty"`
	Error  *updateError           `json:"error,omitempty"`
}

type updateData struct {
	Account string `json:"account,omitempty"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type updateError struct {
	Account     string `json:"account,omitempty"`
	Code        int    `json:"code"`
	Description string `json:"description"`
}

func (r *ResolverAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResolverAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nc, diags := r.getConnection()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer nc.Close()

	if err := r.pushJWT(nc, data.JWT.ValueString(), &resp.Diagnostics); err != nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResolverAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResolverAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nc, diags := r.getConnection()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer nc.Close()

	claims, err := natsjwt.DecodeAccountClaims(data.JWT.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid JWT", fmt.Sprintf("Failed to decode account JWT: %s", err))
		return
	}

	lookupSubj := fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject)
	msg, err := nc.Request(lookupSubj, nil, 5*time.Second)
	if err != nil {
		if err == nats.ErrNoResponders {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Failed to read account claims",
			fmt.Sprintf("Failed to request account claims for subject %q: %s", claims.Subject, err),
		)
		return
	}
	if len(msg.Data) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Compare the returned JWT with the stored one.
	if string(msg.Data) != data.JWT.ValueString() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResolverAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResolverAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nc, diags := r.getConnection()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer nc.Close()

	if err := r.pushJWT(nc, data.JWT.ValueString(), &resp.Diagnostics); err != nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResolverAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResolverAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.OperatorSeed.IsNull() || data.OperatorSeed.ValueString() == "" {
		resp.Diagnostics.AddWarning(
			"Account Not Deleted from Server",
			"The 'operator_seed' attribute was not provided, so the account JWT was not deleted from the NATS resolver server. It has only been removed from Terraform state.",
		)
		return
	}

	nc, diags := r.getConnection()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer nc.Close()

	claims, err := natsjwt.DecodeAccountClaims(data.JWT.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid JWT", fmt.Sprintf("Failed to decode account JWT: %s", err))
		return
	}

	operatorKP, err := keypairFromSeed(data.OperatorSeed.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Operator Seed", fmt.Sprintf("Failed to parse operator seed: %s", err))
		return
	}

	// Build a generic delete request claim for this account.
	genClaims := natsjwt.NewGenericClaims(claims.Subject)
	genClaims.Data = map[string]interface{}{
		"accounts": []string{claims.Subject},
	}

	deleteJWT, err := issuance.EncodeDeterministic(genClaims, operatorKP)
	if err != nil {
		resp.Diagnostics.AddError("JWT Encoding Error", fmt.Sprintf("Failed to encode delete request JWT: %s", err))
		return
	}

	msg, err := nc.Request("$SYS.REQ.CLAIMS.DELETE", []byte(deleteJWT), 5*time.Second)
	if err != nil {
		resp.Diagnostics.AddError("NATS Request Error", fmt.Sprintf("Failed to send delete request: %s", err))
		return
	}

	var ur updateResponse
	if err := json.Unmarshal(msg.Data, &ur); err != nil {
		resp.Diagnostics.AddError("Response Parse Error", fmt.Sprintf("Failed to parse delete response: %s", err))
		return
	}
	if ur.Error != nil {
		resp.Diagnostics.AddError(
			"Delete Failed",
			fmt.Sprintf("Server returned error (code %d): %s", ur.Error.Code, ur.Error.Description),
		)
		return
	}
}

func (r *ResolverAccountResource) pushJWT(nc NatsRequester, jwtStr string, diags *diag.Diagnostics) error {
	claims, err := natsjwt.DecodeAccountClaims(jwtStr)
	if err != nil {
		diags.AddError("Invalid JWT", fmt.Sprintf("Failed to decode account JWT: %s", err))
		return err
	}

	updateSubj := fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject)
	msg, err := nc.Request(updateSubj, []byte(jwtStr), 5*time.Second)
	if err != nil {
		diags.AddError("NATS Request Error", fmt.Sprintf("Failed to send update request: %s", err))
		return err
	}

	var ur updateResponse
	if err := json.Unmarshal(msg.Data, &ur); err != nil {
		diags.AddError("Response Parse Error", fmt.Sprintf("Failed to parse update response: %s", err))
		return err
	}
	if ur.Error != nil {
		diags.AddError(
			"Update Failed",
			fmt.Sprintf("Server returned error (code %d): %s", ur.Error.Code, ur.Error.Description),
		)
		return fmt.Errorf("update failed: %s", ur.Error.Description)
	}
	return nil
}
