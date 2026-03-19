package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type GetActiveConfigUseCase = usecase.GatewayUseCase[contracts.GetActiveConfigQuery, contracts.GetActiveConfigReply]

func NewGetActiveConfigUseCase(gateway ports.ConfigctlGateway) *GetActiveConfigUseCase {
	return usecase.NewGateway[contracts.GetActiveConfigQuery, contracts.GetActiveConfigReply](
		gateway.GetActiveConfig, "configctl",
	)
}
