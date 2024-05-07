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
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	bip39 "github.com/cosmos/go-bip39"
	"github.com/hyperledger/firefly-cli/internal/docker"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	etherminthd "github.com/tharsis/ethermint/crypto/hd"
)

const (
	defaultEntropySize = 256
)

func CreateWalletFile(ctx context.Context, outputDirectory, prefix, password string) (*secp256k1.KeyPair, error) {
	entropy, err := bip39.NewEntropy(defaultEntropySize)
	if err != nil {
		return nil, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}

	mnemonic = "apology false junior asset sphere puppy upset dirt miracle rice horn spell ring vast wrist crisp snake oak give cement pause swallow barely clever"

	privateKey, err := etherminthd.EthSecp256k1.Derive()(mnemonic, "", "m/44'/118'/0'/0/0")
	if err != nil {
		return nil, err
	}

	keyPair, err := secp256k1.NewSecp256k1KeyPair(privateKey)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return nil, err
	}

	//if err := docker.RunDockerCommand(ctx, outputDirectory, "run", "--rm", "-v", fmt.Sprintf("%s:/root/.gnchain/keyring-test", outputDirectory), gnchaindImage, "sh", "-c", fmt.Sprintf("\"echo '%s' | gnchaind keys add %s --recover --algo eth_secp256k1 --keyring-backend test\"", mnemonic, keyPair.Address.String()[2:])); err != nil {
	if err := docker.RunDockerCommand(ctx, outputDirectory, "run", "--rm", "-v", fmt.Sprintf("%s:/root/.gnchain/keyring-test", outputDirectory), "-e", "GNCHAIND_ALGO=eth_secp256k1", "-e", "GNCHAIND_KEYRING_BACKEND=test", gnchaindImage, "/recover.sh", mnemonic, keyPair.Address.String()[2:]); err != nil {
		return nil, err
	}

	return keyPair, nil
}

func bench32Address(address string) string {
	if bs, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(address), "0x")); err == nil {
		bech32Addr, err := bech32.ConvertAndEncode("gnc", bs)
		if err != nil {
			panic(err)
		}
		return bech32Addr
	}
	return address
}

func hexAddress(address string) string {
	if _, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(address), "0x")); err == nil {
		return address
	}
	_, bz, err := bech32.DecodeAndConvert(address)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("0x%x", bz)
}
