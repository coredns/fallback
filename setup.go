package fallback

import (
	"fmt"
	"strings"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/proxy"

	"github.com/caddyserver/caddy"
	"github.com/miekg/dns"
)

func init() {
	caddy.RegisterPlugin("fallback", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	t := dnsserver.GetConfig(c).Handler("trace")
	f := New(t)

	for c.Next() {
		var (
			original bool
			rcode    string
		)
		if !c.Dispenser.Args(&rcode) {
			return c.ArgErr()
		}
		if rcode == "original" {
			original = true
			// Reread parameter is not rcode. Get it again.
			if !c.Dispenser.Args(&rcode) {
				return c.ArgErr()
			}
		}

		rc, ok := dns.StringToRcode[strings.ToUpper(rcode)]
		if !ok {
			return fmt.Errorf("%s is not a valid rcode", rcode)
		}

		u, err := proxy.NewStaticUpstream(&c.Dispenser)
		if err != nil {
			return plugin.Error("fallback", err)
		}

		if _, ok := f.rules[rc]; ok {
			return fmt.Errorf("rcode '%s' is specified more than once", rcode)
		}
		f.rules[rc] = rule{original: original, proxyUpstream: u}
		if original {
			f.original = true
		}
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		f.Next = next
		return f
	})

	return nil
}
