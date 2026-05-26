package addforms

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

// For use in the module factory
const AddFormsModuleType = "addForms"

type IAddFormsService interface {
	modules.IHiveModule

	// AddTDSecForms updates the given TD with base URL, security scheme and forms for affordances
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)
}
