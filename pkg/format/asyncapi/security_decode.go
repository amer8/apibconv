package asyncapi

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	apierrors "github.com/amer8/apibconv/pkg/errors"
	"github.com/amer8/apibconv/pkg/model"
	"go.yaml.in/yaml/v3"
)

type inputSecurityScheme struct {
	Ref              string           `yaml:"$ref"`
	Type             string           `yaml:"type"`
	Description      string           `yaml:"description"`
	Name             string           `yaml:"name"`
	In               string           `yaml:"in"`
	Scheme           string           `yaml:"scheme"`
	BearerFormat     string           `yaml:"bearerFormat"`
	OpenIDConnectURL string           `yaml:"openIdConnectUrl"`
	Scopes           []string         `yaml:"scopes"`
	Flows            *inputOAuthFlows `yaml:"flows"`
}

type inputOAuthFlows struct {
	Implicit          *inputOAuthFlow `yaml:"implicit"`
	Password          *inputOAuthFlow `yaml:"password"`
	ClientCredentials *inputOAuthFlow `yaml:"clientCredentials"`
	AuthorizationCode *inputOAuthFlow `yaml:"authorizationCode"`
}

type inputOAuthFlow struct {
	AuthorizationURL string            `yaml:"authorizationUrl"`
	TokenURL         string            `yaml:"tokenUrl"`
	RefreshURL       string            `yaml:"refreshUrl"`
	Scopes           map[string]string `yaml:"scopes"`
	AvailableScopes  map[string]string `yaml:"availableScopes"`
}

type securityDecoder struct {
	api     *model.API
	schemes map[string]interface{}
	traits  map[string]interface{}
	v3      bool
}

func newSecurityDecoder(api *model.API, components map[string]interface{}, v3 bool) *securityDecoder {
	d := &securityDecoder{api: api, v3: v3}
	if schemes, ok := components["securitySchemes"].(map[string]interface{}); ok {
		d.schemes = schemes
	}
	if traits, ok := components["operationTraits"].(map[string]interface{}); ok {
		d.traits = traits
	}
	// Keep supported definitions, including those not currently referenced.
	for _, name := range sortedSchemeNames(d.schemes) {
		if scheme, _, err := d.resolve(name, make(map[string]bool)); err == nil {
			api.Components.SecuritySchemes[name] = scheme
		}
	}
	return d
}

func sortedSchemeNames(schemes map[string]interface{}) []string {
	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func securityInputError(reason string) error {
	return fmt.Errorf("%w in AsyncAPI input: %s", apierrors.ErrUnsupportedSecurity, reason)
}

func (d *securityDecoder) resolveTrait(trait map[string]interface{}, seen map[string]bool) (map[string]interface{}, error) {
	ref, ok := trait["$ref"]
	if !ok {
		return trait, nil
	}
	refString, ok := ref.(string)
	const prefix = "#/components/operationTraits/"
	if !ok || !strings.HasPrefix(refString, prefix) {
		return nil, securityInputError("external operation trait cannot be resolved")
	}
	name, err := decodePointerToken(strings.TrimPrefix(refString, prefix))
	if err != nil {
		return nil, err
	}
	if seen[name] {
		return nil, securityInputError("cyclic operation trait reference")
	}
	seen[name] = true
	raw, ok := d.traits[name]
	if !ok {
		return nil, securityInputError(fmt.Sprintf("operation trait %q is not defined", name))
	}
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, securityInputError(err.Error())
	}
	var resolved map[string]interface{}
	if err := yaml.Unmarshal(data, &resolved); err != nil {
		return nil, securityInputError(err.Error())
	}
	return d.resolveTrait(resolved, seen)
}

func (d *securityDecoder) operationSecurityV2(op *Operation) ([]model.SecurityRequirement, error) {
	requirements := op.Security
	for _, trait := range op.Traits {
		resolved, err := d.resolveTrait(trait, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		if raw, ok := resolved["security"]; ok {
			data, err := yaml.Marshal(raw)
			if err != nil {
				return nil, securityInputError(err.Error())
			}
			if err := yaml.Unmarshal(data, &requirements); err != nil {
				return nil, securityInputError(err.Error())
			}
		}
	}
	return d.requirementsV2(requirements)
}

func (d *securityDecoder) operationSecurityV3(op *Operation3) ([]model.SecurityRequirement, error) {
	requirements := op.Security
	for _, trait := range op.Traits {
		if requirements != nil {
			break // Traits cannot override security already present on the operation.
		}
		resolved, err := d.resolveTrait(trait, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		if raw, ok := resolved["security"]; ok {
			data, err := yaml.Marshal(raw)
			if err != nil {
				return nil, securityInputError(err.Error())
			}
			if err := yaml.Unmarshal(data, &requirements); err != nil {
				return nil, securityInputError(err.Error())
			}
		}
	}
	return d.requirementsV3(requirements)
}

func (d *securityDecoder) resolve(name string, seen map[string]bool) (model.SecurityScheme, []string, error) {
	if seen[name] {
		return model.SecurityScheme{}, nil, securityInputError("cyclic security scheme reference")
	}
	seen[name] = true
	raw, ok := d.schemes[name]
	if !ok {
		return model.SecurityScheme{}, nil, securityInputError(fmt.Sprintf("security scheme %q is not defined", name))
	}
	return d.decode(raw, seen)
}

func (d *securityDecoder) decode(raw interface{}, seen map[string]bool) (model.SecurityScheme, []string, error) {
	data, err := yaml.Marshal(raw)
	if err != nil {
		return model.SecurityScheme{}, nil, securityInputError(err.Error())
	}
	var s inputSecurityScheme
	if err := yaml.Unmarshal(data, &s); err != nil {
		return model.SecurityScheme{}, nil, securityInputError(err.Error())
	}
	if s.Ref != "" {
		const prefix = "#/components/securitySchemes/"
		if !strings.HasPrefix(s.Ref, prefix) {
			return model.SecurityScheme{}, nil, securityInputError(fmt.Sprintf("external security reference %q cannot be resolved", s.Ref))
		}
		name, err := decodePointerToken(strings.TrimPrefix(s.Ref, prefix))
		if err != nil {
			return model.SecurityScheme{}, nil, err
		}
		return d.resolve(name, seen)
	}
	result := model.SecurityScheme{Description: s.Description, Name: s.Name, In: s.In, Scheme: s.Scheme, BearerFormat: s.BearerFormat, OpenIDConnectURL: s.OpenIDConnectURL}
	switch s.Type {
	case "httpApiKey":
		result.Type = model.SecurityTypeAPIKey
	case "apiKey":
		if s.In != "user" && s.In != "password" {
			return result, nil, securityInputError("invalid AsyncAPI API key location")
		}
		result.Type = model.SecurityTypeAsyncAPIKey
	case "http":
		result.Type = model.SecurityTypeHTTP
	case "oauth2":
		result.Type = model.SecurityTypeOAuth2
		if s.Flows == nil {
			return result, nil, securityInputError("OAuth2 scheme has no flows")
		}
		result.Flows = &model.OAuthFlows{Implicit: d.flow(s.Flows.Implicit), Password: d.flow(s.Flows.Password), ClientCredentials: d.flow(s.Flows.ClientCredentials), AuthorizationCode: d.flow(s.Flows.AuthorizationCode)}
	case "openIdConnect":
		result.Type = model.SecurityTypeOpenIDConnect
	case "userPassword", "X509", "symmetricEncryption", "asymmetricEncryption", "plain", "scramSha256", "scramSha512", "gssapi":
		result.Type = model.SecuritySchemeType(s.Type)
	default:
		return result, nil, securityInputError(fmt.Sprintf("security scheme type %q cannot be represented", s.Type))
	}
	if len(s.Scopes) > 0 && result.Type != model.SecurityTypeOAuth2 && result.Type != model.SecurityTypeOpenIDConnect {
		return result, nil, securityInputError("scopes on a non-OAuth security scheme")
	}
	return result, s.Scopes, nil
}

func (d *securityDecoder) flow(f *inputOAuthFlow) *model.OAuthFlow {
	if f == nil {
		return nil
	}
	scopes := f.Scopes
	if d.v3 {
		scopes = f.AvailableScopes
	}
	return &model.OAuthFlow{AuthorizationURL: f.AuthorizationURL, TokenURL: f.TokenURL, RefreshURL: f.RefreshURL, Scopes: scopes}
}

func (d *securityDecoder) requirementsV2(reqs []map[string][]string) ([]model.SecurityRequirement, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	result := make([]model.SecurityRequirement, 0, len(reqs))
	for _, req := range reqs {
		out := make(model.SecurityRequirement, len(req))
		for name, scopes := range req {
			scheme, _, err := d.resolve(name, make(map[string]bool))
			if err != nil {
				return nil, err
			}
			if len(scopes) > 0 && scheme.Type != model.SecurityTypeOAuth2 && scheme.Type != model.SecurityTypeOpenIDConnect {
				return nil, securityInputError(fmt.Sprintf("scheme %q cannot require scopes", name))
			}
			d.api.Components.SecuritySchemes[name] = scheme
			out[name] = scopes
		}
		result = append(result, out)
	}
	return result, nil
}

func (d *securityDecoder) requirementsV3(reqs []map[string]interface{}) ([]model.SecurityRequirement, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	result := make([]model.SecurityRequirement, 0, len(reqs))
	for _, req := range reqs {
		scheme, scopes, err := d.decode(req, make(map[string]bool))
		if err != nil {
			return nil, err
		}
		name := ""
		if ref, ok := req["$ref"].(string); ok && strings.HasPrefix(ref, "#/components/securitySchemes/") {
			name, err = decodePointerToken(strings.TrimPrefix(ref, "#/components/securitySchemes/"))
			if err != nil {
				return nil, err
			}
		}
		if name == "" {
			for _, candidate := range sortedModelSchemeNames(d.api.Components.SecuritySchemes) {
				if reflect.DeepEqual(d.api.Components.SecuritySchemes[candidate], scheme) {
					name = candidate
					break
				}
			}
		}
		if name == "" {
			for i := 1; ; i++ {
				candidate := fmt.Sprintf("asyncapiSecurity%d", i)
				_, reserved := d.schemes[candidate]
				_, used := d.api.Components.SecuritySchemes[candidate]
				if !reserved && !used {
					name = candidate
					break
				}
			}
		}
		d.api.Components.SecuritySchemes[name] = scheme
		if scopes == nil {
			scopes = []string{}
		}
		result = append(result, model.SecurityRequirement{name: scopes})
	}
	return result, nil
}

func sortedModelSchemeNames(schemes map[string]model.SecurityScheme) []string {
	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func decodePointerToken(token string) (string, error) {
	if strings.Contains(token, "/") || strings.Contains(strings.ReplaceAll(strings.ReplaceAll(token, "~0", ""), "~1", ""), "~") {
		return "", securityInputError(fmt.Sprintf("invalid reference token %q", token))
	}
	return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~"), nil
}

func uniformServerSecurity(policies [][]model.SecurityRequirement) ([]model.SecurityRequirement, error) {
	if len(policies) == 0 {
		return nil, nil
	}
	for _, policy := range policies[1:] {
		if !reflect.DeepEqual(policy, policies[0]) {
			return nil, securityInputError("different server security policies cannot be represented")
		}
	}
	return policies[0], nil
}

func channelServerSecurity(names []string, policies map[string][]model.SecurityRequirement) ([]model.SecurityRequirement, error) {
	if len(names) == 0 {
		names = make([]string, 0, len(policies))
		for name := range policies {
			names = append(names, name)
		}
		sort.Strings(names)
	}
	selected := make([][]model.SecurityRequirement, 0, len(names))
	for _, name := range names {
		policy, ok := policies[name]
		if !ok {
			return nil, securityInputError(fmt.Sprintf("channel references unknown server %q", name))
		}
		selected = append(selected, policy)
	}
	return uniformServerSecurity(selected)
}

func hasSecuredServer(policies map[string][]model.SecurityRequirement) bool {
	for _, policy := range policies {
		if hasActualRequirement(policy) {
			return true
		}
	}
	return false
}

func channelV3ServerNames(servers []map[string]interface{}) ([]string, error) {
	names := make([]string, 0, len(servers))
	for _, server := range servers {
		ref, ok := server["$ref"].(string)
		const prefix = "#/servers/"
		if !ok || !strings.HasPrefix(ref, prefix) {
			return nil, securityInputError("channel server reference cannot be resolved")
		}
		name, err := decodePointerToken(strings.TrimPrefix(ref, prefix))
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}
