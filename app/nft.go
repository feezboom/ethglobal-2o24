package app

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

var currentNftId *big.Int

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

	// Update the ABI to include the tokenID return type
	contractAbi, err := abi.JSON(strings.NewReader(`[{"constant": false, "inputs": [{"name": "to", "type": "address"}, {"name": "tokenId", "type": "uint256"}], "name": "mint", "outputs": [], "type": "function"}]`))
	if err != nil {
		return NFT{}, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	receiverAddress := common.HexToAddress(req.Receiver)

	newNftId, err := generateTokenId()
	if err != nil {
		return NFT{}, fmt.Errorf("failed to generate NFT ID: %v", err)
	}

	println("newNftId=" + newNftId.String())

	data, err := contractAbi.Pack("mint", receiverAddress, newNftId)
	if err != nil {
		return NFT{}, fmt.Errorf("failed to pack data: %v", err)
	}

	toAddress := common.HexToAddress(contractAddressHex)
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

	// Update the current NFT ID in the database
	if err = updateCurrentNftIdInDB(currentNftId); err != nil {
		return NFT{}, fmt.Errorf("failed to update NFT ID in database: %v", err)
	}

	// Decode the returned tokenID from the logs
	nft := NFT{
		TokenID:  currentNftId.String(),
		Contract: contractAddressHex,
	}

	return nft, nil
}

func updateCurrentNftIdInDB(tokenId *big.Int) error {
	_, err := nftIdCollection.UpdateOne(
		context.Background(),
		bson.D{},
		bson.D{{"$set", bson.D{{"tokenId", tokenId.Uint64()}}}},
		options.Update().SetUpsert(true),
	)
	return err
}

func generateTokenId() (*big.Int, error) {
	if currentNftId == nil {
		var result struct {
			TokenID uint64 `bson:"tokenId"`
		}
		opts := options.FindOne().SetSort(bson.D{{"tokenId", -1}})
		err := nftIdCollection.FindOne(context.Background(), bson.D{}, opts).Decode(&result)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			currentNftId = big.NewInt(0)
		} else {
			currentNftId = big.NewInt(int64(result.TokenID + 1))
		}
	} else {
		currentNftId = currentNftId.Add(currentNftId, big.NewInt(1))
	}

	return currentNftId, nil
}
