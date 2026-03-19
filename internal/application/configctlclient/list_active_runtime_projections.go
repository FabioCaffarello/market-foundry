package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type ListActiveRuntimeProjectionsUseCase = usecase.CommandUseCase[contracts.ListActiveRuntimeProjectionsQuery, contracts.ListActiveRuntimeProjectionsReply]

func NewListActiveRuntimeProjectionsUseCase(gateway ports.ConfigctlGateway) *ListActiveRuntimeProjectionsUseCase {
	return usecase.NewCommand[contracts.ListActiveRuntimeProjectionsQuery, contracts.ListActiveRuntimeProjectionsReply](
		gateway.ListActiveRuntimeProjections, "configctl",
	)
}
