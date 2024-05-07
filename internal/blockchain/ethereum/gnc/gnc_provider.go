// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnc

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	bip39 "github.com/cosmos/go-bip39"
	"github.com/hyperledger/firefly-cli/internal/blockchain/ethereum"
	"github.com/hyperledger/firefly-cli/internal/blockchain/ethereum/connector"
	"github.com/hyperledger/firefly-cli/internal/blockchain/ethereum/connector/ethconnect"
	"github.com/hyperledger/firefly-cli/internal/blockchain/ethereum/connector/evmconnect"
	"github.com/hyperledger/firefly-cli/internal/constants"
	"github.com/hyperledger/firefly-cli/internal/docker"
	"github.com/hyperledger/firefly-cli/internal/log"
	"github.com/hyperledger/firefly-cli/pkg/types"
)

const (
	gnchaindImage         = "glodnet/gnchaind"
	blockchainServiceName = "gnchain"
)

// TODO: Probably randomize this and make it different per member?
var keyPassword = "correcthorsebatterystaple"

type GncProvider struct {
	ctx       context.Context
	stack     *types.Stack
	connector connector.Connector
}

func NewGncProvider(ctx context.Context, stack *types.Stack) *GncProvider {
	var connector connector.Connector
	switch stack.BlockchainConnector {
	case types.BlockchainConnectorEthconnect:
		connector = ethconnect.NewEthconnect(ctx)
	case types.BlockchainConnectorEvmconnect:
		connector = evmconnect.NewEvmconnect(ctx)
	}

	return &GncProvider{
		ctx:       ctx,
		stack:     stack,
		connector: connector,
	}
}

func (p *GncProvider) WriteConfig(options *types.InitOptions) error {
	initDir := filepath.Join(constants.StacksDir, p.stack.Name, "init")
	for i, member := range p.stack.Members {
		// Generate the connector config for each member
		connectorConfigPath := filepath.Join(initDir, "config", fmt.Sprintf("%s_%v.yaml", p.connector.Name(), i))
		if err := p.connector.GenerateConfig(p.stack, member, blockchainServiceName).WriteConfig(connectorConfigPath, options.ExtraConnectorConfigPath); err != nil {
			return nil
		}
	}

	// Create chain.yml
	addresses := ""
	for i, member := range p.stack.Members {
		address := member.Account.(*ethereum.Account).Address
		// Drop the 0x on the front of the address here because that's what geth is expecting in the genesis.json
		if i != 0 {
			addresses += "\n"
		}
		addresses += fmt.Sprintf(`  - address: %s
    roles: ["ROOT_ADMIN"]
    coins: ["10000000000000000000gnc"]`, bench32Address(address))
	}
	blockPeriod := fmt.Sprintf("%ds", options.BlockPeriod)
	chainID := fmt.Sprintf("gnchain_45-%d", options.ChainID)

	chainYaml := chainYAML
	chainYaml = strings.ReplaceAll(chainYaml, "{VAR_ACCOUNTS}", addresses)
	chainYaml = strings.ReplaceAll(chainYaml, "{VAR_CHAINID}", chainID)
	chainYaml = strings.ReplaceAll(chainYaml, "{VAR_BLOCK_PERIOD}", blockPeriod)
	if err := ioutil.WriteFile(filepath.Join(initDir, "blockchain", "chain.yml"), []byte(chainYaml), 0755); err != nil {
		return err
	}
	return nil
}

func (p *GncProvider) FirstTimeSetup() error {
	blockchainVolumeName := fmt.Sprintf("%s_%s", p.stack.Name, blockchainServiceName)
	blockchainDir := path.Join(p.stack.RuntimeDir, "blockchain")
	contractsDir := path.Join(p.stack.RuntimeDir, "contracts")

	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		return err
	}

	for i := range p.stack.Members {
		// Copy connector config to each member's volume
		connectorConfigPath := filepath.Join(p.stack.StackDir, "runtime", "config", fmt.Sprintf("%s_%v.yaml", p.connector.Name(), i))
		connectorConfigVolumeName := fmt.Sprintf("%s_%s_config_%v", p.stack.Name, p.connector.Name(), i)
		docker.CopyFileToVolume(p.ctx, connectorConfigVolumeName, connectorConfigPath, "config.yaml")
	}

	// Copy the wallet files all members to the blockchain volume
	keystoreDirectory := filepath.Join(blockchainDir, "keyring-test")
	if err := docker.CopyFileToVolume(p.ctx, blockchainVolumeName, keystoreDirectory, "/"); err != nil {
		return err
	}

	//// Copy the genesis block information
	//if err := docker.CopyFileToVolume(p.ctx, blockchainVolumeName, path.Join(blockchainDir, "chain.yml"), "chain.yml"); err != nil {
	//	return err
	//}
	//
	//// Initialize the genesis block
	//if err := docker.RunDockerCommand(p.ctx, p.stack.StackDir, "run", "--rm", "-v", fmt.Sprintf("%s:/root/.gnchain", blockchainVolumeName), gnchaindImage, "gnchaind", "chain", "/root/.gnchain/chain.yml"); err != nil {
	//	return err
	//}

	return nil
}

func (p *GncProvider) PreStart() error {
	return nil
}

func (p *GncProvider) PostStart(firstTimeSetup bool) error {
	//l := log.LoggerFromContext(p.ctx)
	//// Unlock accounts
	//for _, account := range p.stack.State.Accounts {
	//	address := account.(*ethereum.Account).Address
	//	l.Info(fmt.Sprintf("unlocking account %s", address))
	//	if err := p.unlockAccount(address, keyPassword); err != nil {
	//		return err
	//	}
	//}

	return nil
}

func (p *GncProvider) unlockAccount(address, password string) error {
	address = hexAddress(address)
	l := log.LoggerFromContext(p.ctx)
	verbose := log.VerbosityFromContext(p.ctx)
	gethClient := NewGethClient(fmt.Sprintf("http://127.0.0.1:%v", p.stack.ExposedBlockchainPort))
	retries := 10
	for {
		if err := gethClient.UnlockAccount(address, password); err != nil {
			if verbose {
				l.Debug(err.Error())
			}
			if retries == 0 {
				return fmt.Errorf("unable to unlock account %s", address)
			}
			time.Sleep(time.Second * 1)
			retries--
		} else {
			break
		}
	}
	return nil
}

func (p *GncProvider) DeployFireFlyContract() (*types.ContractDeploymentResult, error) {
	contract, err := ethereum.ReadFireFlyContract(p.ctx, p.stack)
	if err != nil {
		return nil, err
	}
	return p.connector.DeployContract(contract, "FireFly", p.stack.Members[0], nil)
}

func (p *GncProvider) GetDockerServiceDefinitions(bootnodes string) []*docker.ServiceDefinition {
	mnemonic := ""
	if len(bootnodes) > 0 {
		entropy, _ := bip39.NewEntropy(defaultEntropySize)
		mnemonic, _ = bip39.NewMnemonic(entropy)
	}

	serviceDefinitions := make([]*docker.ServiceDefinition, 1)
	serviceDefinitions[0] = &docker.ServiceDefinition{
		ServiceName: blockchainServiceName,
		Service: &docker.Service{
			Image:         gnchaindImage,
			ContainerName: fmt.Sprintf("%s_%s", p.stack.Name, blockchainServiceName),
			Environment: map[string]interface{}{
				"GNCHAIND_LOG_FORMAT":           "json",
				"GNCHAIND_LOG_LEVEL":            "debug",
				"GNCHAIND_KEYRING_BACKEND":      "test",
				"GNCHAIND_P2P_PERSISTENT_PEERS": bootnodes,
				"RECOVER":                       mnemonic,
			},
			Command: "sh -c '/wait && /init.sh && /run.sh'",
			Volumes: []string{
				fmt.Sprintf("./runtime/blockchain/chain.yml:/root/chain.yml"),
				blockchainServiceName + ":/root/.gnchain",
			},
			Logging: docker.StandardLogOptions,
			Ports: []string{
				fmt.Sprintf("%d:26656/tcp", p.stack.ExposedBlockchainP2P),
				fmt.Sprintf("%d:26656/udp", p.stack.ExposedBlockchainP2P),
				fmt.Sprintf("%d:8545", p.stack.ExposedBlockchainPort),
			},
		},
		VolumeNames: []string{blockchainServiceName},
	}
	serviceDefinitions = append(serviceDefinitions, p.connector.GetServiceDefinitions(p.stack, map[string]string{blockchainServiceName: "service_started"})...)
	return serviceDefinitions
}

func (p *GncProvider) GetBlockchainPluginConfig(stack *types.Stack, m *types.Organization) (blockchainConfig *types.BlockchainConfig) {
	var connectorURL string
	if m.External {
		connectorURL = p.GetConnectorExternalURL(m)
	} else {
		connectorURL = p.GetConnectorURL(m)
	}

	blockchainConfig = &types.BlockchainConfig{
		Type: "ethereum",
		Ethereum: &types.EthereumConfig{
			Ethconnect: &types.EthconnectConfig{
				URL:   connectorURL,
				Topic: m.ID,
			},
		},
	}
	return
}

func (p *GncProvider) GetOrgConfig(stack *types.Stack, m *types.Organization) (orgConfig *types.OrgConfig) {
	account := m.Account.(*ethereum.Account)
	orgConfig = &types.OrgConfig{
		Name: m.OrgName,
		Key:  account.Address,
	}
	return
}

func (p *GncProvider) Reset() error {
	return nil
}

func (p *GncProvider) GetContracts(filename string, extraArgs []string) ([]string, error) {
	contracts, err := ethereum.ReadContractJSON(filename)
	if err != nil {
		return []string{}, err
	}
	contractNames := make([]string, len(contracts.Contracts))
	i := 0
	for contractName := range contracts.Contracts {
		contractNames[i] = contractName
		i++
	}
	return contractNames, err
}

func (p *GncProvider) DeployContract(filename, contractName, instanceName string, member *types.Organization, extraArgs []string) (*types.ContractDeploymentResult, error) {
	contracts, err := ethereum.ReadContractJSON(filename)
	if err != nil {
		return nil, err
	}
	return p.connector.DeployContract(contracts.Contracts[contractName], instanceName, member, extraArgs)
}

func (p *GncProvider) CreateAccount(args []string) (interface{}, error) {
	blockchainVolumeName := fmt.Sprintf("%s_%s", p.stack.Name, blockchainServiceName)
	var directory string
	stackHasRunBefore, err := p.stack.HasRunBefore()
	if err != nil {
		return nil, err
	}
	if stackHasRunBefore {
		directory = p.stack.RuntimeDir
	} else {
		directory = p.stack.InitDir
	}

	prefix := strconv.FormatInt(time.Now().UnixNano(), 10)
	outputDirectory := filepath.Join(directory, "blockchain", "keyring-test")
	keyPair, err := CreateWalletFile(p.ctx, outputDirectory, prefix, keyPassword)
	if err != nil {
		return nil, err
	}

	if stackHasRunBefore {
		// Copy the wallet files all members to the blockchain volume
		if err := docker.CopyFileToVolume(p.ctx, blockchainVolumeName, outputDirectory, "/"); err != nil {
			return nil, err
		}
		//if err := p.unlockAccount(keyPair.Address.String(), keyPassword); err != nil {
		//	return nil, err
		//}
	}

	return &ethereum.Account{
		Address: keyPair.Address.String(),
		// Address:    bench32Address(keyPair.Address.String()),
		PrivateKey: hex.EncodeToString(keyPair.PrivateKey.Serialize()),
	}, nil
}

func (p *GncProvider) ParseAccount(account interface{}) interface{} {
	accountMap := account.(map[string]interface{})
	return &ethereum.Account{
		Address:    accountMap["address"].(string),
		PrivateKey: accountMap["privateKey"].(string),
	}
}

func (p *GncProvider) GetConnectorName() string {
	return p.connector.Name()
}

func (p *GncProvider) GetConnectorURL(org *types.Organization) string {
	return fmt.Sprintf("http://%s_%s:%v", p.connector.Name(), org.ID, p.connector.Port())
}

func (p *GncProvider) GetConnectorExternalURL(org *types.Organization) string {
	return fmt.Sprintf("http://127.0.0.1:%v", org.ExposedConnectorPort)
}
