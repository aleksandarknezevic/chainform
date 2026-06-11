package abi

import (
	"fmt"
	"os"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

// Load reads and parses a Solidity ABI JSON file.
func Load(path string) (*ethabi.ABI, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open abi %s: %w", path, err)
	}
	defer f.Close()

	parsed, err := ethabi.JSON(f)
	if err != nil {
		return nil, fmt.Errorf("parse abi %s: %w", path, err)
	}
	return &parsed, nil
}
