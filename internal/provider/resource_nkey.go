package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nats-io/nkeys"
)

var _ resource.Resource = &NkeyResource{}

type NkeyResource struct{}

type NkeyResourceModel struct {
	Keepers   types.Map    `tfsdk:"keepers"`
	Type      types.String `tfsdk:"type"`
	Seed      types.String `tfsdk:"seed"`
	PublicKey types.String `tfsdk:"public_key"`
}

func NewNkeyResource() resource.Resource {
	return &NkeyResource{}
}

func (r *NkeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nkey"
}

func (r *NkeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generates an NKey pair (seed + public key) for NATS authentication.",
		Attributes: map[string]schema.Attribute{
			"keepers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Arbitrary map of values that, when changed, will trigger recreation of the resource. Similar to the random provider's keepers.",
				PlanModifiers: []planmodifier.Map{
					requiresReplaceIfValuesNotNull{},
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Type of NKey to generate: operator, account, or user.",
				Validators:  []validator.String{NkeyTypeValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"seed": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The generated NKey seed (private key). Starts with SO (operator), SA (account), or SU (user).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_key": schema.StringAttribute{
				Computed:    true,
				Description: "The NKey public key. Starts with O (operator), A (account), or U (user).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NkeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NkeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prefixByte, err := prefixByteFromType(data.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Key Type", err.Error())
		return
	}

	kp, err := nkeys.CreatePair(prefixByte)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Create NKey", fmt.Sprintf("Could not create NKey pair: %s", err))
		return
	}

	seed, err := kp.Seed()
	if err != nil {
		resp.Diagnostics.AddError("Failed to Get Seed", fmt.Sprintf("Could not get seed from keypair: %s", err))
		return
	}

	pub, err := kp.PublicKey()
	if err != nil {
		resp.Diagnostics.AddError("Failed to Get Public Key", fmt.Sprintf("Could not get public key from keypair: %s", err))
		return
	}

	data.Seed = types.StringValue(string(seed))
	data.PublicKey = types.StringValue(pub)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NkeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NkeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-derive public key from seed to verify consistency
	kp, err := nkeys.FromSeed([]byte(data.Seed.ValueString()))
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	pub, err := kp.PublicKey()
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.PublicKey = types.StringValue(pub)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NkeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NkeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve seed and public_key from state (only keepers can change in-place)
	var state NkeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Seed = state.Seed
	data.PublicKey = state.PublicKey

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NkeyResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: state removal is handled by the framework
}

// requiresReplaceIfValuesNotNull triggers replacement when keeper values change from non-null.
type requiresReplaceIfValuesNotNull struct{}

func (r requiresReplaceIfValuesNotNull) Description(_ context.Context) string {
	return "Requires replacement when keeper values change from non-null."
}

func (r requiresReplaceIfValuesNotNull) MarkdownDescription(ctx context.Context) string {
	return r.Description(ctx)
}

func (r requiresReplaceIfValuesNotNull) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.StateValue.IsNull() {
		return
	}

	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	var stateKeepers, planKeepers map[string]types.String
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &stateKeepers, false)...)
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planKeepers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for key, planVal := range planKeepers {
		stateVal, exists := stateKeepers[key]
		if !exists {
			continue
		}
		if !stateVal.IsNull() && planVal != stateVal {
			resp.RequiresReplace = true
			return
		}
	}

	for key, stateVal := range stateKeepers {
		if _, exists := planKeepers[key]; !exists && !stateVal.IsNull() {
			resp.RequiresReplace = true
			return
		}
	}
}
