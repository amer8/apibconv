// Package security provides checks for security information in the API model.
package security

import (
	"fmt"

	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/model"
)

// CheckUnsupported rejects effective security requirements a target writer cannot represent.
func CheckUnsupported(api *model.API, target string) error {
	if hasSecurity(api) {
		return fmt.Errorf("%w in %s output", apierrors.ErrUnsupportedSecurity, target)
	}
	return nil
}

// CheckServerUnsupported rejects server-level security not representable by a target.
func CheckServerUnsupported(api *model.API, target string) error {
	for _, server := range api.Servers {
		for _, requirement := range server.Security {
			if len(requirement) > 0 {
				return fmt.Errorf("%w in %s output: server security cannot be represented", apierrors.ErrUnsupportedSecurity, target)
			}
		}
	}
	return nil
}

func hasSecurity(api *model.API) bool {
	if CheckServerUnsupported(api, "target") != nil {
		return true
	}
	seen := make(map[*model.Operation]bool)
	for path := range api.Paths {
		item := api.Paths[path]
		if pathHasSecurity(&item, api.Security, seen) {
			return true
		}
	}
	for path := range api.Webhooks {
		item := api.Webhooks[path]
		if pathHasSecurity(&item, api.Security, seen) {
			return true
		}
	}
	for _, callback := range api.Components.Callbacks {
		if callbackHasSecurity(callback, api.Security, seen) {
			return true
		}
	}
	return false
}

func callbackHasSecurity(callback model.Callback, global []model.SecurityRequirement, seen map[*model.Operation]bool) bool {
	for path := range callback {
		item := callback[path]
		if pathHasSecurity(&item, global, seen) {
			return true
		}
	}
	return false
}

func pathHasSecurity(item *model.PathItem, global []model.SecurityRequirement, seen map[*model.Operation]bool) bool {
	for _, operation := range []*model.Operation{
		item.Get, item.Put, item.Post, item.Delete,
		item.Options, item.Head, item.Patch, item.Trace,
	} {
		if operation == nil || seen[operation] {
			continue
		}
		seen[operation] = true
		requirements := operation.Security
		if requirements == nil {
			requirements = global
		}
		for _, requirement := range requirements {
			if len(requirement) > 0 {
				return true
			}
		}
		for _, callback := range operation.Callbacks {
			if callbackHasSecurity(callback, global, seen) {
				return true
			}
		}
	}
	return false
}
