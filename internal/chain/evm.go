package chain

import (
	"context"
	"fmt"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client is a live JSON-RPC backed Reader.
type Client struct {
	rpc string
	eth *ethclient.Client
}

// Dial connects to an EVM JSON-RPC endpoint.
func Dial(ctx context.Context, rpcURL string) (*Client, error) {
	eth, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", rpcURL, err)
	}
	return &Client{rpc: rpcURL, eth: eth}, nil
}

// Read performs an eth_call and decodes the result against call.Outputs.
func (c *Client) Read(ctx context.Context, call ViewCall) ([]any, error) {
	data, err := Pack(call.Method, call.Inputs, call.Args...)
	if err != nil {
		return nil, err
	}
	to := call.To
	out, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
	if err != nil {
		return nil, fmt.Errorf("eth_call %s: %w", call.Method, err)
	}
	return Unpack(call.Outputs, out)
}

// Close releases the underlying connection.
func (c *Client) Close() {
	if c.eth != nil {
		c.eth.Close()
	}
}
