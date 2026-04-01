// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelserializer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter/internal/serializer/otelserializer/serializeprofiles"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pprofiletest"
)

func basicProfiles() pprofiletest.Profiles {
	r := pcommon.NewResource()
	r.Attributes().PutStr("key1", "value1")
	return pprofiletest.Profiles{
		ResourceProfiles: []pprofiletest.ResourceProfile{
			{
				Resource: r,
				ScopeProfiles: []pprofiletest.ScopeProfile{
					{
						Scope: pcommon.NewInstrumentationScope(),
						Profiles: []pprofiletest.Profile{
							{
								SampleType: pprofiletest.ValueType{Typ: "samples", Unit: "count"},
								PeriodType: pprofiletest.ValueType{Typ: "cpu", Unit: "nanoseconds"},
								Attributes: []pprofiletest.Attribute{
									{Key: "process.executable.build_id.htlhash", Value: "600DCAFE4A110000F2BF38C493F5FB92"},
									{Key: "profile.frame.type", Value: "native"},
									{Key: "host.id", Value: "localhost"},
								},
								Sample: []pprofiletest.Sample{
									{
										TimestampsUnixNano: []uint64{0},
										Values:             []int64{1},
										Locations: []pprofiletest.Location{
											{
												Mapping: &pprofiletest.Mapping{},
												Address: 111,
											},
										},
									},
								},
								ProfileID: pprofile.NewProfileIDEmpty(),
							},
						},
					},
				},
			},
		},
	}
}

func TestSerializeProfile(t *testing.T) {
	tests := []struct {
		name              string
		buildDictionary   func() pprofile.ProfilesDictionary
		profileCustomizer func(resource pcommon.Resource, scope pcommon.InstrumentationScope, record pprofile.Profile)
		wantErr           bool
		expected          []map[string]any
	}{
		{
			name: "with a simple sample",
			buildDictionary: func() pprofile.ProfilesDictionary {
				dic := pprofile.NewProfilesDictionary()
				dic.StringTable().Append("samples", "count", "cpu", "nanoseconds")

				a := dic.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(4)
				dic.StringTable().Append("process.executable.build_id.htlhash")
				a.Value().SetStr("600DCAFE4A110000F2BF38C493F5FB92")
				a = dic.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(5)
				dic.StringTable().Append("profile.frame.type")
				a.Value().SetStr("native")
				a = dic.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(6)
				dic.StringTable().Append("host.id")
				a.Value().SetStr("localhost")

				dic.MappingTable().AppendEmpty()
				m := dic.MappingTable().AppendEmpty()
				m.AttributeIndices().Append(0)

				l := dic.LocationTable().AppendEmpty()
				l.SetMappingIndex(1)
				l.SetAddress(111)
				l.AttributeIndices().Append(1)

				stack := dic.StackTable().AppendEmpty()
				stack.LocationIndices().Append(0)

				return dic
			},
			profileCustomizer: func(r pcommon.Resource, _ pcommon.InstrumentationScope, profile pprofile.Profile) {
				st := profile.SampleType()
				st.SetTypeStrindex(0)
				st.SetUnitStrindex(1)
				pt := profile.PeriodType()
				pt.SetTypeStrindex(2)
				pt.SetUnitStrindex(3)
				profile.SetPeriod(1e9 / 20)

				profile.AttributeIndices().Append(2)

				sample := profile.Samples().AppendEmpty()
				sample.TimestampsUnixNano().Append(0)
				sample.AttributeIndices().Append(2)
				sample.SetStackIndex(0)

				r.Attributes().PutStr("process.executable.name", "libc.so.6")
			},
			wantErr: false,
			expected: []map[string]any{
				{
					"Stacktrace.frame.ids":   "YA3K_koRAADyvzjEk_X7kgAAAAAAAABv",
					"Stacktrace.frame.types": "AQM",
					"ecs.version":            "1.12.0",
				},
				{
					"@timestamp":                    "1970-01-01T00:00:00Z",
					"Stacktrace.count":              json.Number("1"),
					"Stacktrace.sampling_frequency": json.Number("20"),
					"Stacktrace.id":                 "02VzuClbpt_P3xxwox83Ng",
					"ecs.version":                   "1.12.0",
					"host.id":                       "localhost",
					"process.executable.name":       "libc.so.6",
					"process.thread.name":           "",
					"profiling.project.id":          json.Number("2"),
				},
				{
					"script": map[string]any{
						"params": map[string]any{
							"buildid":     "YA3K_koRAADyvzjEk_X7kg",
							"ecs.version": "1.12.0",
							"filename":    "samples",
							"timestamp":   json.Number(fmt.Sprintf("%d", serializeprofiles.GetStartOfWeekFromTime(time.Now()))),
						},
						"source": serializeprofiles.ExeMetadataUpsertScript,
					},
					"scripted_upsert": true,
					"upsert":          map[string]any{},
				},
				{
					"Stacktrace.frame.id":     []any{"YA3K_koRAADyvzjEk_X7kgAAAAAAAABv"},
					"Symbolization.retries":   json.Number("0"),
					"Symbolization.time.next": "",
					"Time.created":            "",
					"ecs.version":             serializeprofiles.EcsVersionString,
				},
				{
					"Executable.file.id":      []any{"YA3K_koRAADyvzjEk_X7kg"},
					"Symbolization.retries":   json.Number("0"),
					"Symbolization.time.next": "",
					"Time.created":            "",
					"ecs.version":             serializeprofiles.EcsVersionString,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dic := tt.buildDictionary()
			profiles := pprofile.NewProfiles()
			resource := profiles.ResourceProfiles().AppendEmpty()
			scope := resource.ScopeProfiles().AppendEmpty()
			profile := scope.Profiles().AppendEmpty()
			tt.profileCustomizer(resource.Resource(), scope.Scope(), profile)
			profiles.MarkReadOnly()

			buf := []*bytes.Buffer{}
			ser, err := New()
			require.NoError(t, err)
			err = ser.SerializeProfile(dic, resource.Resource(), scope.Scope(), profile, false, func(b *bytes.Buffer, _, _ string) error {
				buf = append(buf, b)
				return nil
			})
			if !tt.wantErr {
				require.NoError(t, err)
			}

			var results []map[string]any
			for _, v := range buf {
				var d map[string]any
				decoder := json.NewDecoder(v)
				decoder.UseNumber()
				require.NoError(t, decoder.Decode(&d))

				// Remove timestamps to allow comparing test results with expected values.
				for k, v := range d {
					switch k {
					case "Symbolization.time.next", "Time.created":
						tm, err := time.Parse(time.RFC3339Nano, v.(string))
						require.NoError(t, err)
						assert.True(t, isWithinLastSecond(tm))
						d[k] = ""
					}
				}
				results = append(results, d)
			}

			assert.Equal(t, tt.expected, results)
		})
	}
}

func isWithinLastSecond(t time.Time) bool {
	return time.Since(t) < time.Second
}

func TestSerializeProfile_SampleCountDataStream(t *testing.T) {
	const sampleCountIndex = "profiles-cpu_nanoseconds_samples_count_50000000-otel"

	dic := pprofile.NewProfilesDictionary()
	dic.StringTable().Append("samples", "count", "cpu", "nanoseconds")

	a := dic.AttributeTable().AppendEmpty()
	a.SetKeyStrindex(4)
	dic.StringTable().Append("profile.frame.type")
	a.Value().SetStr("native")

	a = dic.AttributeTable().AppendEmpty()
	a.SetKeyStrindex(5)
	dic.StringTable().Append("thread.name")
	a.Value().SetStr("worker-7")

	dic.MappingTable().AppendEmpty()
	dic.MappingTable().AppendEmpty()

	dic.StringTable().Append("myfunc", "myfile.go")
	fn := dic.FunctionTable().AppendEmpty()
	fn.SetNameStrindex(6)
	fn.SetFilenameStrindex(7)

	l := dic.LocationTable().AppendEmpty()
	l.SetMappingIndex(1)
	l.SetAddress(111)
	l.AttributeIndices().Append(0)
	line := l.Lines().AppendEmpty()
	line.SetFunctionIndex(0)
	line.SetLine(42)

	stack := dic.StackTable().AppendEmpty()
	stack.LocationIndices().Append(0)

	profiles := pprofile.NewProfiles()
	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("service.name", "my-service")
	rp.Resource().Attributes().PutStr("host.id", "host-123")
	sp := rp.ScopeProfiles().AppendEmpty()
	profile := sp.Profiles().AppendEmpty()

	st := profile.SampleType()
	st.SetTypeStrindex(0)
	st.SetUnitStrindex(1)
	pt := profile.PeriodType()
	pt.SetTypeStrindex(2)
	pt.SetUnitStrindex(3)
	profile.SetPeriod(1e9 / 20)

	sample := profile.Samples().AppendEmpty()
	sample.TimestampsUnixNano().Append(0)
	sample.Values().Append(3)
	sample.SetStackIndex(0)
	sample.AttributeIndices().Append(1)

	profiles.MarkReadOnly()

	ser, err := New()
	require.NoError(t, err)

	type pushedDoc struct {
		docID string
		body  map[string]any
	}
	var got []pushedDoc
	err = ser.SerializeProfile(dic, rp.Resource(), sp.Scope(), profile, true, func(b *bytes.Buffer, docID, index string) error {
		if index != sampleCountIndex {
			return nil
		}
		var body map[string]any
		decoder := json.NewDecoder(b)
		decoder.UseNumber()
		require.NoError(t, decoder.Decode(&body))
		got = append(got, pushedDoc{docID: docID, body: body})
		return nil
	})
	require.NoError(t, err)
	require.Len(t, got, 1)

	assert.Empty(t, got[0].docID)
	assert.Equal(t, "1970-01-01T00:00:00Z", got[0].body["@timestamp"])
	assert.Equal(t, json.Number("3"), got[0].body["sample.value"])
	assert.Equal(t, json.Number("50000000"), got[0].body["period"])
	assert.Equal(t, "my-service", got[0].body["service.name"])
	assert.Equal(t, "host-123", got[0].body["host.id"])
	assert.Equal(t, "worker-7", got[0].body["thread.name"])

	stackAny, ok := got[0].body["stack"].([]any)
	require.True(t, ok)
	require.Len(t, stackAny, 1)
	frame, ok := stackAny[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "native", frame["frame.type"])
	assert.Equal(t, []any{"myfunc"}, frame["function.name"])
	assert.Equal(t, []any{"myfile.go"}, frame["file.name"])
	assert.Equal(t, []any{json.Number("42")}, frame["line.number"])
}

func TestSerializeProfile_SampleCountDataStreamDisabled(t *testing.T) {
	dic := pprofile.NewProfilesDictionary()
	dic.StringTable().Append("samples", "count", "cpu", "nanoseconds")
	dic.StackTable().AppendEmpty()

	profiles := pprofile.NewProfiles()
	rp := profiles.ResourceProfiles().AppendEmpty()
	sp := rp.ScopeProfiles().AppendEmpty()
	profile := sp.Profiles().AppendEmpty()

	st := profile.SampleType()
	st.SetTypeStrindex(0)
	st.SetUnitStrindex(1)
	pt := profile.PeriodType()
	pt.SetTypeStrindex(2)
	pt.SetUnitStrindex(3)

	sample := profile.Samples().AppendEmpty()
	sample.TimestampsUnixNano().Append(0)
	sample.Values().Append(3)
	sample.SetStackIndex(0)

	profiles.MarkReadOnly()

	ser, err := New()
	require.NoError(t, err)

	var indices []string
	err = ser.SerializeProfile(dic, rp.Resource(), sp.Scope(), profile, false, func(_ *bytes.Buffer, _, index string) error {
		indices = append(indices, index)
		return nil
	})
	require.NoError(t, err)
	assert.NotContains(t, indices, "profiles-cpu_nanoseconds_samples_count_50000000-otel")
}

func BenchmarkSerializeProfile(b *testing.B) {
	ser, err := New()
	require.NoError(b, err)

	profiles := basicProfiles().Transform()
	resource := profiles.ResourceProfiles().At(0)
	scope := resource.ScopeProfiles().At(0)
	profile := scope.Profiles().At(0)
	pushData := func(_ *bytes.Buffer, _, _ string) error {
		return nil
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = ser.SerializeProfile(profiles.Dictionary(), resource.Resource(), scope.Scope(), profile, true, pushData)
	}
}
