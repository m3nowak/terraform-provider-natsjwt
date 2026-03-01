package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &seedPublicKeyFunction{}

func NewSeedPublicKeyFunction() function.Function {
	return &seedPublicKeyFunction{}
}

type seedPublicKeyFunction struct{}

func (f *seedPublicKeyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "seed_public_key"
}

func (f *seedPublicKeyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Converts an NATS NKey seed into its corresponding public key.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "seed",
				Description: "NATS seed (SO..., SA..., or SU...) to convert.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *seedPublicKeyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var seed string
	resp.Error = req.Arguments.GetArgument(ctx, 0, &seed)
	if resp.Error != nil {
		return
	}

	publicKey, err := publicKeyFromSeed(seed)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("failed to convert seed to public key: %s", err))
		return
	}

	resp.Error = resp.Result.Set(ctx, publicKey)
}
