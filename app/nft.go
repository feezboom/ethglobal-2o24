package app

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type NFT struct {
	TokenID  string `json:"id,omitempty" bson:"id,omitempty"`
	Contract string `json:"contract,omitempty" bson:"contract,omitempty"`
}

func mintNft(req SubmitQuestionRequest) (NFT, error) {
	privateKeyHex := os.Getenv("TECHNICAL_WALLET_PRIVATE_KEY")
	contractAddressHex := os.Getenv("NFT_CONTRACT_ADDRESS")
	rpcURL := os.Getenv("BLOCKCHAIN_RPC_URL")

	if privateKeyHex == "" || contractAddressHex == "" || rpcURL == "" {
		return NFT{}, fmt.Errorf("required environment variables are not set")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return NFT{}, fmt.Errorf("invalid private key: %v", err)
	}

	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to connect to the Ethereum ethClient: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return NFT{}, fmt.Errorf("invalid public key")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to get nonce: %v", err)
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return NFT{}, fmt.Errorf("failed to get gas price: %v", err)
	}

	toAddress := common.HexToAddress(contractAddressHex)
	receiverAddress := common.HexToAddress(req.Receiver)

	// Update the ABI to include the tokenID return type
	contractAbi, err := abi.JSON(strings.NewReader(`[{"constant": false, "inputs": [{"name": "receiver", "type": "address"}], "name": "mintForAddress", "outputs": [{"name": "", "type": "uint256"}], "type": "function"}]`))
	if err != nil {
		return NFT{}, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	data, err := contractAbi.Pack("mintForAddress", receiverAddress)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to pack data: %v", err)
	}

	tx := types.NewTransaction(nonce, toAddress, big.NewInt(0), uint64(3000000), gasPrice, data)

	chainID, err := ethClient.NetworkID(context.Background())
	if err != nil {
		return NFT{}, fmt.Errorf("failed to get network ID: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to send transaction: %v", err)
	}

	receipt, err := bind.WaitMined(context.Background(), ethClient, signedTx)
	if err != nil {
		return NFT{}, fmt.Errorf("transaction mining failed: %v", err)
	}

	if receipt.Status != 1 {
		return NFT{}, fmt.Errorf("transaction failed")
	}

	// Decode the returned tokenID from the logs
	var tokenID *big.Int
	for _, vLog := range receipt.Logs {
		if vLog.Address == toAddress {
			eventAbi, err := abi.JSON(strings.NewReader(`[{"anonymous":false,"inputs":[{"indexed":true,"name":"receiver","type":"address"},{"indexed":false,"name":"tokenID","type":"uint256"}],"name":"MintForAddress","type":"event"}]`))
			if err != nil {
				return NFT{}, fmt.Errorf("failed to parse event ABI: %v", err)
			}

			err = eventAbi.UnpackIntoInterface(&tokenID, "MintForAddress", vLog.Data)
			if err != nil {
				return NFT{}, fmt.Errorf("failed to unpack log data: %v", err)
			}
			break
		}
	}

	if tokenID == nil {
		return NFT{}, fmt.Errorf("failed to get tokenID from transaction receipt")
	}

	nft := NFT{
		TokenID:  tokenID.String(),
		Contract: contractAddressHex,
	}

	return nft, nil
}
