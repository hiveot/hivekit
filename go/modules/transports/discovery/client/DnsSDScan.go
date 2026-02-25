// Package discodiscoclientvery with client for DNS-SD service discovery
package discoveryclient

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// DnsSDScan scans zeroconf publications on local domain
//
// The zeroconf library does not support browsing of all services, but a workaround is
// to search the service types with "_services._dns-sd._udp" then query each of the service types.
//
// results are handled through a callback until the waitTime ends or the callback returns stop=true
//
//	instanceName to look for, or "" for all possible instances
//	serviceType to look for in format "_{serviceName}._tcp", or "" to discover all service types (not all services)
//	waitTime with duration to wait while collecting results. 0 means exit on the first result.
//	cb is the optional callback invoked when a result is found
func DnsSDScan(instanceName string, serviceType string, waitTime time.Duration,
	cb func(*zeroconf.ServiceEntry) (stop bool)) (records []*zeroconf.ServiceEntry, err error) {

	sdDomain := "local"
	mu := &sync.Mutex{}

	if serviceType == "" {
		// https://github.com/grandcat/zeroconf/pull/15
		serviceType = "_services._dns-sd._udp"
	}
	if waitTime == 0 {
		waitTime = time.Second * 3
	}
	records = make([]*zeroconf.ServiceEntry, 0)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		slog.Error("Failed to create DNS-SD resolver", "err", err)
		return records, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	// 'records' channel captures the result
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if instanceName == "" || instanceName == entry.Instance {
				rec := entry.ServiceRecord
				slog.Info("DnsSDScan: Found service",
					"instance", rec.Instance,
					// "ipv4", entry.AddrIPv4,
					"service", rec.Service,
					"domain", rec.Domain,
					"ip4", entry.AddrIPv4,
					slog.Int("port", entry.Port))

				mu.Lock()
				records = append(records, entry)
				mu.Unlock()
				if cb != nil {
					stop := cb(entry)
					if stop {
						cancel()
					}
				}
			} else {
				// ignore this record
			}
		}
		slog.Debug("DnsSDScan: No more entries.")
	}(entries)

	err = resolver.Browse(ctx, serviceType, sdDomain, entries)
	if err != nil {
		slog.Error("DnsSDScan: Failed to browse", "err", err)
	}
	<-ctx.Done()
	mu.Lock()
	results := records
	mu.Unlock()

	return results, err
}
