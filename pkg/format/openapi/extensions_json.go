package openapi

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Info) MarshalJSON() ([]byte, error) {
	type plain Info
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value Contact) MarshalJSON() ([]byte, error) {
	type plain Contact
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value License) MarshalJSON() ([]byte, error) {
	type plain License
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value Server) MarshalJSON() ([]byte, error) {
	type plain Server
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value ServerVariable) MarshalJSON() ([]byte, error) {
	type plain ServerVariable
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value PathItem) MarshalJSON() ([]byte, error) {
	type plain PathItem
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value PathItemV2) MarshalJSON() ([]byte, error) {
	type plain PathItemV2
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Parameter) MarshalJSON() ([]byte, error) {
	type plain Parameter
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value RequestBody) MarshalJSON() ([]byte, error) {
	type plain RequestBody
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value Response) MarshalJSON() ([]byte, error) {
	type plain Response
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value MediaType) MarshalJSON() ([]byte, error) {
	type plain MediaType
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Schema) MarshalJSON() ([]byte, error) {
	type plain Schema
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Components) MarshalJSON() ([]byte, error) {
	type plain Components
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Header) MarshalJSON() ([]byte, error) {
	type plain Header
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value SecurityScheme) MarshalJSON() ([]byte, error) {
	type plain SecurityScheme
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value OAuthFlows) MarshalJSON() ([]byte, error) {
	type plain OAuthFlows
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value OAuthFlow) MarshalJSON() ([]byte, error) {
	type plain OAuthFlow
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value Tag) MarshalJSON() ([]byte, error) {
	type plain Tag
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value ExternalDocs) MarshalJSON() ([]byte, error) {
	type plain ExternalDocs
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value Encoding) MarshalJSON() ([]byte, error) {
	type plain Encoding
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Example) MarshalJSON() ([]byte, error) {
	type plain Example
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
//
//nolint:gocritic // Value receivers preserve marshaling of map values.
func (value Link) MarshalJSON() ([]byte, error) {
	type plain Link
	return marshalExtensionsJSON(plain(value), value.Extensions)
}

// MarshalJSON includes vendor extensions.
func (value Discriminator) MarshalJSON() ([]byte, error) {
	type plain Discriminator
	return marshalExtensionsJSON(plain(value), value.Extensions)
}
