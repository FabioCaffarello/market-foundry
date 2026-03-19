package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type GetConfigUseCase = usecase.CommandUseCase[contracts.GetConfigQuery, contracts.GetConfigReply]

func NewGetConfigUseCase(gateway ports.ConfigctlGateway) *GetConfigUseCase {
	return usecase.NewCommand[contracts.GetConfigQuery, contracts.GetConfigReply](
		gateway.GetConfig, "configctl",
	)
}
