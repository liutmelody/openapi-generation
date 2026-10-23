package changes

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

func TestChanges_MethodChanges_ServerSentEvents(t *testing.T) {
	oldSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /reports:
    post:
      operationId: generateReport
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ReportRequest'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Report'
            text/event-stream:
              schema:
                $ref: '#/components/schemas/ReportEvent'
components:
  schemas:
    ReportRequest:
      type: object
      required: [format]
      properties:
        format:
          type: string
          enum: [pdf, csv]
    Report:
      type: object
      properties:
        id:
          type: string
    ReportEvent:
      type: object
      properties:
        data:
          type: string`

	newSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /reports:
    post:
      operationId: generateReport
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ReportRequest'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Report'
            text/event-stream:
              schema:
                $ref: '#/components/schemas/ReportEvent'
  /widgets:
    post:
      operationId: createWidget
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Report'
            text/event-stream:
              schema:
                $ref: '#/components/schemas/ReportEvent'
  /events:
    get:
      operationId: tailEvents
      responses:
        '200':
          description: Success
          content:
            text/event-stream:
              schema:
                $ref: '#/components/schemas/ReportEvent'
components:
  schemas:
    ReportRequest:
      type: object
      required: [format]
      properties:
        format:
          type: string
          enum: [pdf, csv, html]
    Report:
      type: object
      properties:
        id:
          type: string
    ReportEvent:
      type: object
      properties:
        data:
          type: string`

	lang := "python"
	oldConfig, newConfig := createTestConfigs(t, testSpecPair{
		oldSpec: oldSpec,
		newSpec: newSpec,
		lang:    lang,
	})

	diff, err := Changes(t.Context(), oldConfig, newConfig)
	if err != nil {
		t.Fatalf("failed to get changes. error: %s", err.Error())
	}

	snapshot := formatTestSnapshotWithDiff("server_sent_events", oldSpec, newSpec, diff)
	snaps.MatchSnapshot(t, snapshot)
}
