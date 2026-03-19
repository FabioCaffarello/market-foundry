package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type ValidateConfigUseCase = usecase.CommandUseCase[contracts.ValidateConfigCommand, contracts.ValidateConfigReply]

func NewValidateConfigUseCase(gateway ports.ConfigctlGateway) *ValidateConfigUseCase {
	return usecase.NewCommand[contracts.ValidateConfigCommand, contracts.ValidateConfigReply](
		gateway.ValidateConfig, "configctl",
	)
}
