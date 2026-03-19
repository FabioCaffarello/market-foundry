package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type ValidateDraftUseCase = usecase.CommandUseCase[contracts.ValidateDraftCommand, contracts.ValidateDraftReply]

func NewValidateDraftUseCase(gateway ports.ConfigctlGateway) *ValidateDraftUseCase {
	return usecase.NewCommand[contracts.ValidateDraftCommand, contracts.ValidateDraftReply](
		gateway.ValidateDraft, "configctl",
	)
}
