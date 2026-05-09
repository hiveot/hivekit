package wotmodel

import (
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/clients"
	clientspkg "github.com/hiveot/hivekit/go/modules/clients/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
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

	// the connection this module links to
	// connection transports.ITransportClient

	// discovery records found after Discover()
	records []*discoverypkg.DiscoveryResult

	// discovered things by thingID
	things map[string]*td.TD
}

func (model *WotModel) GetThings() map[string]*td.TD {
	return model.things
}

// Load the TD from the discovery record URL
// This adds the TD to the known things and returns the TD, or an error
func (model *WotModel) LoadDiscoveredTD(r *discoverypkg.DiscoveryResult) (tdoc *td.TD, err error) {

	tdURL := r.AsURL()
	resp, err := http.Get(tdURL)
	if err == nil {
		raw, _ := io.ReadAll(resp.Body)
		tdoc, err = td.UnmarshalTD(string(raw))
	}
	if err == nil {
		model.things[tdoc.ID] = tdoc
	}
	return tdoc, err
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
			fmt.Printf(" Error reading TD: %s\n", err.Error())
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

func (model *WotModel) GetRecords() []*discoverypkg.DiscoveryResult {
	return model.records
}

// ReadDirectory reads all the TD in the discovered directory, up to the given limit
// Note that without credentials this can fails
func (model *WotModel) ReadDirTDs(dirTD *td.TD, limit int) {
	var n = 0
	slog.Info("ReadDirTDs started")
	slog.Info("ReadDirTDs completed", "count", n)
}

// ReadThing reads the properties of the Thing and display a list of values
// Note that without credentials this can fails
func (model *WotModel) ReadThing(thingID string) {
	tdoc, found := model.things[thingID]
	if !found {
		fmt.Println("No TD for thing: " + thingID)
		return
	}
	fmt.Printf("")
	fmt.Printf("ThingID:     %s\n", tdoc.ID)
	fmt.Printf("Title:       %s\n", tdoc.Title)
	fmt.Printf("Type:        %s\n", tdoc.AtType)
	fmt.Printf("Description: %s\n", tdoc.Description)

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

	fmt.Printf("Properties: (%d)\n", len(tdoc.Properties))
	propValues, err := model.ReadAllProperties(thingID)
	if err != nil {
		fmt.Printf("   ERROR: %s\n", err.Error())
	} else {
		for name := range tdoc.Properties {
			propVal, found := propValues[name]
			if found {
				fmt.Printf("   %s: %v\n", name, propVal)
			} else {
				fmt.Printf("   %s: %s\n", name, propValues[name])
			}
		}
	}

	fmt.Printf("Events: (%d)\n", len(tdoc.Events))
	eventValues, err := model.ReadAllEvents(thingID)
	if err != nil {
		fmt.Printf("   ERROR: %s\n", err.Error())
	} else {
		// list events defined in the TD with their value
		for name := range tdoc.Events {
			evVal, found := eventValues[name]
			if found {
				fmt.Printf("   %s: %s at %s\n", name, evVal.ToString(0), evVal.Timestamp)
			} else {
				fmt.Printf("   %s: %s\n", name, "n/a")
			}
		}
	}

	fmt.Printf("Actions: (%d)\n", len(tdoc.Actions))
	actionValues, err := model.QueryAllActions(thingID)
	if err != nil {
		fmt.Printf("   ERROR: %s\n", err.Error())
	} else {
		for name := range tdoc.Actions {
			av, found := actionValues[name]
			if found {
				fmt.Printf("   %s: %s\n", name, av.Status)
			} else {
				fmt.Printf("   %s\n", name)
			}
		}
	}
}

// ShowSummary shows a summary of discovery and things
// intended for display at the top of the commandline input
// func (cli *CliModel) GetSummary() string {
// 	return fmt.Sprintf("\n%d Things loaded", len(cli.things))
// }

func NewWotModel() *WotModel {
	cl := &WotModel{
		records:   make([]*discoverypkg.DiscoveryResult, 0),
		things:    make(map[string]*td.TD),
		Consumer:  *clientspkg.NewConsumer(""),
		authToken: "no-token",
	}
	return cl
}

func NewWotModelFactory(f factory.IModuleFactory) modules.IHiveModule {
	cl := NewWotModel()
	return cl
}
