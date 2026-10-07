package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/health/v4"
)

// DataPointsInput is the shared parameter shape for every data-type List tool.
type DataPointsInput struct {
	Filter    string `json:"filter,omitempty" jsonschema:"optional AIP-160 filter expression restricting the time range (field name pattern depends on the data type, e.g. steps.interval.start_time or weight.sample_time.physical_time)"`
	PageSize  int64  `json:"pageSize,omitempty" jsonschema:"maximum number of data points to return (Google default 1440, max 10000)"`
	PageToken string `json:"pageToken,omitempty" jsonschema:"nextPageToken from a previous call, to fetch the next page"`
}

// DataPointsOutput is the shared response shape for every data-type List tool.
type DataPointsOutput[T any] struct {
	Samples       []T    `json:"samples"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// registerDataPointsTool registers an MCP tool that lists data points of the
// given Google Health data type and extracts them into T via extract.
func registerDataPointsTool[T any](
	server *mcp.Server,
	healthClient *health.Service,
	name, description, dataType string,
	extract func(*health.DataPoint) (T, bool),
) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Description: description,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in DataPointsInput) (*mcp.CallToolResult, DataPointsOutput[T], error) {
		call := healthClient.Users.DataTypes.DataPoints.List("users/me/dataTypes/" + dataType).Context(ctx)
		if in.Filter != "" {
			call = call.Filter(in.Filter)
		}
		if in.PageSize > 0 {
			call = call.PageSize(in.PageSize)
		}
		if in.PageToken != "" {
			call = call.PageToken(in.PageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, DataPointsOutput[T]{}, fmt.Errorf("list %s data points: %w", dataType, err)
		}

		out := DataPointsOutput[T]{NextPageToken: resp.NextPageToken}
		for _, dp := range resp.DataPoints {
			if v, ok := extract(dp); ok {
				out.Samples = append(out.Samples, v)
			}
		}

		return nil, out, nil
	})
}

type HeartRateSample struct {
	BeatsPerMinute int64  `json:"beatsPerMinute"`
	SampleTime     string `json:"sampleTime,omitempty"`
}

func extractHeartRate(dp *health.DataPoint) (HeartRateSample, bool) {
	if dp.HeartRate == nil {
		return HeartRateSample{}, false
	}
	s := HeartRateSample{BeatsPerMinute: dp.HeartRate.BeatsPerMinute}
	if dp.HeartRate.SampleTime != nil {
		s.SampleTime = dp.HeartRate.SampleTime.PhysicalTime
	}
	return s, true
}

type StepsSample struct {
	Count     int64  `json:"count"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

func extractSteps(dp *health.DataPoint) (StepsSample, bool) {
	if dp.Steps == nil {
		return StepsSample{}, false
	}
	s := StepsSample{Count: dp.Steps.Count}
	if dp.Steps.Interval != nil {
		s.StartTime = dp.Steps.Interval.StartTime
		s.EndTime = dp.Steps.Interval.EndTime
	}
	return s, true
}

type WeightSample struct {
	WeightGrams float64 `json:"weightGrams"`
	Notes       string  `json:"notes,omitempty"`
	SampleTime  string  `json:"sampleTime,omitempty"`
}

func extractWeight(dp *health.DataPoint) (WeightSample, bool) {
	if dp.Weight == nil {
		return WeightSample{}, false
	}
	s := WeightSample{WeightGrams: dp.Weight.WeightGrams, Notes: dp.Weight.Notes}
	if dp.Weight.SampleTime != nil {
		s.SampleTime = dp.Weight.SampleTime.PhysicalTime
	}
	return s, true
}

type DistanceSample struct {
	Millimeters int64  `json:"millimeters"`
	StartTime   string `json:"startTime,omitempty"`
	EndTime     string `json:"endTime,omitempty"`
}

func extractDistance(dp *health.DataPoint) (DistanceSample, bool) {
	if dp.Distance == nil {
		return DistanceSample{}, false
	}
	s := DistanceSample{Millimeters: dp.Distance.Millimeters}
	if dp.Distance.Interval != nil {
		s.StartTime = dp.Distance.Interval.StartTime
		s.EndTime = dp.Distance.Interval.EndTime
	}
	return s, true
}

type FloorsSample struct {
	Count     int64  `json:"count"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

func extractFloors(dp *health.DataPoint) (FloorsSample, bool) {
	if dp.Floors == nil {
		return FloorsSample{}, false
	}
	s := FloorsSample{Count: dp.Floors.Count}
	if dp.Floors.Interval != nil {
		s.StartTime = dp.Floors.Interval.StartTime
		s.EndTime = dp.Floors.Interval.EndTime
	}
	return s, true
}

type HeightSample struct {
	HeightMillimeters int64  `json:"heightMillimeters"`
	SampleTime        string `json:"sampleTime,omitempty"`
}

func extractHeight(dp *health.DataPoint) (HeightSample, bool) {
	if dp.Height == nil {
		return HeightSample{}, false
	}
	s := HeightSample{HeightMillimeters: dp.Height.HeightMillimeters}
	if dp.Height.SampleTime != nil {
		s.SampleTime = dp.Height.SampleTime.PhysicalTime
	}
	return s, true
}

type BodyFatSample struct {
	Percentage float64 `json:"percentage"`
	SampleTime string  `json:"sampleTime,omitempty"`
}

func extractBodyFat(dp *health.DataPoint) (BodyFatSample, bool) {
	if dp.BodyFat == nil {
		return BodyFatSample{}, false
	}
	s := BodyFatSample{Percentage: dp.BodyFat.Percentage}
	if dp.BodyFat.SampleTime != nil {
		s.SampleTime = dp.BodyFat.SampleTime.PhysicalTime
	}
	return s, true
}

type OxygenSaturationSample struct {
	Percentage float64 `json:"percentage"`
	SampleTime string  `json:"sampleTime,omitempty"`
}

func extractOxygenSaturation(dp *health.DataPoint) (OxygenSaturationSample, bool) {
	if dp.OxygenSaturation == nil {
		return OxygenSaturationSample{}, false
	}
	s := OxygenSaturationSample{Percentage: dp.OxygenSaturation.Percentage}
	if dp.OxygenSaturation.SampleTime != nil {
		s.SampleTime = dp.OxygenSaturation.SampleTime.PhysicalTime
	}
	return s, true
}

type BloodGlucoseSample struct {
	MilligramsPerDeciliter float64 `json:"milligramsPerDeciliter"`
	MealType               string  `json:"mealType,omitempty"`
	MeasurementTiming      string  `json:"measurementTiming,omitempty"`
	Notes                  string  `json:"notes,omitempty"`
	SampleTime             string  `json:"sampleTime,omitempty"`
}

func extractBloodGlucose(dp *health.DataPoint) (BloodGlucoseSample, bool) {
	if dp.BloodGlucose == nil {
		return BloodGlucoseSample{}, false
	}
	s := BloodGlucoseSample{
		MilligramsPerDeciliter: dp.BloodGlucose.BloodGlucoseMilligramsPerDeciliter,
		MealType:               dp.BloodGlucose.MealType,
		MeasurementTiming:      dp.BloodGlucose.MeasurementTiming,
		Notes:                  dp.BloodGlucose.Notes,
	}
	if dp.BloodGlucose.SampleTime != nil {
		s.SampleTime = dp.BloodGlucose.SampleTime.PhysicalTime
	}
	return s, true
}

type HeartRateVariabilitySample struct {
	RmssdMilliseconds             float64 `json:"rmssdMilliseconds"`
	StandardDeviationMilliseconds float64 `json:"standardDeviationMilliseconds"`
	SampleTime                    string  `json:"sampleTime,omitempty"`
}

func extractHeartRateVariability(dp *health.DataPoint) (HeartRateVariabilitySample, bool) {
	if dp.HeartRateVariability == nil {
		return HeartRateVariabilitySample{}, false
	}
	s := HeartRateVariabilitySample{
		RmssdMilliseconds:             dp.HeartRateVariability.RootMeanSquareOfSuccessiveDifferencesMilliseconds,
		StandardDeviationMilliseconds: dp.HeartRateVariability.StandardDeviationMilliseconds,
	}
	if dp.HeartRateVariability.SampleTime != nil {
		s.SampleTime = dp.HeartRateVariability.SampleTime.PhysicalTime
	}
	return s, true
}

type SleepSample struct {
	Type      string `json:"type,omitempty"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

func extractSleep(dp *health.DataPoint) (SleepSample, bool) {
	if dp.Sleep == nil {
		return SleepSample{}, false
	}
	s := SleepSample{Type: dp.Sleep.Type}
	if dp.Sleep.Interval != nil {
		s.StartTime = dp.Sleep.Interval.StartTime
		s.EndTime = dp.Sleep.Interval.EndTime
	}
	return s, true
}

type ExerciseSample struct {
	ExerciseType   string `json:"exerciseType,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	StartTime      string `json:"startTime,omitempty"`
	EndTime        string `json:"endTime,omitempty"`
	ActiveDuration string `json:"activeDuration,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

func extractExercise(dp *health.DataPoint) (ExerciseSample, bool) {
	if dp.Exercise == nil {
		return ExerciseSample{}, false
	}
	s := ExerciseSample{
		ExerciseType:   dp.Exercise.ExerciseType,
		DisplayName:    dp.Exercise.DisplayName,
		ActiveDuration: dp.Exercise.ActiveDuration,
		Notes:          dp.Exercise.Notes,
	}
	if dp.Exercise.Interval != nil {
		s.StartTime = dp.Exercise.Interval.StartTime
		s.EndTime = dp.Exercise.Interval.EndTime
	}
	return s, true
}

// registerDataTypeTools registers every supported data-type read tool.
func registerDataTypeTools(server *mcp.Server, healthClient *health.Service) {
	registerDataPointsTool(server, healthClient, "get_heart_rate",
		"Fetch the user's recorded heart-rate samples from Google Health.", "heart-rate", extractHeartRate)
	registerDataPointsTool(server, healthClient, "get_steps",
		"Fetch the user's step-count intervals from Google Health.", "steps", extractSteps)
	registerDataPointsTool(server, healthClient, "get_weight",
		"Fetch the user's recorded weight samples from Google Health.", "weight", extractWeight)
	registerDataPointsTool(server, healthClient, "get_distance",
		"Fetch the user's distance-traveled intervals from Google Health.", "distance", extractDistance)
	registerDataPointsTool(server, healthClient, "get_floors",
		"Fetch the user's floors-climbed intervals from Google Health.", "floors", extractFloors)
	registerDataPointsTool(server, healthClient, "get_height",
		"Fetch the user's recorded height samples from Google Health.", "height", extractHeight)
	registerDataPointsTool(server, healthClient, "get_body_fat",
		"Fetch the user's recorded body-fat percentage samples from Google Health.", "body-fat", extractBodyFat)
	registerDataPointsTool(server, healthClient, "get_oxygen_saturation",
		"Fetch the user's recorded blood-oxygen saturation samples from Google Health.", "oxygen-saturation", extractOxygenSaturation)
	registerDataPointsTool(server, healthClient, "get_blood_glucose",
		"Fetch the user's recorded blood-glucose samples from Google Health.", "blood-glucose", extractBloodGlucose)
	registerDataPointsTool(server, healthClient, "get_heart_rate_variability",
		"Fetch the user's recorded heart-rate variability samples from Google Health.", "heart-rate-variability", extractHeartRateVariability)
	registerDataPointsTool(server, healthClient, "get_sleep",
		"Fetch the user's sleep sessions from Google Health.", "sleep", extractSleep)
	registerDataPointsTool(server, healthClient, "get_exercise",
		"Fetch the user's exercise sessions from Google Health.", "exercise", extractExercise)
}
