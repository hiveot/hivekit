package internal

import (
	"fmt"
	"strings"

	"github.com/hiveot/hivekit/go/modules/digitwin"
)

// Create a digital twin ID from the device client ID, thing ID and
// the digitwin prefix.
// This handles devices that host multiple Things.
func MakeDigitwinID(clientID string, thingID string) string {
	digitwinThingID := fmt.Sprintf("%s%s:%s",
		digitwin.DigitwinIDPrefix, clientID, thingID)
	return digitwinThingID
}

// Split the digital twin ID into the device client ID and the thingID.
// This returns an error if the given ID is not a digitwin ID
func SplitDigitwinID(digitwinID string) (clientID string, thingID string, err error) {
	parts := strings.Split(digitwinID, ":")
	if len(parts) != 3 || !strings.HasPrefix(digitwinID, digitwin.DigitwinIDPrefix) {
		return "", "", fmt.Errorf("The given id '%s' is not a digital twin thingID", digitwinID)
	}
	return parts[1], parts[2], nil
}
