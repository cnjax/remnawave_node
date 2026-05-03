package xray_client

import (
	"context"
	"crypto/md5"
	"fmt"
	"net"
	"time"

	routerpb "github.com/xtls/xray-core/app/router"
	routerService "github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/serial"
)

// BlockIP adds a source-IP routing rule that sends traffic from ip to the BLOCK outbound.
// The rule tag is computed with generateRuleTag so it matches the TS objectHash output.
func (c *Client) BlockIP(ip, _ string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ruleTag := generateRuleTag(ip)
	return c.AddSrcIPRule(ctx, ruleTag, "BLOCK", ip)
}

// UnblockIP removes the block rule for ip (identified by its rule tag).
func (c *Client) UnblockIP(ip, _ string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ruleTag := generateRuleTag(ip)
	return c.RemoveRuleByTagCtx(ctx, ruleTag)
}

// generateRuleTag produces the same hash as the TS objectHash(ip, {algorithm:'md5'}).
// object-hash serialises a plain string as "string:<byteLength>:<value>" before hashing.
func generateRuleTag(ip string) string {
	input := fmt.Sprintf("string:%d:%s", len(ip), ip)
	h := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", h)
}

// AddSrcIPRule constructs a real xray RoutingRule that matches the source IP and routes
// traffic to outboundTag, then submits it via the router gRPC service.
func (c *Client) AddSrcIPRule(ctx context.Context, ruleTag, outboundTag, ip string) error {
	ipBytes, prefix, err := parseIPCIDR(ip)
	if err != nil {
		return fmt.Errorf("invalid IP %q: %w", ip, err)
	}

	rule := &routerpb.RoutingRule{
		RuleTag: ruleTag,
		TargetTag: &routerpb.RoutingRule_Tag{
			Tag: outboundTag,
		},
		SourceGeoip: []*routerpb.GeoIP{
			{
				Cidr: []*routerpb.CIDR{
					{Ip: ipBytes, Prefix: prefix},
				},
			},
		},
	}

	_, err = c.router.AddRule(ctx, &routerService.AddRuleRequest{
		Config:       serial.ToTypedMessage(rule),
		ShouldAppend: true,
	})
	if err != nil {
		return fmt.Errorf("failed to add src-ip rule (tag=%s ip=%s): %w", ruleTag, ip, err)
	}
	return nil
}

// RemoveRuleByTag removes a routing rule by its tag (creates its own context).
func (c *Client) RemoveRuleByTag(ruleTag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.RemoveRuleByTagCtx(ctx, ruleTag)
}

// RemoveRuleByTagCtx removes a routing rule using a caller-supplied context.
func (c *Client) RemoveRuleByTagCtx(ctx context.Context, ruleTag string) error {
	_, err := c.router.RemoveRule(ctx, &routerService.RemoveRuleRequest{
		RuleTag: ruleTag,
	})
	if err != nil {
		return fmt.Errorf("failed to remove rule (tag=%s): %w", ruleTag, err)
	}
	return nil
}

// parseIPCIDR returns the raw bytes and prefix length for a host address.
// IPv4 → 4 bytes, /32; IPv6 → 16 bytes, /128.
func parseIPCIDR(ip string) ([]byte, uint32, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, 0, fmt.Errorf("cannot parse %q as IP", ip)
	}
	if v4 := parsed.To4(); v4 != nil {
		return []byte(v4), 32, nil
	}
	return []byte(parsed.To16()), 128, nil
}
