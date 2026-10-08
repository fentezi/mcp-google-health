package mcpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/fentezi/mcp-google-health/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/health/v4"
	"google.golang.org/api/option"
)

// newSession connects an MCP client to a server whose Health client talks to api.
func newSession(t *testing.T, api http.Handler) *mcp.ClientSession {
	t.Helper()

	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	healthClient, err := health.NewService(t.Context(), option.WithEndpoint(srv.URL+"/"), option.WithoutAuthentication())
	require.NoError(t, err)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err = mcpserver.New(healthClient).Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func respondJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

func decodeStructured[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// dataPointsTools lists every data-type tool with a data point its extractor must accept.
var dataPointsTools = []struct {
	tool     string
	dataType string
	dp       *health.DataPoint
}{
	{tool: "get_heart_rate", dataType: "heart-rate", dp: &health.DataPoint{HeartRate: &health.HeartRate{BeatsPerMinute: 60}}},
	{tool: "get_steps", dataType: "steps", dp: &health.DataPoint{Steps: &health.Steps{Count: 1}}},
	{tool: "get_weight", dataType: "weight", dp: &health.DataPoint{Weight: &health.Weight{WeightGrams: 1}}},
	{tool: "get_distance", dataType: "distance", dp: &health.DataPoint{Distance: &health.Distance{Millimeters: 1}}},
	{tool: "get_floors", dataType: "floors", dp: &health.DataPoint{Floors: &health.Floors{Count: 1}}},
	{tool: "get_height", dataType: "height", dp: &health.DataPoint{Height: &health.Height{HeightMillimeters: 1}}},
	{tool: "get_body_fat", dataType: "body-fat", dp: &health.DataPoint{BodyFat: &health.BodyFat{Percentage: 1}}},
	{tool: "get_oxygen_saturation", dataType: "oxygen-saturation", dp: &health.DataPoint{OxygenSaturation: &health.OxygenSaturation{Percentage: 1}}},
	{tool: "get_blood_glucose", dataType: "blood-glucose", dp: &health.DataPoint{BloodGlucose: &health.BloodGlucose{BloodGlucoseMilligramsPerDeciliter: 1}}},
	{tool: "get_heart_rate_variability", dataType: "heart-rate-variability", dp: &health.DataPoint{HeartRateVariability: &health.HeartRateVariability{RootMeanSquareOfSuccessiveDifferencesMilliseconds: 1}}},
	{tool: "get_sleep", dataType: "sleep", dp: &health.DataPoint{Sleep: &health.Sleep{Type: "STAGES"}}},
	{tool: "get_exercise", dataType: "exercise", dp: &health.DataPoint{Exercise: &health.Exercise{ExerciseType: "RUNNING"}}},
}

func TestNew_RegistersTools(t *testing.T) {
	t.Parallel()

	session := newSession(t, http.NotFoundHandler())

	res, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)

	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	want := []string{"ping"}
	for _, tt := range dataPointsTools {
		want = append(want, tt.tool)
	}
	assert.ElementsMatch(t, want, got)
}

func TestPing(t *testing.T) {
	t.Parallel()

	session := newSession(t, http.NotFoundHandler())

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "ping"})
	require.NoError(t, err)

	require.False(t, res.IsError)
	assert.Equal(t, mcpserver.PingOutput{Message: "pong"}, decodeStructured[mcpserver.PingOutput](t, res))
}

func TestDataPointsTools_DataTypeAndExtractor(t *testing.T) {
	t.Parallel()

	for _, tt := range dataPointsTools {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()

			var gotPath string
			session := newSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				respondJSON(t, w, health.ListDataPointsResponse{DataPoints: []*health.DataPoint{tt.dp}})
			}))

			res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: tt.tool})
			require.NoError(t, err)
			require.False(t, res.IsError)

			assert.Equal(t, "/v4/users/me/dataTypes/"+tt.dataType+"/dataPoints", gotPath)
			out := decodeStructured[struct {
				Samples []json.RawMessage `json:"samples"`
			}](t, res)
			assert.Len(t, out.Samples, 1, "extractor must accept a %s data point", tt.dataType)
		})
	}
}

func TestDataPointsTool_ForwardsQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments map[string]any
		wantQuery url.Values
	}{
		{
			name:      "no arguments",
			arguments: map[string]any{},
			wantQuery: url.Values{},
		},
		{
			name: "all arguments",
			arguments: map[string]any{
				"filter":    `steps.interval.start_time >= "2026-01-01T00:00:00Z"`,
				"pageSize":  50,
				"pageToken": "next",
			},
			wantQuery: url.Values{
				"filter":    {`steps.interval.start_time >= "2026-01-01T00:00:00Z"`},
				"pageSize":  {"50"},
				"pageToken": {"next"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotQuery url.Values
			session := newSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				respondJSON(t, w, health.ListDataPointsResponse{})
			}))

			res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "get_steps", Arguments: tt.arguments})
			require.NoError(t, err)
			require.False(t, res.IsError)

			for _, key := range []string{"filter", "pageSize", "pageToken"} {
				assert.Equal(t, tt.wantQuery[key], gotQuery[key], key)
			}
		})
	}
}

func TestDataPointsTool_MapsResponse(t *testing.T) {
	t.Parallel()

	session := newSession(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(t, w, health.ListDataPointsResponse{
			DataPoints: []*health.DataPoint{
				{Steps: &health.Steps{Count: 100, Interval: &health.ObservationTimeInterval{StartTime: "s1", EndTime: "e1"}}},
				{HeartRate: &health.HeartRate{BeatsPerMinute: 60}},
				{Steps: &health.Steps{Count: 200}},
			},
			NextPageToken: "page-2",
		})
	}))

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "get_steps"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	assert.Equal(t, mcpserver.DataPointsOutput[mcpserver.StepsSample]{
		Samples: []mcpserver.StepsSample{
			{Count: 100, StartTime: "s1", EndTime: "e1"},
			{Count: 200},
		},
		NextPageToken: "page-2",
	}, decodeStructured[mcpserver.DataPointsOutput[mcpserver.StepsSample]](t, res))
}

func TestDataPointsTool_APIError(t *testing.T) {
	t.Parallel()

	session := newSession(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "get_steps"})
	require.NoError(t, err)

	require.True(t, res.IsError)
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "list steps data points")
}

func TestRequireBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		authorization string
		wantCode      int
	}{
		{name: "missing header", wantCode: http.StatusUnauthorized},
		{name: "other scheme", authorization: "Basic secret", wantCode: http.StatusUnauthorized},
		{name: "lowercase scheme", authorization: "bearer secret", wantCode: http.StatusUnauthorized},
		{name: "empty token", authorization: "Bearer ", wantCode: http.StatusUnauthorized},
		{name: "wrong token", authorization: "Bearer wrong", wantCode: http.StatusUnauthorized},
		{name: "token prefix", authorization: "Bearer secre", wantCode: http.StatusUnauthorized},
		{name: "valid token", authorization: "Bearer secret", wantCode: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var reached bool
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			mcpserver.RequireBearerToken("secret")(next).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, tt.wantCode == http.StatusOK, reached)
		})
	}
}
