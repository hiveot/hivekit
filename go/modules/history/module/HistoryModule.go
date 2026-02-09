package module

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/msg"
	"go.yaml.in/yaml/v2"
)

// HistoryService provides storage for request and notification history.
//
// Requests received are forwarded to the registered sink and stored if they pass the filter.
// Similarly, notifications are forwarded as-is and stored if they pass the notification filter.
//
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService interface
type HistoryModule struct {
	modules.HiveModuleBase

	// The underlying bucketstore instance
	bucketStore bucketstore.IBucketStore

	// Storage of the latest properties of a things
	//propsStore *LatestPropertiesStore
	// the manage history sub-service
	// manageHistSvc *server.ManageHistory
	// the read-history sub-service
	// readHistSvc *server.ReadHistory

	config history.HistoryConfig

	// the messaging agent used to pubsub service to subscribe to event
	// ag *agent.Agent

	// optional handling of pubsub events. nil if not used
	//subEventHandler *PubSubEventHandler

	// backend
	historyStore *HistoryStore
}

// Forward notifications to the registered sink and store if they pass the filter.
func (m *HistoryModule) HandleNotification(notif *msg.NotificationMessage) {
	if m.config.NotificationFilter != nil {
		go func() {
			if m.config.NotificationFilter.RetainNotification(notif) {
				m.StoreNotification(notif)
			}
		}()
	}
	m.ForwardNotification(notif)
}

// Forward requests to the registered sink and store if they pass the filter.
func (m *HistoryModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) {
	if m.config.NotificationFilter != nil {
		go func() {
			if m.config.RequestFilter.RetainRequest(req) {
				m.StoreRequest(req)
			}
		}()
	}
	m.ForwardRequest(req, replyTo)
}

// Start the history service
// this loads the filters
func (svc *HistoryModule) Start(yamlConfig string) (err error) {

	slog.Info("Starting HistoryService", "moduleID", svc.GetModuleID())
	err = yaml.Unmarshal([]byte(yamlConfig), &svc.config)
	if err != nil {
		slog.Error("Failed to load history service config", "error", err)
		return err
	}

	// setup
	// svc.ag = ag
	// svc.agentID = ag.GetClientID()
	// svc.manageHistSvc = NewManageHistory(nil)
	// err = svc.manageHistSvc.Start()
	// if err == nil {
	// 	svc.readHistSvc = NewReadHistory(svc.bucketStore)
	// 	err = svc.readHistSvc.Start()
	// }
	if err != nil {
		return err
	}

	// Set the required permissions for using this service
	// any user roles can view the history
	// permissions := authz.ThingPermissions{
	// 	AgentID: ag.GetClientID(),
	// 	ThingID: historyapi.ReadHistoryServiceID,
	// 	Deny:    []authz.ClientRole{authz.ClientRoleNone},
	// }
	// err = authz.UserSetPermissions(ag.Consumer, permissions)

	//if err == nil {
	//	// only admin role can manage the history
	//	err = myProfile.SetServicePermissions(historyapi.ManageHistoryThingID, []string{api.ClientRoleAdmin})
	//}

	// // subscribe to events to add to the history store
	// if err == nil && svc.ag != nil {

	// 	// handler of adding events to the history
	// 	svc.addHistory = NewAddHistory(svc.bucketStore, svc.manageHistSvc)

	// 	// register the history service methods and listen for requests
	// 	StartHistoryAgent(svc, svc.ag)

	// 	// TODO: add actions to the history, filtered through retention manager
	// 	// subscribe to receive the events to add to the history, filtered through the retention manager
	// 	err = svc.ag.Subscribe("", "")
	// 	err = svc.ag.ObserveProperty("", "")
	// }

	return err
}

// Stop using the history service and release resources
func (svc *HistoryModule) Stop() {
	slog.Info("Stopping HistoryService")
	// if svc.readHistSvc != nil {
	// 	svc.readHistSvc.Stop()
	// 	svc.readHistSvc = nil
	// }
	// if svc.manageHistSvc != nil {
	// 	svc.manageHistSvc.Stop()
	// 	svc.manageHistSvc = nil
	// }
	_ = svc.bucketStore.Close()
}

// Store notifications for later retrieval
func (m *HistoryModule) StoreNotification(notif *msg.NotificationMessage) {
}

// Store notifications for later retrieval
func (m *HistoryModule) StoreRequest(req *msg.RequestMessage) {
}

// NewHistoryService creates a new instance for the history service using the given
// storage bucket.
//
//	config optional configuration or nil to use defaults
//	store contains an opened bucket store to use. This will be closed on Stop.
//	hc connection with the hub
func NewHistoryService(bucketStore bucketstore.IBucketStore) *HistoryModule {

	historyStore := NewHistoryStore(bucketStore)
	svc := &HistoryModule{
		bucketStore:  bucketStore,
		historyStore: historyStore,
	}
	return svc
}
