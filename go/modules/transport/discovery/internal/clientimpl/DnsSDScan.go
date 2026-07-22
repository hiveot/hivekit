// Package discodiscoclientvery with client for DNS-SD service discovery
package clientimpl

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
// The provided callback is concurrent safe. The scan does not return while the callback is invoked.
//
// results are handled through a callback until the waitTime ends or the callback returns stop=true
//
//	instanceName to look for, or "" for all possible instances
//	serviceType to look for in format "_{serviceName}._tcp", or "" to discover all service types (not all services)
//	waitTime with duration to wait while collecting results. Default is 3 seconds
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
	// 'records' channel captures the result
	records = make([]*zeroconf.ServiceEntry, 0)

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		slog.Error("Failed to create DNS-SD resolver", "err", err)
		return records, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if instanceName == "" || instanceName == entry.Instance {
				// dont let the main function return until this record is handled
				mu.Lock()
				if ctx.Err() != nil {
					mu.Unlock()
					return
				}

				rec := entry.ServiceRecord
				// txtRecord := entry.Text
				slog.Info("DnsSDScan: Found service",
					"instance", rec.Instance,
					// "ipv4", entry.AddrIPv4,
					"service", rec.Service,
					"serviceType", rec.ServiceTypeName(),
					"domain", rec.Domain,
					"ip4", entry.AddrIPv4,
					slog.Int("port", entry.Port),
					// "scheme", strings.Join(txtRecord, ";"), // if found
				)
				records = append(records, entry)
				if cb != nil {
					stop := cb(entry)
					if stop {
						cancel()
					}
				}
				mu.Unlock()
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
	results := records[:]
	mu.Unlock()

	return results, err
}
