package resolve

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nnu-connect/nnu-connect/client"
	"github.com/nnu-connect/nnu-connect/internal/dnsutil"
	"github.com/nnu-connect/nnu-connect/internal/ippool"
	"github.com/nnu-connect/nnu-connect/log"
	"github.com/nnu-connect/nnu-connect/stack"
	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

type Resolver struct {
	remoteUDPResolver *net.Resolver
	remoteTCPResolver *net.Resolver
	secondaryResolver *net.Resolver
	ttl               uint64
	domainResources   map[string]client.DomainResource
	dnsResource       map[string]net.IP
	useRemoteDNS      bool

	dnsCache *cache.Cache

	IPPool *ippool.IPPool[client.DomainResource]

	timer  *time.Timer
	useTCP bool
	// check to use tcp resolver or udp resolver
	tcpLock sync.RWMutex
	lookups singleflight.Group

	closeOnce sync.Once
}

type contextKey string

var (
	ContextKeyFakeIP         = contextKey("FAKE_IP")
	ContextKeyResolveHost    = contextKey("RESOLVE_HOST")
	ContextKeyDomainResource = contextKey("DOMAIN_RESOURCE")
)

// Resolve ip address. If the host could be visited via VPN, this function set a DOMAIN_RESOURCE value in context. If resolve success, this function set a RESOLVE_HOST value in context.
func (r *Resolver) Resolve(ctx context.Context, host string) (resCtx context.Context, resIP net.IP, resErr error) {
	defer func() {
		if resErr == nil {
			resCtx = context.WithValue(resCtx, ContextKeyResolveHost, host)
		}
	}()
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	var domainResourceFound = false
	var domainResource client.DomainResource
	var bestLength int
	for domain, resource := range r.domainResources {
		exact := !strings.HasPrefix(domain, ".") && !strings.HasPrefix(domain, "*")
		domain = strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(domain), "*"), ".")
		if host != domain && !strings.HasSuffix(host, "."+domain) {
			continue
		}
		score := 2 * len(domain)
		if exact {
			score++
		}
		if score > bestLength {
			bestLength = score
			domainResourceFound = true
			domainResource = resource
		}
	}
	if domainResourceFound {
		ctx = context.WithValue(ctx, ContextKeyDomainResource, domainResource)
	}

	if cachedIP, found := r.getDNSCache(host); found {
		log.Printf("%s -> %s", host, cachedIP.String())
		return ctx, cachedIP, nil
	}

	if r.dnsResource != nil {
		if ip, found := r.dnsResource[host]; found {
			log.Printf("%s -> %s", host, ip.String())
			if domainResourceFound {
				err := r.IPPool.SetIPDomain(ip, host, domainResource)
				if err != nil {
					log.DebugPrintf("Set IP err: %s", err)
				}
			}
			return ctx, ip, nil
		}

		if fakeIPValue := ctx.Value(ContextKeyFakeIP); fakeIPValue != nil {
			if domainResourceFound {
				ip := r.IPPool.GenerateIP(host, domainResource)
				log.Printf("%s -> %s (Fake IP)", host, ip.String())
				return ctx, ip, nil
			}
		}
	}

	result := r.lookups.DoChan(host, func() (any, error) {
		if ip, ok := r.getDNSCache(host); ok {
			return ip, nil
		}
		query, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if r.useRemoteDNS {
			r.tcpLock.RLock()
			useTCP := r.useTCP
			r.tcpLock.RUnlock()
			if !useTCP {
				if ips, err := lookupRemote(query, r.remoteUDPResolver, host, 900*time.Millisecond); err == nil {
					r.setDNSCache(host, ips[0])
					return ips[0], nil
				}
			}
			if ips, err := lookupRemote(query, r.remoteTCPResolver, host, 1500*time.Millisecond); err == nil {
				if !useTCP {
					r.tcpLock.Lock()
					r.useTCP = true
					if r.timer == nil {
						r.timer = time.AfterFunc(10*time.Minute, func() {
							r.tcpLock.Lock()
							r.useTCP = false
							r.timer = nil
							r.tcpLock.Unlock()
						})
					}
					r.tcpLock.Unlock()
				}
				r.setDNSCache(host, ips[0])
				return ips[0], nil
			}
		}
		_, ip, err := r.ResolveWithSecondaryDNS(query, host)
		if err == nil {
			r.setDNSCache(host, ip)
		}
		return ip, err
	})
	select {
	case <-ctx.Done():
		return ctx, nil, ctx.Err()
	case resolved := <-result:
		if resolved.Err != nil {
			return ctx, nil, resolved.Err
		}
		ip := resolved.Val.(net.IP)
		log.Printf("%s -> %s", host, ip)
		return ctx, ip, nil
	}
}

func (r *Resolver) RemoteUDPResolver() (*net.Resolver, error) {
	if r.remoteUDPResolver != nil {
		return r.remoteUDPResolver, nil
	} else {
		return nil, errors.New("remote UDP resolver is nil")
	}
}

func (r *Resolver) RemoteTCPResolver() (*net.Resolver, error) {
	if r.remoteTCPResolver != nil {
		return r.remoteTCPResolver, nil
	} else {
		return nil, errors.New("remote TCP resolver is nil")
	}
}

func (r *Resolver) ResolveWithSecondaryDNS(ctx context.Context, host string) (context.Context, net.IP, error) {
	if targets, err := dnsutil.LookupIP(ctx, r.secondaryResolver, "ip4", host); err != nil {
		log.Printf("Resolve IPv4 addr failed using secondary DNS: %s. Try IPv6 addr", host)

		if targets, err = dnsutil.LookupIP(ctx, r.secondaryResolver, "ip6", host); err != nil {
			log.Printf("Resolve IPv6 addr failed using secondary DNS: %s", host)
			return ctx, nil, err
		} else {
			log.Printf("%s -> %s", host, targets[0].String())
			return ctx, targets[0], nil
		}
	} else {
		log.Printf("%s -> %s", host, targets[0].String())
		return ctx, targets[0], nil
	}
}

func (r *Resolver) Close() {
	r.closeOnce.Do(func() {
		r.tcpLock.Lock()
		if r.timer != nil {
			r.timer.Stop()
			r.timer = nil
		}
		r.tcpLock.Unlock()
	})
}

func NewResolver(stack stack.Stack, remoteDNSServer, secondaryDNSServer string, ttl uint64, domainResources map[string]client.DomainResource, dnsResource map[string]net.IP, useRemoteDNS bool) *Resolver {
	//domainSuffixTree := domainsuffixtrie.NewDomainSuffixTrie[bool]()
	//for domain := range domainResource {
	//	_ = domainSuffixTree.AddDomainSuffix(domain, true)
	//}

	resolver := &Resolver{
		remoteUDPResolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				if strings.HasPrefix(network, "tcp") {
					return stack.DialTCP(ctx, &net.TCPAddr{IP: net.ParseIP(remoteDNSServer), Port: 53})
				}
				return stack.DialUDP(ctx, &net.UDPAddr{
					IP:   net.ParseIP(remoteDNSServer),
					Port: 53,
				})
			},
		},
		remoteTCPResolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return stack.DialTCP(ctx, &net.TCPAddr{
					IP:   net.ParseIP(remoteDNSServer),
					Port: 53,
				})
			},
		},
		ttl:             ttl,
		domainResources: domainResources,
		dnsResource:     dnsResource,
		dnsCache:        cache.New(time.Duration(ttl)*time.Second, time.Duration(ttl)*2*time.Second),
		useRemoteDNS:    useRemoteDNS,
	}

	if secondaryDNSServer != "" {
		resolver.secondaryResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(secondaryDNSServer, "53"))
			},
		}
	} else {
		resolver.secondaryResolver = &net.Resolver{
			PreferGo: true,
		}
	}
	var err error
	resolver.IPPool, err = ippool.NewIPPool[client.DomainResource]("198.18.0.0/16")
	if err != nil {
		log.Fatalf("Create Fake IP Pool failed: %v", err)
	}

	return resolver
}

func lookupRemote(ctx context.Context, resolver *net.Resolver, host string, timeout time.Duration) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ips, err := resolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	var real []net.IP
	for _, ip := range ips {
		if !dnsutil.IsFakeIP(ip) {
			real = append(real, ip)
		}
	}
	if len(real) == 0 {
		return nil, errors.New("campus DNS returned no real addresses")
	}
	return real, nil
}
