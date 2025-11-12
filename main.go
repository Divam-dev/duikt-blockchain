package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

func account() (*ecdsa.PrivateKey, common.Address, string) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hexutil.Encode(privateKeyBytes)[2:]

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return privateKey, address, privateKeyHex
}

func getBalance(client *ethclient.Client, address string) {
	add := common.HexToAddress(address)
	ctx := context.Background()
	balance, err := client.BalanceAt(ctx, add, nil)
	if err != nil {
		log.Fatal(err)
	}

	etherValue := new(big.Float).Quo(
		new(big.Float).SetInt(balance),
		big.NewFloat(1e18),
	)
	fmt.Println("Баланс:", etherValue, "ETH")
}

func signMessage(privateKey *ecdsa.PrivateKey, message string) ([]byte, error) {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(message))
	messageHash := hash.Sum(nil)

	signature, err := crypto.Sign(messageHash, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func verifySignature(message string, signature []byte, expectedAddress common.Address) (bool, error) {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(message))
	messageHash := hash.Sum(nil)

	sigPublicKey, err := crypto.SigToPub(messageHash, signature)
	if err != nil {
		return false, err
	}

	recoveredAddress := crypto.PubkeyToAddress(*sigPublicKey)
	return recoveredAddress == expectedAddress, nil
}

func sendTransaction(client *ethclient.Client, privateKeyHex string, toAddress string, valueInWei *big.Int) (string, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("error converting private key to ECDSA: %w", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("error getting nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("error getting gas price: %w", err)
	}

	gasLimit := uint64(21000)
	toAddr := common.HexToAddress(toAddress)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &toAddr,
		Value:    valueInWei,
	})

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("error getting chain ID: %w", err)
	}

	signedTX, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("error signing transaction: %w", err)
	}

	err = client.SendTransaction(context.Background(), signedTX)
	if err != nil {
		return "", fmt.Errorf("error sending transaction: %w", err)
	}

	return signedTX.Hash().Hex(), nil
}

func main() {
	client, err := ethclient.Dial("https://ethereum-sepolia-rpc.publicnode.com")
	if err != nil {
		log.Fatal(err)
	}

	acc := flag.Bool("acc", false, "Створити новий акаунт")
	balance := flag.String("balance", "", "Перевірити баланс акаунта <address>")
	send := flag.Bool("send", false, "Відправити ETH")
	to := flag.String("to", "", "Адреса отримувача")
	amount := flag.Float64("amount", 0, "Сума ETH (тільки для --send)")
	key := flag.String("key", "", "Приватний ключ (hex) для --send або --sign")
	sign := flag.String("sign", "", "Підписати повідомлення <message>")
	verify := flag.String("verify", "", "Перевірити підпис <message>")
	sig := flag.String("sig", "", "Підпис (hex) для перевірки")
	pubkey := flag.String("pubkey", "", "Публічний ключ (hex) для перевірки підпису")

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println("  --acc")
		fmt.Println("        Створити новий акаунт")
		fmt.Println("  --balance <address>")
		fmt.Println("        Перевірити баланс акаунта (у ETH)")
		fmt.Println("  --send --to <address> --amount <ETH> --key <privkey>")
		fmt.Println("        Відправити ETH")
		fmt.Println("  --sign <message> --key <privkey>")
		fmt.Println("        Підписати повідомлення")
		fmt.Println("  --verify <message> --sig <signature> --pubkey <pubkey>")
		fmt.Println("        Перевірити підпис")
	}
	flag.Parse()

	if *acc {
		_, address, privHex := account()
		fmt.Println("Адреса акаунта:", address.Hex())
		fmt.Println("Приватний ключ:", privHex)
		os.Exit(0)
	}

	if *balance != "" {
		getBalance(client, *balance)
		os.Exit(0)
	}

	if *send {
		if *to == "" || *key == "" || *amount <= 0 {
			fmt.Println("Помилка: Для відправки вкажіть --to, --amount та --key")
			os.Exit(1)
		}
		wei := new(big.Int)
		new(big.Float).SetFloat64(*amount).Mul(new(big.Float).SetFloat64(*amount), big.NewFloat(1e18)).Int(wei)
		txHash, _ := sendTransaction(client, *key, *to, wei)
		fmt.Println("Транзакція надіслана. Хеш:", txHash)
		os.Exit(0)
	}

	if *sign != "" && *key != "" {
		privKey, _ := crypto.HexToECDSA(*key)
		sigBytes, _ := signMessage(privKey, *sign)
		fmt.Printf("Підпис: %x\n", sigBytes)
		os.Exit(0)
	}

	if *verify != "" && *sig != "" && *pubkey != "" {
		expectedAddress := common.HexToAddress(*pubkey)
		sigBytes := common.FromHex(*sig)

		valid, err := verifySignature(*verify, sigBytes, expectedAddress)
		if err != nil {
			fmt.Println("Помилка перевірки підпису:", err)
			os.Exit(1)
		}

		if valid {
			fmt.Println("Підпис валідний")
		} else {
			fmt.Println("Підпис не валідний")
		}
		os.Exit(0)
	}

	flag.Usage()
}
