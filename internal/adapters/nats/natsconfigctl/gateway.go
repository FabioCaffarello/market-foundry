package natsconfigctl

import (
	"context"

	"internal/adapters/nats/natskit"
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/problem"
)

type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.ConfigctlGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) CreateDraft(ctx context.Context, command contracts.CreateDraftCommand) (contracts.CreateDraftReply, *problem.Problem) {
	return invokeControl[contracts.CreateDraftCommand, contracts.CreateDraftReply](ctx, g, g.registry.CreateDraft, command, "request configctl create draft")
}

func (g *Gateway) GetConfig(ctx context.Context, query contracts.GetConfigQuery) (contracts.GetConfigReply, *problem.Problem) {
	return invokeControl[contracts.GetConfigQuery, contracts.GetConfigReply](ctx, g, g.registry.GetConfig, query, "request configctl get config")
}

func (g *Gateway) GetActiveConfig(ctx context.Context, query contracts.GetActiveConfigQuery) (contracts.GetActiveConfigReply, *problem.Problem) {
	return invokeControl[contracts.GetActiveConfigQuery, contracts.GetActiveConfigReply](ctx, g, g.registry.GetActive, query, "request configctl active config")
}

func (g *Gateway) ListActiveRuntimeProjections(ctx context.Context, query contracts.ListActiveRuntimeProjectionsQuery) (contracts.ListActiveRuntimeProjectionsReply, *problem.Problem) {
	return invokeControl[contracts.ListActiveRuntimeProjectionsQuery, contracts.ListActiveRuntimeProjectionsReply](ctx, g, g.registry.ListActiveRuntimeProjections, query, "request configctl active runtime projections")
}

func (g *Gateway) ListActiveIngestionBindings(ctx context.Context, query contracts.ListActiveIngestionBindingsQuery) (contracts.ListActiveIngestionBindingsReply, *problem.Problem) {
	return invokeControl[contracts.ListActiveIngestionBindingsQuery, contracts.ListActiveIngestionBindingsReply](ctx, g, g.registry.ListActiveIngestionBindings, query, "request configctl active ingestion bindings")
}

func (g *Gateway) ListConfigs(ctx context.Context, query contracts.ListConfigsQuery) (contracts.ListConfigsReply, *problem.Problem) {
	return invokeControl[contracts.ListConfigsQuery, contracts.ListConfigsReply](ctx, g, g.registry.ListConfigs, query, "request configctl list configs")
}

func (g *Gateway) ValidateDraft(ctx context.Context, command contracts.ValidateDraftCommand) (contracts.ValidateDraftReply, *problem.Problem) {
	return invokeControl[contracts.ValidateDraftCommand, contracts.ValidateDraftReply](ctx, g, g.registry.ValidateDraft, command, "request configctl validate draft")
}

func (g *Gateway) ValidateConfig(ctx context.Context, command contracts.ValidateConfigCommand) (contracts.ValidateConfigReply, *problem.Problem) {
	return invokeControl[contracts.ValidateConfigCommand, contracts.ValidateConfigReply](ctx, g, g.registry.ValidateConfig, command, "request configctl validate config")
}

func (g *Gateway) CompileConfig(ctx context.Context, command contracts.CompileConfigCommand) (contracts.CompileConfigReply, *problem.Problem) {
	return invokeControl[contracts.CompileConfigCommand, contracts.CompileConfigReply](ctx, g, g.registry.CompileConfig, command, "request configctl compile config")
}

func (g *Gateway) ActivateConfig(ctx context.Context, command contracts.ActivateConfigCommand) (contracts.ActivateConfigReply, *problem.Problem) {
	return invokeControl[contracts.ActivateConfigCommand, contracts.ActivateConfigReply](ctx, g, g.registry.ActivateConfig, command, "request configctl activate config")
}

func invokeControl[Req any, Res any](ctx context.Context, gateway *Gateway, spec natskit.ControlSpec, payload Req, action string) (Res, *problem.Problem) {
	var zero Res
	if gateway == nil || gateway.client == nil {
		return zero, problem.New(problem.Unavailable, "configctl gateway is unavailable")
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, gateway.source, payload)
	if prob != nil {
		return zero, prob
	}

	replyBytes, err := gateway.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return zero, problem.Wrap(err, problem.Unavailable, action+" failed")
	}

	return natskit.DecodeControlReply[Res](spec, replyBytes)
}
