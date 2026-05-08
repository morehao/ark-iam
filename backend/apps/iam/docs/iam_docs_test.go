package docs

import (
	"strings"
	"testing"
)

func TestConnectorCallbackSwaggerRequiresState(t *testing.T) {
	const expected = `"name": "state",
                        "in": "query",
                        "required": true`

	if !strings.Contains(docTemplateiam, expected) {
		t.Fatalf("expected connector callback state query param to be required in swagger doc")
	}
}
