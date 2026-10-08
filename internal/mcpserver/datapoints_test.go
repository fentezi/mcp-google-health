package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/health/v4"
)

type extractCase[T any] struct {
	name   string
	dp     *health.DataPoint
	want   T
	wantOK bool
}

func runExtractCases[T any](t *testing.T, extract func(*health.DataPoint) (T, bool), cases []extractCase[T]) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := extract(tc.dp)

			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

const (
	sampleTime = "2026-01-02T03:04:05Z"
	startTime  = "2026-01-02T03:00:00Z"
	endTime    = "2026-01-02T04:00:00Z"
)

var (
	observationSample   = &health.ObservationSampleTime{PhysicalTime: sampleTime}
	observationInterval = &health.ObservationTimeInterval{StartTime: startTime, EndTime: endTime}
	sessionInterval     = &health.SessionTimeInterval{StartTime: startTime, EndTime: endTime}
)

func TestExtractHeartRate(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractHeartRate, []extractCase[HeartRateSample]{
		{name: "other data type", dp: &health.DataPoint{Steps: &health.Steps{Count: 1}}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{HeartRate: &health.HeartRate{BeatsPerMinute: 72}},
			want:   HeartRateSample{BeatsPerMinute: 72},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{HeartRate: &health.HeartRate{BeatsPerMinute: 72, SampleTime: observationSample}},
			want:   HeartRateSample{BeatsPerMinute: 72, SampleTime: sampleTime},
			wantOK: true,
		},
	})
}

func TestExtractSteps(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractSteps, []extractCase[StepsSample]{
		{name: "other data type", dp: &health.DataPoint{HeartRate: &health.HeartRate{BeatsPerMinute: 1}}},
		{
			name:   "without interval",
			dp:     &health.DataPoint{Steps: &health.Steps{Count: 1200}},
			want:   StepsSample{Count: 1200},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{Steps: &health.Steps{Count: 1200, Interval: observationInterval}},
			want:   StepsSample{Count: 1200, StartTime: startTime, EndTime: endTime},
			wantOK: true,
		},
	})
}

func TestExtractWeight(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractWeight, []extractCase[WeightSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{Weight: &health.Weight{WeightGrams: 70500}},
			want:   WeightSample{WeightGrams: 70500},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{Weight: &health.Weight{WeightGrams: 70500, Notes: "morning", SampleTime: observationSample}},
			want:   WeightSample{WeightGrams: 70500, Notes: "morning", SampleTime: sampleTime},
			wantOK: true,
		},
	})
}

func TestExtractDistance(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractDistance, []extractCase[DistanceSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without interval",
			dp:     &health.DataPoint{Distance: &health.Distance{Millimeters: 5000}},
			want:   DistanceSample{Millimeters: 5000},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{Distance: &health.Distance{Millimeters: 5000, Interval: observationInterval}},
			want:   DistanceSample{Millimeters: 5000, StartTime: startTime, EndTime: endTime},
			wantOK: true,
		},
	})
}

func TestExtractFloors(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractFloors, []extractCase[FloorsSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without interval",
			dp:     &health.DataPoint{Floors: &health.Floors{Count: 3}},
			want:   FloorsSample{Count: 3},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{Floors: &health.Floors{Count: 3, Interval: observationInterval}},
			want:   FloorsSample{Count: 3, StartTime: startTime, EndTime: endTime},
			wantOK: true,
		},
	})
}

func TestExtractHeight(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractHeight, []extractCase[HeightSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{Height: &health.Height{HeightMillimeters: 1800}},
			want:   HeightSample{HeightMillimeters: 1800},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{Height: &health.Height{HeightMillimeters: 1800, SampleTime: observationSample}},
			want:   HeightSample{HeightMillimeters: 1800, SampleTime: sampleTime},
			wantOK: true,
		},
	})
}

func TestExtractBodyFat(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractBodyFat, []extractCase[BodyFatSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{BodyFat: &health.BodyFat{Percentage: 18.5}},
			want:   BodyFatSample{Percentage: 18.5},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{BodyFat: &health.BodyFat{Percentage: 18.5, SampleTime: observationSample}},
			want:   BodyFatSample{Percentage: 18.5, SampleTime: sampleTime},
			wantOK: true,
		},
	})
}

func TestExtractOxygenSaturation(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractOxygenSaturation, []extractCase[OxygenSaturationSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{OxygenSaturation: &health.OxygenSaturation{Percentage: 97}},
			want:   OxygenSaturationSample{Percentage: 97},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{OxygenSaturation: &health.OxygenSaturation{Percentage: 97, SampleTime: observationSample}},
			want:   OxygenSaturationSample{Percentage: 97, SampleTime: sampleTime},
			wantOK: true,
		},
	})
}

func TestExtractBloodGlucose(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractBloodGlucose, []extractCase[BloodGlucoseSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{BloodGlucose: &health.BloodGlucose{BloodGlucoseMilligramsPerDeciliter: 95}},
			want:   BloodGlucoseSample{MilligramsPerDeciliter: 95},
			wantOK: true,
		},
		{
			name: "full",
			dp: &health.DataPoint{BloodGlucose: &health.BloodGlucose{
				BloodGlucoseMilligramsPerDeciliter: 95,
				MealType:                           "BREAKFAST",
				MeasurementTiming:                  "BEFORE_MEAL",
				Notes:                              "fasting",
				SampleTime:                         observationSample,
			}},
			want: BloodGlucoseSample{
				MilligramsPerDeciliter: 95,
				MealType:               "BREAKFAST",
				MeasurementTiming:      "BEFORE_MEAL",
				Notes:                  "fasting",
				SampleTime:             sampleTime,
			},
			wantOK: true,
		},
	})
}

func TestExtractHeartRateVariability(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractHeartRateVariability, []extractCase[HeartRateVariabilitySample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without sample time",
			dp:     &health.DataPoint{HeartRateVariability: &health.HeartRateVariability{RootMeanSquareOfSuccessiveDifferencesMilliseconds: 42}},
			want:   HeartRateVariabilitySample{RmssdMilliseconds: 42},
			wantOK: true,
		},
		{
			name: "full",
			dp: &health.DataPoint{HeartRateVariability: &health.HeartRateVariability{
				RootMeanSquareOfSuccessiveDifferencesMilliseconds: 42,
				StandardDeviationMilliseconds:                     55,
				SampleTime:                                        observationSample,
			}},
			want:   HeartRateVariabilitySample{RmssdMilliseconds: 42, StandardDeviationMilliseconds: 55, SampleTime: sampleTime},
			wantOK: true,
		},
	})
}

func TestExtractSleep(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractSleep, []extractCase[SleepSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without interval",
			dp:     &health.DataPoint{Sleep: &health.Sleep{Type: "STAGES"}},
			want:   SleepSample{Type: "STAGES"},
			wantOK: true,
		},
		{
			name:   "full",
			dp:     &health.DataPoint{Sleep: &health.Sleep{Type: "STAGES", Interval: sessionInterval}},
			want:   SleepSample{Type: "STAGES", StartTime: startTime, EndTime: endTime},
			wantOK: true,
		},
	})
}

func TestExtractExercise(t *testing.T) {
	t.Parallel()

	runExtractCases(t, extractExercise, []extractCase[ExerciseSample]{
		{name: "other data type", dp: &health.DataPoint{}},
		{
			name:   "without interval",
			dp:     &health.DataPoint{Exercise: &health.Exercise{ExerciseType: "RUNNING"}},
			want:   ExerciseSample{ExerciseType: "RUNNING"},
			wantOK: true,
		},
		{
			name: "full",
			dp: &health.DataPoint{Exercise: &health.Exercise{
				ExerciseType:   "RUNNING",
				DisplayName:    "Morning run",
				ActiveDuration: "1800s",
				Notes:          "easy pace",
				Interval:       sessionInterval,
			}},
			want: ExerciseSample{
				ExerciseType:   "RUNNING",
				DisplayName:    "Morning run",
				StartTime:      startTime,
				EndTime:        endTime,
				ActiveDuration: "1800s",
				Notes:          "easy pace",
			},
			wantOK: true,
		},
	})
}
