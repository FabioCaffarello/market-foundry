package configctlclient

import (
	"internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/shared/usecase"
)

type CompileConfigUseCase = usecase.CommandUseCase[contracts.CompileConfigCommand, contracts.CompileConfigReply]

func NewCompileConfigUseCase(gateway ports.ConfigctlGateway) *CompileConfigUseCase {
	return usecase.NewCommand[contracts.CompileConfigCommand, contracts.CompileConfigReply](
		gateway.CompileConfig, "configctl",
	)
}
