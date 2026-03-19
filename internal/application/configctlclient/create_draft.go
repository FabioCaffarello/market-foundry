package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type CreateDraftUseCase = usecase.CommandUseCase[contracts.CreateDraftCommand, contracts.CreateDraftReply]

func NewCreateDraftUseCase(gateway ports.ConfigctlGateway) *CreateDraftUseCase {
	return usecase.NewCommand[contracts.CreateDraftCommand, contracts.CreateDraftReply](
		gateway.CreateDraft, "configctl",
	)
}
