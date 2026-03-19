package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type ActivateConfigUseCase = usecase.CommandUseCase[contracts.ActivateConfigCommand, contracts.ActivateConfigReply]

func NewActivateConfigUseCase(gateway ports.ConfigctlGateway) *ActivateConfigUseCase {
	return usecase.NewCommand[contracts.ActivateConfigCommand, contracts.ActivateConfigReply](
		gateway.ActivateConfig, "configctl",
	)
}
