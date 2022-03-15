package api

import "time"

func DeepCopy(in *Application) *Application {
	if in == nil {
		return nil
	}
	newApp := &Application{
		Title:           in.Title,
		Version:         in.Version,
		Company:         in.Company,
		Website:         in.Website,
		Source:          in.Source,
		License:         in.License,
		Description:     in.Description,
		CreateTimeStamp: in.CreateTimeStamp,
		ResourceVersion: in.ResourceVersion,
	}

	if in.DeleteTimeStamp != nil {
		d := &time.Time{}
		*d = *in.DeleteTimeStamp
		newApp.DeleteTimeStamp = d
	}

	if in.Maintainers != nil {
		ms := make([]Maintainer, 0)
		for _, each := range in.Maintainers {
			m := Maintainer{
				Name:  each.Name,
				Email: each.Email,
			}
			ms = append(ms, m)
		}
		newApp.Maintainers = ms
	}
	return newApp
}
