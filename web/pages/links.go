package pages

import (
	"voxelprismatic/library-management-senior-project/web/common"
)

var managementAsideLinks = common.NavAsideEntryList{
	{Name: "Catalog", Link: "/management"},
	{Name: "Add Books", Link: "/management/books/add"},
	{Name: "Add Lists", Link: "/management/lists/add"},
	{},
	{Name: "View Holds", Link: "/management/holds"},
	{Name: "View Fines", Link: "/management/fines"},
	{Name: "Repair Log", Link: "/management/repair"},
}
