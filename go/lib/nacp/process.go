package nacp

import (
	"fmt"
)

func Process(nacp *Nacp, titleId uint64) error {
	// TODO: these are required
	if nacp.GetTitle() == "" {
		return fmt.Errorf("invalid title name in control.nacp")
	}
	if nacp.GetAuthor() == "" {
		return fmt.Errorf("invalid publisher/author in control.nacp")
	}

	// TODO: these are optional
	nacp.SetLogoHandling(0)

	if titleId != 0 {
		nacp.SetID(titleId)
	}
	return nil
}
