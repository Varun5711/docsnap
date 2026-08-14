package flare

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client interface {
	Anchor(req model.AnchorRequest) (model.AnchorResult, error)

	BuildSubmitCalldata(req model.AnchorRequest) (model.SubmitCalldata, error)

	VerifySubmission(ctx context.Context, txHash string, req model.AnchorRequest) (model.AnchorResult, string, error)
}

type Config struct {
	Mode                  string
	RPCURL                string
	ContractAddress       string
	PrivateKey            string
	TEEReporterPrivateKey string
}

func NewClient(ctx context.Context, cfg Config) (Client, error) {
	if cfg.Mode == "" || cfg.Mode == "simulated" {
		return NewSimulatedClient(), nil
	}
	if cfg.Mode != "coston2" {
		return nil, errors.New("unsupported flare mode")
	}
	return NewCoston2Client(ctx, cfg)
}

type SimulatedClient struct{}

func NewSimulatedClient() SimulatedClient {
	return SimulatedClient{}
}

func (SimulatedClient) Anchor(req model.AnchorRequest) (model.AnchorResult, error) {
	seed := req.EvidenceID + "|" + req.EvidenceCommitment + "|" + req.ClaimsRoot + "|" + time.Now().UTC().Format(time.RFC3339Nano)
	certificateHash := req.TEECertificateHash
	if certificateHash == "" {
		certificateHash = digest(seed + "|tee-cert")
	}
	if !strings.HasPrefix(certificateHash, "0x") {
		certificateHash = "0x" + certificateHash
	}
	return model.AnchorResult{
		TxHash:             "0x" + digest(seed+"|tx"),
		TEECertificateHash: certificateHash,
		Status:             "certified",
	}, nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (SimulatedClient) BuildSubmitCalldata(req model.AnchorRequest) (model.SubmitCalldata, error) {
	return model.SubmitCalldata{To: "0x0000000000000000000000000000000000000000", Data: "0x", ChainID: 114}, nil
}

func (c SimulatedClient) VerifySubmission(ctx context.Context, txHash string, req model.AnchorRequest) (model.AnchorResult, string, error) {
	result, err := c.Anchor(req)
	return result, "0x0000000000000000000000000000000000000000", err
}

type Coston2Client struct {
	rpc         *ethclient.Client
	contract    common.Address
	submitter   *signer
	reporter    *signer
	contractABI abi.ABI
}

type signer struct {
	privateKey string
	address    common.Address
	mu         sync.Mutex
}

const docsnapAnchorABI = `[
	{"type":"function","name":"submitEvidence","inputs":[{"name":"evidenceId","type":"bytes32"},{"name":"evidenceCommitment","type":"bytes32"},{"name":"screenshotHash","type":"bytes32"},{"name":"scrapedTextHash","type":"bytes32"},{"name":"metadataCommitment","type":"bytes32"},{"name":"claimsRoot","type":"bytes32"}],"outputs":[],"stateMutability":"nonpayable"},
	{"type":"function","name":"recordTEECertificate","inputs":[{"name":"evidenceId","type":"bytes32"},{"name":"teeCertificateHash","type":"bytes32"},{"name":"accepted","type":"bool"}],"outputs":[],"stateMutability":"nonpayable"}
]`

func NewCoston2Client(ctx context.Context, cfg Config) (*Coston2Client, error) {
	if cfg.RPCURL == "" || cfg.ContractAddress == "" || cfg.PrivateKey == "" {
		return nil, errors.New("coston2 mode requires rpc url, contract address, and private key")
	}
	rpc, err := ethclient.DialContext(ctx, cfg.RPCURL)
	if err != nil {
		return nil, err
	}
	chainID, err := rpc.ChainID(ctx)
	if err != nil {
		rpc.Close()
		return nil, err
	}
	if chainID.Cmp(big.NewInt(114)) != 0 {
		rpc.Close()
		return nil, errors.New("rpc is not Coston2 chain id 114")
	}

	submitter, err := newSigner(cfg.PrivateKey)
	if err != nil {
		rpc.Close()
		return nil, err
	}

	var reporter *signer
	if cfg.TEEReporterPrivateKey != "" {
		parsed, err := newSigner(cfg.TEEReporterPrivateKey)
		if err != nil {
			rpc.Close()
			return nil, err
		}
		reporter = parsed
	}

	parsedABI, err := abi.JSON(strings.NewReader(docsnapAnchorABI))
	if err != nil {
		rpc.Close()
		return nil, err
	}

	return &Coston2Client{
		rpc:         rpc,
		contract:    common.HexToAddress(cfg.ContractAddress),
		submitter:   submitter,
		reporter:    reporter,
		contractABI: parsedABI,
	}, nil
}

func (c *Coston2Client) Anchor(req model.AnchorRequest) (model.AnchorResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	submitData, err := c.buildSubmitData(req)
	if err != nil {
		return model.AnchorResult{}, err
	}

	submitTx, err := c.send(ctx, c.submitter, submitData)
	if err != nil {
		return model.AnchorResult{}, err
	}

	return c.certify(ctx, req, submitTx.Hash().Hex())
}

func (c *Coston2Client) BuildSubmitCalldata(req model.AnchorRequest) (model.SubmitCalldata, error) {
	data, err := c.buildSubmitData(req)
	if err != nil {
		return model.SubmitCalldata{}, err
	}
	return model.SubmitCalldata{To: c.contract.Hex(), Data: "0x" + hex.EncodeToString(data), ChainID: 114}, nil
}

func (c *Coston2Client) VerifySubmission(ctx context.Context, txHash string, req model.AnchorRequest) (model.AnchorResult, string, error) {
	hash := common.HexToHash(txHash)

	tx, submitter, err := c.waitMined(ctx, hash)
	if err != nil {
		return model.AnchorResult{}, "", err
	}
	if tx.To() == nil || *tx.To() != c.contract {
		return model.AnchorResult{}, "", errors.New("transaction was not sent to the DocSnap contract")
	}
	wantData, err := c.buildSubmitData(req)
	if err != nil {
		return model.AnchorResult{}, "", err
	}
	if !bytes.Equal(tx.Data(), wantData) {
		return model.AnchorResult{}, "", errors.New("transaction calldata doesn't match this evidence")
	}

	result, err := c.certify(ctx, req, hash.Hex())
	return result, submitter.Hex(), err
}

func (c *Coston2Client) waitMined(ctx context.Context, hash common.Hash) (*types.Transaction, common.Address, error) {
	deadline := time.Now().Add(30 * time.Second)
	for {
		receipt, err := c.rpc.TransactionReceipt(ctx, hash)
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, common.Address{}, errors.New("transaction failed on-chain")
			}
			tx, _, err := c.rpc.TransactionByHash(ctx, hash)
			if err != nil {
				return nil, common.Address{}, err
			}
			signer := types.LatestSignerForChainID(big.NewInt(114))
			from, err := types.Sender(signer, tx)
			if err != nil {
				return nil, common.Address{}, err
			}
			return tx, from, nil
		}
		if time.Now().After(deadline) {
			return nil, common.Address{}, errors.New("transaction not confirmed within 30s — try again in a moment")
		}
		select {
		case <-ctx.Done():
			return nil, common.Address{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (c *Coston2Client) buildSubmitData(req model.AnchorRequest) ([]byte, error) {
	evidenceID := crypto.Keccak256Hash([]byte(req.EvidenceID))
	evidenceCommitment, err := hashToBytes32(req.EvidenceCommitment)
	if err != nil {
		return nil, err
	}
	screenshotHash, err := hashToBytes32(req.ScreenshotHash)
	if err != nil {
		return nil, err
	}
	scrapedTextHash, err := hashToBytes32(req.ScrapedTextHash)
	if err != nil {
		return nil, err
	}
	metadataCommitment, err := hashToBytes32(req.MetadataCommitment)
	if err != nil {
		return nil, err
	}
	claimsRoot, err := hashToBytes32(req.ClaimsRoot)
	if err != nil {
		return nil, err
	}
	return c.contractABI.Pack("submitEvidence", evidenceID, evidenceCommitment, screenshotHash, scrapedTextHash, metadataCommitment, claimsRoot)
}

func (c *Coston2Client) certify(ctx context.Context, req model.AnchorRequest, submitTxHash string) (model.AnchorResult, error) {
	evidenceID := crypto.Keccak256Hash([]byte(req.EvidenceID))

	certificateHash := req.TEECertificateHash
	if certificateHash == "" {
		certificateHash = digest(req.EvidenceID + "|" + req.EvidenceCommitment + "|" + req.ClaimsRoot + "|coston2")
	}
	if !strings.HasPrefix(certificateHash, "0x") {
		certificateHash = "0x" + certificateHash
	}
	status := "pending"
	txHash := submitTxHash

	if c.reporter != nil {
		certBytes, err := hashToBytes32(certificateHash)
		if err != nil {
			return model.AnchorResult{}, err
		}
		certData, err := c.contractABI.Pack("recordTEECertificate", evidenceID, certBytes, true)
		if err != nil {
			return model.AnchorResult{}, err
		}
		certTx, err := c.send(ctx, c.reporter, certData)
		if err != nil {
			return model.AnchorResult{}, err
		}
		txHash = certTx.Hash().Hex()
		status = "certified"
	}

	return model.AnchorResult{
		TxHash:             txHash,
		TEECertificateHash: certificateHash,
		Status:             status,
	}, nil
}

func (c *Coston2Client) send(ctx context.Context, signer *signer, data []byte) (*types.Transaction, error) {
	signer.mu.Lock()
	defer signer.mu.Unlock()

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(signer.privateKey, "0x"))
	if err != nil {
		return nil, err
	}
	nonce, err := c.rpc.PendingNonceAt(ctx, signer.address)
	if err != nil {
		return nil, err
	}
	gasPrice, err := c.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		From:     signer.address,
		To:       &c.contract,
		GasPrice: gasPrice,
		Value:    big.NewInt(0),
		Data:     data,
	}
	gasLimit, err := c.rpc.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = 300000
	}

	tx := types.NewTransaction(nonce, c.contract, big.NewInt(0), gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(114)), privateKey)
	if err != nil {
		return nil, err
	}
	if err := c.rpc.SendTransaction(ctx, signed); err != nil {
		return nil, err
	}
	return signed, nil
}

func newSigner(privateKey string) (*signer, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return nil, err
	}
	return &signer{privateKey: privateKey, address: crypto.PubkeyToAddress(key.PublicKey)}, nil
}

func hashToBytes32(value string) ([32]byte, error) {
	var out [32]byte
	value = strings.TrimPrefix(value, "0x")
	body, err := hex.DecodeString(value)
	if err != nil {
		return out, err
	}
	if len(body) != 32 {
		return out, errors.New("expected 32 byte hash")
	}
	copy(out[:], body)
	return out, nil
}
