package module

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/authn"
)

func (m *AuthnModule) AddClient(
	clientID string, displayName string, role authn.ClientRole, pubKeyPem string) error {

	_, err := m.authnStore.GetProfile(clientID)
	if err == nil {
		return fmt.Errorf("Account for client '%s' already exists", clientID)
	}

	newProfile := authn.ClientProfile{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
		PubKeyPem:   pubKeyPem,
	}
	return m.authnStore.Add(newProfile)
}

// // Return the authenticator for use by other modules
// func (m *AuthnModule) GetAuthenticator() transports.IAuthValidator {
// 	return m.authenticator
// }

// // GetProfile return the client's profile
// func (m *AuthnModule) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
// 	return m.authnStore.GetProfile(clientID)
// }

// // GetProfile return a list of client profiles
// func (m *AuthnModule) GetProfiles() (profiles []authn.ClientProfile, err error) {
// 	return m.authnStore.GetProfiles()
// }

// Login verifies the password and generates a new limited authentication token
//
// This uses the configured session authenticator.
// func (m *AuthnModule) Login(clientID string, password string) (
// 	newToken string, validUntil time.Time, err error) {

// 	// the module uses the configured authenticator
// 	newToken, validUntil, err = m.authenticator.Login(clientID, password)
// 	return newToken, validUntil, err
// }

// // Logout disables the client's sessions
// //
// // This uses the configured session authenticator.
// func (m *AuthnModule) Logout(clientID string) {

// 	// the module uses the configured authenticator
// 	m.authenticator.Logout(clientID)
// }

// RefreshToken refreshes the auth token using the session authenticator.
//
// This uses the configured session authenticator.
// func (m *AuthnModule) RefreshToken(clientID, oldToken string) (
// 	newToken string, validUntil time.Time, err error) {

// 	newToken, validUntil, err = m.authenticator.RefreshToken(clientID, oldToken)
// 	return newToken, validUntil, err
// }

// // Remove a client
// func (m *AuthnModule) RemoveClient(clientID string) error {
// 	return m.authnStore.Remove(clientID)
// }

// // Change the role of a client
// func (m *AuthnModule) SetRole(clientID string, role string) error {
// 	return m.authnStore.SetRole(clientID, role)
// }

// Change the password of a client
// func (m *AuthnModule) SetPassword(clientID string, password string) error {
// 	return m.authenticator.SetPassword(clientID, password)
// }

// // UpdateProfile update the client profile
// // only administrators are allowed to update the role
// func (m *AuthnModule) UpdateProfile(senderID string, newProfile authn.ClientProfile) error {
// 	senderProf, err := m.authnStore.GetProfile(senderID)
// 	if err != nil {
// 		return fmt.Errorf("Unknown sender '%s'", senderID)
// 	}
// 	clientProf, err := m.authnStore.GetProfile(newProfile.ClientID)
// 	if err != nil {
// 		return fmt.Errorf("Unknown client '%s'", newProfile.ClientID)
// 	}
// 	if senderID != newProfile.ClientID {
// 		// only admin roles can update client profiles
// 		if senderProf.Role != authn.ClientRoleAdmin && senderProf.Role != authn.ClientRoleService {
// 			return fmt.Errorf("Sender '%s' is not admin, not allowed to update profile", senderID)
// 		}
// 	} else {
// 		// client cannot change its own role
// 		if newProfile.Role != "" && newProfile.Role != clientProf.Role {
// 			return fmt.Errorf("Client '%s' is not allowed to change its role", senderID)
// 		}
// 	}
// 	return m.authnStore.UpdateProfile(newProfile)
// }
