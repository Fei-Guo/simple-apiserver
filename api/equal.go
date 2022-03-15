package api

import "sort"

func Equal(a *Application, b *Application) bool {
	if a.Title == b.Title && a.Version == b.Version && a.Company == b.Company && a.Website == b.Website && a.Source == b.Source && a.License == b.License && a.Description == b.Description {
		// check maintainers
		if len(a.Maintainers) != len(b.Maintainers) {
			return false
		} else if len(a.Maintainers) == 0 {
			return true
		} else {
			// make sure the contents are equal even if the order is different
			sort.Slice(a.Maintainers, func(i, j int) bool {
				if a.Maintainers[i].Name == a.Maintainers[j].Name {
					return a.Maintainers[i].Email < a.Maintainers[j].Email
				}
				return a.Maintainers[i].Name < a.Maintainers[j].Name
			})

			sort.Slice(b.Maintainers, func(i, j int) bool {
				if b.Maintainers[i].Name == b.Maintainers[j].Name {
					return b.Maintainers[i].Email < b.Maintainers[j].Email
				}
				return b.Maintainers[i].Name < b.Maintainers[j].Name
			})

			for i := 0; i < len(a.Maintainers); i++ {
				if a.Maintainers[i].Name != b.Maintainers[i].Name || a.Maintainers[i].Email != b.Maintainers[i].Email {
					return false
				}
			}
			return true
		}
	} else {
		return false
	}
}
