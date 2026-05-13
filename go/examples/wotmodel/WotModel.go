package wotmodel

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
	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
)

// WotModel contains the client side data model for discovering and loading Things
// This is used in the examples to show how to use the discovery and client modules,
// and to keep the state of discovered things and their TDs.
//
// This model is implemented as a HiveModule, so it can be used in the TUI example,
// but it can also be used in a CLI or other application.
type WotModel struct {
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
func (model *WotModel) Connect(thingID string) (transports.ITransportClient, error) {
	model.mux.Lock()
	defer model.mux.Unlock()
	c, found := model.clients[thingID]
	if found {
		return c, nil
	}
	tdoc, found := model.things[thingID]
	if !found {
		return nil, fmt.Errorf("No TD for thing %s", thingID)
	}
	c, err := clients.NewTransportClientFromTD(tdoc, model.caCert, nil)
	if err == nil {
		err = fmt.Errorf("n/c")
		// FIXME: determine clientID and token
		err = c.ConnectWithToken("", "")
		model.clients[thingID] = c
	}
	return c, err
}

// Discover all published things and directories
func (model *WotModel) Discover() (err error) {
	// fmt.Print("Discover started ")

	disco := discoverypkg.NewDiscoveryClient()
	waitDuration := time.Second * 1

	model.records, err = disco.DiscoverThings("", waitDuration, func(r *discoverypkg.DiscoveryResult) bool {

		// load the TD to present nr of affordances
		_, err := model.LoadDiscoveredTD(r)

		if err != nil {
			slog.Error("Error reading TD", "err", err.Error())
		}

		// notify event listeners of the newly discovered record
		// TODO: formalize this with a TD
		notif := msg.NewNotificationMessage(model.GetClientID(),
			msg.AffordanceTypeEvent, r.Instance, "discovery", r)
		model.ForwardNotification(notif)
		return false
	})
	return err
}

// Return a list of discovered directories
func (model *WotModel) GetDirectories() map[string]*td.TD {
	return model.directories
}

// Return a list of discovered things
func (model *WotModel) GetThings() map[string]*td.TD {
	return model.things
}

func (model *WotModel) GetConnection(thingID string) (c transports.ITransportClient, found bool) {
	model.mux.RLock()
	defer model.mux.RUnlock()

	c, found = model.clients[thingID]
	return c, found
}

func (model *WotModel) GetRecords() []*discoverypkg.DiscoveryResult {
	return model.records
}

// return the property value as a string
// This establishes a connection if it doesn't yet exist
func (model *WotModel) GetPropValue(thingID string, name string) string {
	var err error
	c, found := model.GetConnection(thingID)
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

// Load the TD from the discovery record URL
// This adds the TD to the known things or directories and returns the TD, or an error
func (model *WotModel) LoadDiscoveredTD(r *discoverypkg.DiscoveryResult) (tdoc *td.TD, err error) {

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
		model.directories[tdoc.ID] = tdoc
	} else {
		model.things[tdoc.ID] = tdoc
	}
	return tdoc, err
}

// ReadDirectory reads all the TD in the discovered directory, up to the given limit
// Note that without credentials this can fails
func (model *WotModel) ReadDirTDs(dirTD *td.TD, limit int) {
	var n = 0
	slog.Info("ReadDirTDs started")
	slog.Info("ReadDirTDs completed", "count", n)
}

// ReadThing reads the properties of the Thing and returns a text document
// containing a list of values.
// Note that without credentials this can fails
func (model *WotModel) ReadThing(thingID string) []string {
	lines := []string{}
	tdoc, found := model.things[thingID]
	if !found {
		slog.Warn("No TD for thing: " + thingID)
		return nil
	}
	lines = append(lines, fmt.Sprintf("Thing ID: %s", thingID))
	lines = append(lines, fmt.Sprintf("ThingID:     %s\n", tdoc.ID))
	lines = append(lines, fmt.Sprintf("Title:       %s\n", tdoc.Title))
	lines = append(lines, fmt.Sprintf("Type:        %s\n", tdoc.AtType))
	lines = append(lines, fmt.Sprintf("Description: %s\n", tdoc.Description))

	c, err := clients.NewTransportClientFromTD(tdoc, model.caCert, nil)
	if err == nil {
		defer c.Close()
		c.SetTimeout(time.Minute) // for testing
		err = c.ConnectWithToken(model.GetClientID(), model.authToken)
	}
	if err == nil {
		model.SetRequestSink(c.HandleRequest)
	} else {
		slog.Warn("ReadThing: failed", "err", err.Error())
	}

	lines = append(lines, fmt.Sprintf("Properties: (%d)\n", len(tdoc.Properties)))
	propValues, err := model.ReadAllProperties(thingID)
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
	eventValues, err := model.ReadAllEvents(thingID)
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
	actionValues, err := model.QueryAllActions(thingID)
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

func NewWotModel() *WotModel {
	cl := &WotModel{
		authToken:   "no-token",
		records:     make([]*discoverypkg.DiscoveryResult, 0),
		clients:     make(map[string]transports.ITransportClient),
		Consumer:    *clientspkg.NewConsumer(""),
		directories: make(map[string]*td.TD),
		things:      make(map[string]*td.TD),
	}
	return cl
}

func NewWotModelFactory(f factory.IModuleFactory) modules.IHiveModule {
	cl := NewWotModel()
	return cl
}
