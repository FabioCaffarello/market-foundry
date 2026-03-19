package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type ListConfigsUseCase = usecase.GatewayUseCase[contracts.ListConfigsQuery, contracts.ListConfigsReply]

func NewListConfigsUseCase(gateway ports.ConfigctlGateway) *ListConfigsUseCase {
	return usecase.NewGateway[contracts.ListConfigsQuery, contracts.ListConfigsReply](
		gateway.ListConfigs, "configctl",
	)
}
