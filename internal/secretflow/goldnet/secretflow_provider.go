package goldnet

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly-cli/internal/core"
	"github.com/hyperledger/firefly-cli/internal/docker"
	"github.com/hyperledger/firefly-cli/internal/log"
	"github.com/hyperledger/firefly-cli/internal/secretflow"
	"github.com/hyperledger/firefly-cli/pkg/types"
)

const providerName = "goldnet"

type SecretFlowProvider struct {
	ctx   context.Context
	stack *types.Stack
}

func NewSecretFlowProvider(ctx context.Context, stack *types.Stack) *SecretFlowProvider {
	return &SecretFlowProvider{
		ctx:   ctx,
		stack: stack,
	}
}

func (p *SecretFlowProvider) FirstTimeSetup(idx int) error {
	l := log.LoggerFromContext(p.ctx)
	for _, member := range p.stack.Members {
		l.Info(fmt.Sprintf("registering secretflow on member %s", member.ID))
		connectorName := fmt.Sprintf("secretflows-%v-%v", member.ID, idx)
		body := map[string]interface{}{
			"address":     connectorName,
			"description": fmt.Sprintf("%s's training node", member.OrgName),
			"head":        p.stack.RayHeadAddress,
			"name":        fmt.Sprintf("%s-%s", member.OrgName, member.NodeName),
			"port":        9090, //member.ExposedSecretFlowsPorts[idx],
			"range_end":   member.ExposedSecretFlowsPorts[idx] + secretflow.RangeInterval,
			"range_start": member.ExposedSecretFlowsPorts[idx] + 1,
		}
		nodeUrl := fmt.Sprintf("http://localhost:%d/api/v1/secretflow/nodes", member.ExposedFireflyPort)
		if err := core.RequestWithRetry(p.ctx, "POST", nodeUrl, body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (p *SecretFlowProvider) GetDockerServiceDefinitions(idx int) []*docker.ServiceDefinition {
	serviceDefinitions := make([]*docker.ServiceDefinition, 0, len(p.stack.Members))
	dependsOn := map[string]map[string]string{}
	if p.stack.RayHeadAddress == "secretflow_head:6379" {
		connectorName := "secretflow_head"
		serviceDefinitions = make([]*docker.ServiceDefinition, 0, len(p.stack.Members)+1)
		dependsOn = map[string]map[string]string{
			connectorName: {
				"condition": "service_started",
			},
		}
		serviceDefinitions = append(serviceDefinitions, &docker.ServiceDefinition{
			ServiceName: connectorName,
			Service: &docker.Service{
				Image:         p.stack.VersionManifest.SecretFlowGoldNet.GetDockerImageString(),
				ContainerName: fmt.Sprintf("%s_%s", p.stack.Name, connectorName),
				Environment: map[string]interface{}{
					"node_alias": "head",
				},
				Volumes: []string{
					fmt.Sprintf("%s_shm:/dev/shm", connectorName),
				},
				Logging: docker.StandardLogOptions,
			},
			VolumeNames: []string{
				fmt.Sprintf("%s_shm", connectorName),
			},
		})
	}

	for i, member := range p.stack.Members {
		connectorName := fmt.Sprintf("secretflows-%v-%v", member.ID, idx)

		env := map[string]interface{}{
			"Env":             "test",
			"DEPLOY_METHOD":   "integration",
			"DATABASE_ENGINE": "django.db.backends.sqlite3",
			"DATABASE_NAME":   "/root/nueva/sqlite3.db",
			"ADDRESS":         p.stack.RayHeadAddress,
			"node_alias":      fmt.Sprintf("%s-%s", member.OrgName, member.NodeName),
		}

		var ports []string
		for j := 0; j < secretflow.RangeInterval; j++ {
			port := member.ExposedSecretFlowsPorts[idx] + j + 1
			ports = append(ports, fmt.Sprintf("%d:%d", port, port))
		}
		serviceDefinitions = append(serviceDefinitions, &docker.ServiceDefinition{
			ServiceName: connectorName,
			Service: &docker.Service{
				Image:         p.stack.VersionManifest.SecretFlowGoldNet.GetDockerImageString(),
				ContainerName: fmt.Sprintf("%s_secretflow_%v_%v", p.stack.Name, i, idx),
				Ports:         append(ports, []string{fmt.Sprintf("%d:9090", member.ExposedSecretFlowsPorts[idx])}...),
				Volumes: []string{
					fmt.Sprintf("secretflow_%v_%v:/root/nueva", i, idx),
					fmt.Sprintf("secretflow_%v_%v_shm:/dev/shm", i, idx),
				},
				Environment: env,
				DependsOn:   dependsOn,
				HealthCheck: &docker.HealthCheck{
					Test: []string{"CMD", "curl", "http://localhost:9090/api/system/user/index/"},
				},
				Logging: docker.StandardLogOptions,
			},
			VolumeNames: []string{
				fmt.Sprintf("secretflow_%v_%v", i, idx),
				fmt.Sprintf("secretflow_%v_%v_shm", i, idx),
			},
		})
	}
	return serviceDefinitions
}

func (p *SecretFlowProvider) GetFireflyConfig(m *types.Organization, idx int) *types.SecretFlowsConfig {
	name := providerName
	if idx > 0 {
		name = fmt.Sprintf("%s_%d", providerName, idx)
	}
	return &types.SecretFlowsConfig{
		Type: "goldnet",
		Name: name,
		GNSecretFlow: &types.GNSecretFlowConfig{
			URL: p.getURL(m, idx),
		},
	}
}

func (p *SecretFlowProvider) getURL(member *types.Organization, idx int) string {
	if !member.External {
		return fmt.Sprintf("http://secretflows-%v-%d:9090", member.ID, idx)
	} else {
		return fmt.Sprintf("http://127.0.0.1:%v", member.ExposedSecretFlowsPorts[idx])
	}
}

func (p *SecretFlowProvider) GetName() string {
	return providerName
}
