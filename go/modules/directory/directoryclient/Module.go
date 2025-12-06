// package directoryclient with the module interface
package directoryclient

import (
	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/wot/td"
	"gopkg.in/yaml.v3"
)

// Define the configuration of this module
type ModuleConfig struct {
	// URL of the directory server, nil for pure manual mode
	DirectoryURL string `yaml:"directoryURL"`
}

// GetTM: this module does not expose a TM
func (m *DirectoryClient) GetTM() *td.TD {
	return nil
}

// HandleRequest does not expect any requests
func (m *DirectoryClient) HandleRequest(request *messaging.RequestMessage) *messaging.ResponseMessage {
	return nil
}

// HandleNotification passes directory update notifications from the actual directory
// to the sinks.
func (m *DirectoryClient) HandleNotification(*messaging.NotificationMessage) {
}

// AddSink adds a sink to forward directory updates to.
func (m *DirectoryClient) AddSink(sink modules.IHiveModule) error {
	m.sinks = append(m.sinks, sink)
	return nil
}

// Publish a TD update to sinks
// TODO: differentitate between add/update/remove
func (m *DirectoryClient) publishAsNotification(tdi *td.TD) {
	// tdiJSON, _ := json.Marshal(tdi)
	// notif := messaging.NewNotificationMessage(wot.OpSubscribeEvent, ThingIDDirectory, UpdateTDEventName, tdiJSON)
	// for _, sink := range m.sinks {
	// 	sink.HandleNotification(notif)
	// }
}

// Start readies the module for use using the given yaml configuration.
// Start must be invoked before passing messages.
//
// yamlConfig contains the settings to use.
func (m *DirectoryClient) Start(yamlConfig string) error {
	config := ModuleConfig{}
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	if err != nil {
		return err
	}

	// if no directory is provided, then do nothing
	if config.DirectoryURL == "" {
		return nil
	}

	err = m.Connect(config.DirectoryURL)
	if err != nil {
		return err
	}

	// the problem is that the WoT directory API is http, does not support events.
	// refresh the TD on startup which notifies sinks
	err = m.ListTD(1000)

	return err
}

// Stop any running actions
func (m *DirectoryClient) Stop() {
}
