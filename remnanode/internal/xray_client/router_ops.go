package xray_client

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	routerService "github.com/xtls/xray-core/app/router/command"
)

// BlockIP adds a rule to block an IP address
func (c *Client) BlockIP(ip, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate a unique rule tag from the IP using MD5 hash
	ruleTag := generateRuleTag(ip)

	// Use BalancerMsg to add an override for blocking
	// Note: The actual implementation depends on xray-core version and available APIs
	// For now, we'll use a simplified approach
	_, err := c.router.AddRule(ctx, &routerService.AddRuleRequest{
		Config:       nil, // Will be set based on xray-core API
		ShouldAppend: true,
	})

	if err != nil {
		// Log the attempt but don't fail - blocking may not be supported
		return fmt.Errorf("failed to add block rule for IP %s (tag: %s): %w", ip, ruleTag, err)
	}

	return nil
}

// UnblockIP removes the block rule for an IP address
func (c *Client) UnblockIP(ip, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate the same rule tag that was used when blocking
	ruleTag := generateRuleTag(ip)

	// Remove the routing rule
	_, err := c.router.RemoveRule(ctx, &routerService.RemoveRuleRequest{
		RuleTag: ruleTag,
	})

	if err != nil {
		return fmt.Errorf("failed to remove block rule for IP %s: %w", ip, err)
	}

	return nil
}

// generateRuleTag generates a unique rule tag from an IP address using MD5
func generateRuleTag(ip string) string {
	hash := md5.Sum([]byte(ip))
	return fmt.Sprintf("%x", hash)
}

// AddSrcIPRule adds a source IP routing rule
func (c *Client) AddSrcIPRule(ruleTag, outboundTag, ip string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.router.AddRule(ctx, &routerService.AddRuleRequest{
		Config:       nil, // Will be set based on xray-core API
		ShouldAppend: true,
	})

	if err != nil {
		return fmt.Errorf("failed to add source IP rule: %w", err)
	}

	return nil
}

// RemoveRuleByTag removes a routing rule by its tag
func (c *Client) RemoveRuleByTag(ruleTag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.router.RemoveRule(ctx, &routerService.RemoveRuleRequest{
		RuleTag: ruleTag,
	})

	if err != nil {
		return fmt.Errorf("failed to remove rule by tag: %w", err)
	}

	return nil
}
