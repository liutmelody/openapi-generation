package changes

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestChanges_FieldChanges_FieldDeprecated(t *testing.T) {
	oldSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /widgets:
    post:
      operationId: updateWidget
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                color:
                  type: string
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  nickname:
                    type: string
                  label:
                    type: string`

	newSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /widgets:
    post:
      operationId: updateWidget
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                color:
                  type: string
                  deprecated: true
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  nickname:
                    type: string
                    deprecated: true
                  label:
                    type: string
                    nullable: true
                    deprecated: true`

	lang := "typescript"
	oldConfig, newConfig := createTestConfigs(t, testSpecPair{
		oldSpec: oldSpec,
		newSpec: newSpec,
		lang:    lang,
	})

	diff, err := Changes(t.Context(), oldConfig, newConfig)
	if err != nil {
		t.Fatalf("failed to get changes. error: %s", err.Error())
	}

	snapshot := formatTestSnapshotWithDiff("field_deprecated", oldSpec, newSpec, diff)
	snaps.MatchSnapshot(t, snapshot)
}
