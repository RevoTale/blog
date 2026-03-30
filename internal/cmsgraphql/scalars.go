package gql

import "encoding/json"

// JSONObject preserves arbitrary JSON object values when fields are bound to the GraphQL JSONObject scalar.
type JSONObject map[string]json.RawMessage
