// Package config defines the ChainForm desired-state schema and loader.
//
// A ChainForm configuration is the single source of truth for the intended
// on-chain state of a protocol. It is the "desired state" half of the
// reconciliation loop; the actual state is read from the chain at plan time.
//
// Configurations are written in HCL, the same language Terraform uses:
//
//	chain {
//	  name     = "ethereum"
//	  chain_id = 1
//	  rpc      = env("RPC_URL")
//	}
//
//	resource "protocol" "main" {
//	  address = "0x..."
//	  feeBps  = 30
//	  paused  = false
//	}
package config

import (
	"errors"
	"fmt"
)

// Config is the root of a ChainForm configuration document.
type Config struct {
	// Version pins the configuration schema version (currently "1").
	Version string

	// Chain identifies the target EVM network and how to reach it.
	Chain Chain

	// Resources is the set of managed on-chain entities.
	Resources []ResourceConfig
}

// Chain describes the target EVM network.
type Chain struct {
	// Name is a human-readable network label, e.g. "ethereum" or "arbitrum".
	Name string

	// ChainID is the EIP-155 chain id. Required.
	ChainID uint64

	// RPC is the JSON-RPC endpoint. Use the env("VAR") function in HCL to keep
	// secrets and endpoints out of version control.
	RPC string
}

// ResourceConfig is one managed on-chain entity, analogous to a Terraform
// resource block. The Type selects the provider that knows how to read and
// reconcile it; Spec carries the type-specific desired attributes.
type ResourceConfig struct {
	// Type selects the resource provider, e.g. "protocol". Required.
	// Comes from the first label of a `resource "TYPE" "NAME" {}` block.
	Type string

	// Name is a unique local identifier used in plan output. Required.
	// Comes from the second label of the resource block.
	Name string

	// Address is the 0x-prefixed contract address. Required.
	Address string

	// Spec holds the desired attributes, interpreted by the resource provider.
	Spec map[string]any

	// Expect holds read-only assertions declared in an `expect` block: expected
	// values for attributes that have a getter but no setter. Drift on these is
	// reported but never converged into an operation.
	Expect map[string]any
}

// Validate performs schema-level checks that are independent of any provider.
// Provider-specific validation happens when a resource is built.
func (c *Config) Validate() error {
	if c.Chain.ChainID == 0 {
		return errors.New("chain.chain_id is required")
	}
	if len(c.Resources) == 0 {
		return errors.New("no resources defined")
	}
	seen := make(map[string]bool, len(c.Resources))
	for i, r := range c.Resources {
		switch {
		case r.Type == "":
			return fmt.Errorf("resources[%d]: type is required", i)
		case r.Name == "":
			return fmt.Errorf("resources[%d]: name is required", i)
		case r.Address == "":
			return fmt.Errorf("resource %q: address is required", r.Name)
		}
		if seen[r.Name] {
			return fmt.Errorf("duplicate resource name %q", r.Name)
		}
		seen[r.Name] = true
	}
	return nil
}
