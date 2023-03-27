package secretflow

import (
	"github.com/hyperledger/firefly-cli/internal/docker"
	"github.com/hyperledger/firefly-cli/pkg/types"
)

const RangeInterval = 10

type ISecretFlowsProvider interface {
	FirstTimeSetup(idx int) error
	GetDockerServiceDefinitions(idx int) []*docker.ServiceDefinition
	GetFireflyConfig(m *types.Organization, idx int) *types.SecretFlowsConfig
	GetName() string
}
