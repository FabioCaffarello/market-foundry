package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type ListActiveIngestionBindingsUseCase = usecase.GatewayUseCase[contracts.ListActiveIngestionBindingsQuery, contracts.ListActiveIngestionBindingsReply]

func NewListActiveIngestionBindingsUseCase(gateway ports.ConfigctlGateway) *ListActiveIngestionBindingsUseCase {
	return usecase.NewGateway[contracts.ListActiveIngestionBindingsQuery, contracts.ListActiveIngestionBindingsReply](
		gateway.ListActiveIngestionBindings, "configctl",
	)
}
