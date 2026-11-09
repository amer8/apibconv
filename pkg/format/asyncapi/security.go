package asyncapi

import (
	"fmt"
	"strings"

	"github.com/amer8/apibconv/internal/detect"
	"github.com/amer8/apibconv/internal/security"
	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/model"
)

type securityPlan struct {
	schemes  map[string]interface{}
	v2       map[*model.Operation][]map[string][]string
	v3       map[*model.Operation][]map[string]interface{}
	serverV2 map[int][]map[string][]string
	serverV3 map[int][]map[string]interface{}
}

func unsupportedSecurity(reason string) error {
	return fmt.Errorf("%w in AsyncAPI output: %s", apierrors.ErrUnsupportedSecurity, reason)
}

func prepareSecurity(api *model.API, v3 bool, protocol string) (*securityPlan, error) {
	plan := &securityPlan{schemes: make(map[string]interface{}), v2: make(map[*model.Operation][]map[string][]string), v3: make(map[*model.Operation][]map[string]interface{}), serverV2: make(map[int][]map[string][]string), serverV3: make(map[int][]map[string]interface{})}
	if err := checkOmittedSecurity(api); err != nil {
		return nil, err
	}

	// Preserve every representable definition, even if it is currently unused.
	for name := range api.Components.SecuritySchemes {
		scheme, err := resolveSecurityScheme(api, name)
		if err != nil {
			continue
		}
		if converted, err := convertSecurityScheme(&scheme, v3); err == nil {
			plan.schemes[name] = converted
		}
	}
	for i := range api.Servers {
		server := api.Servers[i]
		scope := *api
		scope.Servers = []model.Server{server}
		op := &model.Operation{}
		if err := plan.addRequirements(&scope, op, server.Security, v3, protocol); err != nil {
			return nil, fmt.Errorf("server %q: %w", server.Name, err)
		}
		plan.serverV2[i] = plan.v2[op]
		plan.serverV3[i] = plan.v3[op]
		delete(plan.v2, op)
		delete(plan.v3, op)
	}
	operationIDs := make(map[string]bool)
	channelIDs := make(map[string]bool)
	securityPresent := false
	for _, server := range api.Servers {
		if hasActualRequirement(server.Security) {
			securityPresent = true
		}
	}
	for path := range api.Paths {
		item := api.Paths[path]
		for _, op := range []*model.Operation{item.Get, item.Post} {
			if op != nil && hasActualRequirement(effectiveSecurity(api.Security, op.Security)) {
				securityPresent = true
			}
		}
	}
	for path := range api.Paths {
		item := api.Paths[path]
		cleanPath := path
		if i := strings.Index(cleanPath, "{?"); i >= 0 {
			cleanPath = cleanPath[:i]
		}
		if i := strings.Index(cleanPath, "#"); i >= 0 {
			cleanPath = cleanPath[:i]
		}
		channelID := cleanPath
		if v3 && item.Name != "" {
			channelID = item.Name
		}
		if item.Get == nil && item.Post == nil {
			continue
		}
		if channelIDs[channelID] && securityPresent {
			return nil, unsupportedSecurity("multiple paths map to the same channel")
		}
		channelIDs[channelID] = true
		for _, entry := range []struct {
			op     *model.Operation
			prefix string
		}{{item.Post, "send_"}, {item.Get, "receive_"}} {
			if entry.op == nil {
				continue
			}
			if v3 {
				id := entry.op.OperationID
				if id == "" {
					id = entry.prefix + channelID
				}
				if operationIDs[id] && securityPresent {
					return nil, unsupportedSecurity("multiple operations map to the same operation ID")
				}
				operationIDs[id] = true
			}
			reqs := effectiveSecurity(api.Security, entry.op.Security)
			if err := plan.addRequirements(api, entry.op, reqs, v3, protocol); err != nil {
				return nil, err
			}
		}
	}
	return plan, nil
}

// checkOmittedSecurity rejects requirements on operations the writer cannot emit.
func checkOmittedSecurity(api *model.API) error {
	remaining := *api
	remaining.Components.SecuritySchemes = nil
	remaining.Servers = nil // Server security is handled directly by the AsyncAPI writer.
	remaining.Paths = make(map[string]model.PathItem, len(api.Paths))
	for path := range api.Paths {
		item := api.Paths[path]
		if item.Get != nil {
			copyOp := *item.Get
			copyOp.Security = []model.SecurityRequirement{}
			item.Get = &copyOp
		}
		if item.Post != nil {
			copyOp := *item.Post
			copyOp.Security = []model.SecurityRequirement{}
			item.Post = &copyOp
		}
		remaining.Paths[path] = item
	}
	return security.CheckUnsupported(&remaining, "AsyncAPI")
}

func (plan *securityPlan) addRequirements(api *model.API, op *model.Operation, reqs []model.SecurityRequirement, v3 bool, protocol string) error {
	for _, req := range reqs {
		if v3 && len(req) == 0 && len(reqs) == 1 {
			continue
		}
		if v3 && len(req) != 1 {
			return unsupportedSecurity("AsyncAPI 3 cannot express combined or anonymous security alternatives")
		}
		v2req := make(map[string][]string, len(req))
		for name, scopes := range req {
			scheme, err := resolveSecurityScheme(api, name)
			if err != nil {
				return fmt.Errorf("%w: scheme %q: %v", apierrors.ErrUnsupportedSecurity, name, err)
			}
			if err := checkSecurityProtocol(api, &scheme, name, protocol); err != nil {
				return err
			}
			converted, err := convertSecurityScheme(&scheme, v3)
			if err != nil {
				return fmt.Errorf("%w: scheme %q: %v", apierrors.ErrUnsupportedSecurity, name, err)
			}
			if len(scopes) > 0 && scheme.Type != model.SecurityTypeOAuth2 && scheme.Type != model.SecurityTypeOpenIDConnect {
				return unsupportedSecurity(fmt.Sprintf("scheme %q cannot require scopes", name))
			}
			if scopes == nil {
				scopes = []string{}
			}
			v2req[name] = scopes
			if v3 {
				if scheme.Type == model.SecurityTypeOAuth2 || scheme.Type == model.SecurityTypeOpenIDConnect {
					converted["scopes"] = scopes
				}
				plan.v3[op] = append(plan.v3[op], converted)
			}
		}
		if !v3 {
			plan.v2[op] = append(plan.v2[op], v2req)
		}
	}
	return nil
}

func resolveSecurityScheme(api *model.API, name string) (model.SecurityScheme, error) {
	seen := make(map[string]bool)
	for {
		if seen[name] {
			return model.SecurityScheme{}, fmt.Errorf("cyclic security scheme reference")
		}
		seen[name] = true
		scheme, ok := api.Components.SecuritySchemes[name]
		if !ok {
			return model.SecurityScheme{}, fmt.Errorf("security scheme %q is not defined", name)
		}
		if scheme.Ref == "" {
			return scheme, nil
		}
		const prefix = "#/components/securitySchemes/"
		if !strings.HasPrefix(scheme.Ref, prefix) {
			return model.SecurityScheme{}, fmt.Errorf("external security scheme reference %q cannot be resolved", scheme.Ref)
		}
		name = strings.TrimPrefix(scheme.Ref, prefix)
		if strings.Contains(strings.ReplaceAll(strings.ReplaceAll(name, "~0", ""), "~1", ""), "~") || strings.Contains(name, "/") {
			return model.SecurityScheme{}, fmt.Errorf("invalid security scheme reference %q", scheme.Ref)
		}
		name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
	}
}

func checkSecurityProtocol(api *model.API, scheme *model.SecurityScheme, name, protocol string) error {
	if scheme.Type != model.SecurityTypeAPIKey && scheme.Type != model.SecurityTypeHTTP {
		return nil
	}
	if len(api.Servers) == 0 && !httpCompatibleProtocol(protocol) {
		return unsupportedSecurity(fmt.Sprintf("HTTP security scheme %q is incompatible with protocol %q", name, protocol))
	}
	for _, server := range api.Servers {
		serverProtocol := server.Protocol
		if serverProtocol == "" {
			serverProtocol = protocol
			if serverProtocol == "" || serverProtocol == "auto" {
				serverProtocol = detect.Protocol(server.URL)
			}
		}
		if !httpCompatibleProtocol(serverProtocol) {
			return unsupportedSecurity(fmt.Sprintf("HTTP security scheme %q is incompatible with server protocol %q", name, serverProtocol))
		}
	}
	return nil
}

func httpCompatibleProtocol(protocol string) bool {
	switch strings.ToLower(protocol) {
	case "", "auto", "http", "https", "ws", "wss":
		return true
	default:
		return false
	}
}

func effectiveSecurity(global, operation []model.SecurityRequirement) []model.SecurityRequirement {
	if operation != nil {
		return operation
	}
	return global
}

func hasActualRequirement(requirements []model.SecurityRequirement) bool {
	for _, requirement := range requirements {
		if len(requirement) > 0 {
			return true
		}
	}
	return false
}

func convertSecurityScheme(s *model.SecurityScheme, v3 bool) (map[string]interface{}, error) {
	if s.Ref != "" {
		return nil, fmt.Errorf("security scheme references cannot be resolved")
	}
	out := make(map[string]interface{})
	if s.Description != "" {
		out["description"] = s.Description
	}
	switch s.Type {
	case model.SecurityTypeAPIKey:
		if s.Name == "" || (s.In != "header" && s.In != "query" && s.In != "cookie") {
			return nil, fmt.Errorf("invalid HTTP API key")
		}
		out["type"], out["name"], out["in"] = "httpApiKey", s.Name, s.In
	case model.SecurityTypeAsyncAPIKey:
		if s.In != "user" && s.In != "password" {
			return nil, fmt.Errorf("invalid AsyncAPI API key location")
		}
		out["type"], out["in"] = "apiKey", s.In
	case model.SecurityTypeHTTP:
		if s.Scheme == "" {
			return nil, fmt.Errorf("missing HTTP authentication scheme")
		}
		out["type"], out["scheme"] = "http", s.Scheme
		if s.BearerFormat != "" {
			out["bearerFormat"] = s.BearerFormat
		}
	case model.SecurityTypeOpenIDConnect:
		if s.OpenIDConnectURL == "" {
			return nil, fmt.Errorf("missing OpenID Connect URL")
		}
		out["type"], out["openIdConnectUrl"] = "openIdConnect", s.OpenIDConnectURL
	case "userPassword", "X509", "symmetricEncryption", "asymmetricEncryption", "plain", "scramSha256", "scramSha512", "gssapi":
		out["type"] = string(s.Type)
	case model.SecurityTypeOAuth2:
		if s.Flows == nil {
			return nil, fmt.Errorf("missing OAuth flows")
		}
		flows := make(map[string]interface{})
		for _, f := range []struct {
			name string
			flow *model.OAuthFlow
		}{{"implicit", s.Flows.Implicit}, {"password", s.Flows.Password}, {"clientCredentials", s.Flows.ClientCredentials}, {"authorizationCode", s.Flows.AuthorizationCode}} {
			if f.flow == nil {
				continue
			}
			if (f.name == "implicit" || f.name == "authorizationCode") && f.flow.AuthorizationURL == "" {
				return nil, fmt.Errorf("OAuth flow %q requires authorizationUrl", f.name)
			}
			if f.name != "implicit" && f.flow.TokenURL == "" {
				return nil, fmt.Errorf("OAuth flow %q requires tokenUrl", f.name)
			}
			if (f.name == "implicit" && f.flow.TokenURL != "") || ((f.name == "password" || f.name == "clientCredentials") && f.flow.AuthorizationURL != "") {
				return nil, fmt.Errorf("OAuth flow %q has fields not supported by AsyncAPI", f.name)
			}
			fields := make(map[string]interface{})
			if f.flow.AuthorizationURL != "" {
				fields["authorizationUrl"] = f.flow.AuthorizationURL
			}
			if f.flow.TokenURL != "" {
				fields["tokenUrl"] = f.flow.TokenURL
			}
			if f.flow.RefreshURL != "" {
				fields["refreshUrl"] = f.flow.RefreshURL
			}
			scopes := f.flow.Scopes
			if scopes == nil {
				scopes = map[string]string{}
			}
			if v3 {
				fields["availableScopes"] = scopes
			} else {
				fields["scopes"] = scopes
			}
			flows[f.name] = fields
		}
		if len(flows) == 0 {
			return nil, fmt.Errorf("missing OAuth flows")
		}
		out["type"], out["flows"] = "oauth2", flows
	default:
		return nil, fmt.Errorf("security scheme type %q is not supported", s.Type)
	}
	return out, nil
}
