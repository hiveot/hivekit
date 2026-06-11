package addforms

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

// For use in the module factory
const AddFormsModuleType = "addForms"

// AddFormsService modifies TD's sent with directory update and create commands with base, security, and form information from the configured transports.
type IAddFormsService interface {
	modules.IHiveModule

	// AddTDSecForms is updated to update the TD with forms for all configured servers
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)
}
