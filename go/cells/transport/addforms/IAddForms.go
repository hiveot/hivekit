package addforms

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
)

// For use in the cell factory
const AddFormsCellType = "addForms"

// AddFormsService modifies TD's sent with directory update and create commands with base, security, and form information from the configured transports.
type IAddFormsService interface {
	api.IHiveCell

	// AddTDSecForms is updated to update the TD with forms for all configured servers
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)
}
