package wotco

import (
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/clients"
	clientspkg "github.com/hiveot/hivekit/go/modules/clients/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
)

// WotConsumer contains the consumer of WoT devices for discovery and loading of Things.
// This is used in the examples to show how to use the discovery and client modules,
// and to keep the state of discovered things and their TDs.
//
// This consumer is a HiveModule, so it can be used as a sink in the TUI example,
// but it can also be used in a CLI or other application.
type WotConsumer struct {
	clientspkg.Consumer

	// The token for authentication
	// todo. right now this is a placeholder
	authToken string

	// The CA for connecting
	// todo right now this is a placeholder
	caCert *x509.Certificate

	// existing connections by thingID
	clients map[string]transports.ITransportClient

	// discovered directories by directoryID
	directories map[string]*td.TD

	// the connection this module links to
	// connection transports.ITransportClient

	// discovery records found after Discover()
	records []*discoverypkg.DiscoveryResult

	// discovered things by thingID
	things map[string]*td.TD

	mux sync.RWMutex
}

// Create a Thing connection using the TD of the thingID
func (co *WotConsumer) Connect(thingID string) (transports.ITransportClient, error) {
	co.mux.Lock()
	defer co.mux.Unlock()
	c, found := co.clients[thingID]
	if found {
		return c, nil
	}
	tdoc, found := co.things[thingID]
	if !found {
		return nil, fmt.Errorf("No TD for thing %s", thingID)
	}
	c, err := clients.NewTransportClientFromTD(tdoc, co.caCert, nil)
	if err == nil {
		err = fmt.Errorf("n/c")
		// FIXME: determine clientID and token
		err = c.ConnectWithToken("", "")

		co.clients[thingID] = c
	}
	return c, err
}

// Discover all published things and directories.
// cb is an optional callback to invoke with ongoing results. Return true to cancel.
func (co *WotConsumer) Discover(cb func(r *discoverypkg.DiscoveryResult) bool) (err error) {
	// fmt.Print("Discover started ")

	disco := discoverypkg.NewDiscoveryClient()
	waitDuration := time.Second * 1

	co.records, err = disco.DiscoverThings("", waitDuration, func(r *discoverypkg.DiscoveryResult) bool {

		// load the TD to present nr of affordances
		_, err := co.LoadDiscoveredTD(r)

		if err != nil {
			slog.Error("Error reading TD", "err", err.Error())
		}

		if cb != nil {
			cancel := cb(r)
			if cancel {
				return true
			}
		}

		// notify event listeners of the newly discovered record
		// TODO: formalize this with a TD
		notif := msg.NewNotificationMessage(co.GetClientID(),
			msg.AffordanceTypeEvent, discovery.DefaultDiscoveryThingID, "discovery", r)
		co.ForwardNotification(notif)
		return false
	})
	return err
}

func (co *WotConsumer) GetConnection(thingID string) (c transports.ITransportClient, found bool) {
	co.mux.RLock()
	defer co.mux.RUnlock()

	c, found = co.clients[thingID]
	return c, found
}

// Return a list of discovered directories
func (co *WotConsumer) GetDirectories() map[string]*td.TD {
	return co.directories
}

// return the property value as a string
// This establishes a connection if it doesn't yet exist
func (co *WotConsumer) GetPropValue(thingID string, name string) string {
	var err error
	c, found := co.GetConnection(thingID)
	if !found {
		// c, err = model.Connect(thingID)
		return ""
	}
	if err != nil {
		return err.Error()
	}
	req := msg.NewRequestMessage("", td.OpReadProperty, thingID, name, nil, "")
	resp, err := msg.ForwardRequestWait(req, c.HandleRequest, msg.DefaultRnRTimeout)
	if err != nil {
		return err.Error()
	}
	return resp.ToString(0)
}

func (co *WotConsumer) GetRecords() []*discoverypkg.DiscoveryResult {
	return co.records
}

// Return the TD of a discovered thing
// Intended for the router module to connect to a device.
func (co *WotConsumer) GetTD(thingID string) *td.TD {
	td := co.things[thingID]
	return td
}

// Return a list of discovered things
func (co *WotConsumer) GetThings() map[string]*td.TD {
	return co.things
}

// Load the TD from the discovery record URL
// This adds the TD to the known things or directories and returns the TD, or an error
func (co *WotConsumer) LoadDiscoveredTD(r *discoverypkg.DiscoveryResult) (tdoc *td.TD, err error) {

	tdURL := r.AsURL()
	resp, err := http.Get(tdURL)
	if err == nil {
		raw, _ := io.ReadAll(resp.Body)
		tdoc, err = td.UnmarshalTD(string(raw))
	}
	if err != nil {
		return nil, err
	}
	if r.IsDirectory {
		co.directories[tdoc.ID] = tdoc
	} else {
		co.things[tdoc.ID] = tdoc
	}
	return tdoc, err
}

// ReadDirectory reads all the TD in the discovered directory, up to the given limit
// Note that without credentials this can fails
func (co *WotConsumer) ReadDirTDs(dirTD *td.TD, limit int) {
	var n = 0
	slog.Info("ReadDirTDs started")
	// TODO
	slog.Info("ReadDirTDs completed", "count", n)
}

// ReadThing reads the properties of the Thing and returns a text document
// containing a list of values.
// Note that without credentials this can fails
func (co *WotConsumer) ReadThing(thingID string) []string {
	lines := []string{}
	tdoc, found := co.things[thingID]
	if !found {
		slog.Warn("No TD for thing: " + thingID)
		return nil
	}
	lines = append(lines, fmt.Sprintf("Thing ID: %s", thingID))
	lines = append(lines, fmt.Sprintf("ThingID:     %s\n", tdoc.ID))
	lines = append(lines, fmt.Sprintf("Title:       %s\n", tdoc.Title))
	lines = append(lines, fmt.Sprintf("Type:        %s\n", tdoc.AtType))
	lines = append(lines, fmt.Sprintf("Description: %s\n", tdoc.Description))

	c, err := clients.NewTransportClientFromTD(tdoc, co.caCert, nil)
	if err == nil {
		defer c.Close()
		c.SetTimeout(time.Minute) // for testing
		err = c.ConnectWithToken(co.GetClientID(), co.authToken)
	}
	if err == nil {
		co.SetRequestSink(c.HandleRequest)
	} else {
		slog.Warn("ReadThing: failed", "err", err.Error())
	}

	lines = append(lines, fmt.Sprintf("Properties: (%d)\n", len(tdoc.Properties)))
	propValues, err := co.ReadAllProperties(thingID)
	if err != nil {
		slog.Error("ReadAllProperties Failed", "err", err.Error())
	} else {
		for name := range tdoc.Properties {
			propVal, found := propValues[name]
			if found {
				lines = append(lines, fmt.Sprintf("   %s: %v\n", name, propVal))
			} else {
				lines = append(lines, fmt.Sprintf("   %s: %s\n", name, propValues[name]))
			}
		}
	}

	lines = append(lines, fmt.Sprintf("Events: (%d)\n", len(tdoc.Events)))
	eventValues, err := co.ReadAllEvents(thingID)
	if err != nil {
		slog.Error("ReadAllEvents Failed", "err", err.Error())
	} else {
		// list events defined in the TD with their value
		for name := range tdoc.Events {
			evVal, found := eventValues[name]
			if found {
				lines = append(lines, fmt.Sprintf("   %s: %s at %s\n", name, evVal.ToString(0), evVal.Timestamp))
			} else {
				lines = append(lines, fmt.Sprintf("   %s: %s\n", name, "n/a"))
			}
		}
	}

	lines = append(lines, fmt.Sprintf("Actions: (%d)\n", len(tdoc.Actions)))
	actionValues, err := co.QueryAllActions(thingID)
	if err != nil {
		slog.Error("QueryAllActions Failed", "err", err.Error())
	} else {
		for name := range tdoc.Actions {
			av, found := actionValues[name]
			if found {
				lines = append(lines, fmt.Sprintf("   %s: %s\n", name, av.Status))
			} else {
				lines = append(lines, fmt.Sprintf("   %s\n", name))
			}
		}
	}
	return lines
}

// ShowSummary shows a summary of discovery and things
// intended for display at the top of the commandline input
// func (cli *CliModel) GetSummary() string {
// 	return fmt.Sprintf("\n%d Things loaded", len(cli.things))
// }

// Create a new instance of the WotConsumer.
// This consumer is intended for use by an CLI ro TUI to discover things and show their state.
//
// Use this with the router module as sink to connect to discovered clients and provide
// the router with the callback to get a Thing TD:
//
//	wotConsumer := wotco.NewWotConsumer()
//	r := routerpkg.NewRouterService("", wotConsumer.GetTD, nil, certsBundle.CaCert)
//	wotConsumer.SetRequestSink(r.HandleRequest)
//	r.SetNotificationSink(wotConsumer.HandleNotification)

func NewWotConsumer() *WotConsumer {
	cl := &WotConsumer{
		authToken:   "no-token",
		records:     make([]*discoverypkg.DiscoveryResult, 0),
		clients:     make(map[string]transports.ITransportClient),
		Consumer:    *clientspkg.NewConsumer(""),
		directories: make(map[string]*td.TD),
		things:      make(map[string]*td.TD),
	}
	return cl
}

// This module can be used in a factory recipe.
func NewWotConsumerFactory(f factory.IModuleFactory) modules.IHiveModule {
	cl := NewWotConsumer()
	return cl
}
