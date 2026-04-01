// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelserializer // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter/internal/serializer/otelserializer"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter/internal/elasticsearch"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter/internal/lru"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter/internal/serializer/otelserializer/serializeprofiles"
)

const (
	AllEventsIndex   = "profiling-events-all"
	StackTraceIndex  = "profiling-stacktraces"
	StackFrameIndex  = "profiling-stackframes"
	ExecutablesIndex = "profiling-executables"

	ExecutablesSymQueueIndex = "profiling-sq-executables"
	LeafFramesSymQueueIndex  = "profiling-sq-leafframes"

	HostsMetadataIndex = "profiling-hosts"

	// sampleCountDataStreamType is the fixed data_stream.type used for the
	// normalized, OTel-native sample count data stream.
	sampleCountDataStreamType = "profiles"
	// sampleCountDataStreamNamespace is the fixed data_stream.namespace used
	// for the normalized, OTel-native sample count data stream.
	sampleCountDataStreamNamespace = "otel"

	maxSampleCountDatasetBytes        = 100
	disallowedSampleCountDatasetRunes = "-\\/*?\"<>| ,#:"
)

// SerializeProfile serializes a profile and calls the `pushData` callback for each generated document.
//
// If includeSampleCountDataStream is true, a self-contained, normalized copy
// of each stacktrace sample is additionally pushed to a
// "profiles-<dataset>-otel" data stream, where <dataset> is derived from the
// profile's period type and sample type (e.g. "cpu_nanoseconds_samples_count").
func (s *Serializer) SerializeProfile(dic pprofile.ProfilesDictionary, resource pcommon.Resource, scope pcommon.InstrumentationScope, profile pprofile.Profile, includeSampleCountDataStream bool, pushData func(*bytes.Buffer, string, string) error) error {
	err := s.createLRUs()
	if err != nil {
		return err
	}

	pushDataAsJSON := func(data any, id, index string) (err error) {
		c, err := toJSON(data)
		if err != nil {
			return err
		}
		return pushData(c, id, index)
	}

	data, err := serializeprofiles.Transform(dic, resource, scope, profile)
	if err != nil {
		return err
	}

	err = s.knownTraces.WithLock(func(tracesSet lru.LockedLRUSet) error {
		for i := range data {
			payload := &data[i]
			event := payload.StackTraceEvent

			if event.StackTraceID != "" {
				err = pushDataAsJSON(event, "", AllEventsIndex)
				if err != nil {
					return err
				}
				err = serializeprofiles.IndexDownsampledEvent(event, pushDataAsJSON)
				if err != nil {
					return err
				}
			}

			if payload.StackTrace.DocID != "" {
				if !tracesSet.CheckAndAdd(payload.StackTrace.DocID) {
					err = pushDataAsJSON(payload.StackTrace, payload.StackTrace.DocID, StackTraceIndex)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = s.knownFrames.WithLock(func(framesSet lru.LockedLRUSet) error {
		for i := range data {
			payload := &data[i]
			for j := range payload.StackFrames {
				stackFrame := &payload.StackFrames[j]
				if !framesSet.CheckAndAdd(stackFrame.DocID) {
					err = pushDataAsJSON(stackFrame, stackFrame.DocID, StackFrameIndex)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = s.knownExecutables.WithLock(func(executablesSet lru.LockedLRUSet) error {
		for i := range data {
			payload := &data[i]
			for _, executable := range payload.Executables {
				if !executablesSet.CheckAndAdd(executable.DocID) {
					err = pushDataAsJSON(executable, executable.DocID, ExecutablesIndex)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = s.knownUnsymbolizedFrames.WithLock(func(unsymbolizedFramesSet lru.LockedLRUSet) error {
		for i := range data {
			payload := &data[i]
			for _, frame := range payload.UnsymbolizedLeafFrames {
				if !unsymbolizedFramesSet.CheckAndAdd(frame.DocID) {
					err = pushDataAsJSON(frame, frame.DocID, LeafFramesSymQueueIndex)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = s.knownHosts.WithLock(func(hostMetadata lru.LockedLRUSet) error {
		for i := range data {
			payload := &data[i]
			hostID := payload.ResourceAttrs.HostID
			if hostID == "" {
				continue
			}

			if !hostMetadata.CheckAndAdd(hostID) {
				err = pushDataAsJSON(payload.ResourceAttrs, "", HostsMetadataIndex)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = s.knownUnsymbolizedExecutables.WithLock(func(unsymbolizedExecutablesSet lru.LockedLRUSet) error {
		for i := range data {
			payload := &data[i]
			for _, executable := range payload.UnsymbolizedExecutables {
				if !unsymbolizedExecutablesSet.CheckAndAdd(executable.DocID) {
					err = pushDataAsJSON(executable, executable.DocID, ExecutablesSymQueueIndex)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !includeSampleCountDataStream {
		return nil
	}

	events, err := serializeprofiles.SampleDSEvents(dic, resource, scope, profile)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	index := sampleCountDataStreamIndex(dic, profile)
	for i := range events {
		if err := pushDataAsJSON(&events[i], "", index); err != nil {
			return err
		}
	}

	return nil
}

// sampleCountDataStreamIndex builds the "profiles-<dataset>-otel" data
// stream name for the normalized sample count events. <dataset> is derived
// from the profile's period type and sample type. For sampling-based profiles
// (sample type "samples/count") the period value is appended because different
// sampling frequencies must land in different data streams, e.g.
// "cpu_nanoseconds_samples_count_50000000" for a 20 Hz on-CPU profile.
// For event-based profiles the period is irrelevant and omitted.
func sampleCountDataStreamIndex(dic pprofile.ProfilesDictionary, profile pprofile.Profile) string {
	sType, sUnit, pType, pUnit := serializeprofiles.SampleAndPeriodType(dic, profile)
	raw := pType + "_" + pUnit + "_" + sType + "_" + sUnit
	if sType == "samples" && sUnit == "count" {
		raw += fmt.Sprintf("_%d", profile.Period())
	}
	dataset := sanitizeSampleCountDataset(raw)
	return elasticsearch.NewDataStreamIndex(sampleCountDataStreamType, dataset, sampleCountDataStreamNamespace).Index
}

// sanitizeSampleCountDataset sanitizes dataset to apply the same restrictions
// as data_stream.dataset elsewhere in the exporter, see
// https://www.elastic.co/guide/en/ecs/current/ecs-data_stream.html
func sanitizeSampleCountDataset(dataset string) string {
	dataset = strings.Map(func(r rune) rune {
		if strings.ContainsRune(disallowedSampleCountDatasetRunes, r) {
			return '_'
		}
		return unicode.ToLower(r)
	}, dataset)
	if len(dataset) > maxSampleCountDatasetBytes {
		dataset = dataset[:maxSampleCountDatasetBytes]
	}
	return dataset
}

func toJSON(d any) (*bytes.Buffer, error) {
	c, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(c), nil
}

func (s *Serializer) createLRUs() error {
	s.loadLRUsOnce.Do(func() {
		var err error

		// Create LRUs with MinILMRolloverTime as lifetime to avoid losing data by ILM roll-over.
		s.knownTraces, err = lru.NewLRUSet(knownTracesCacheSize, minILMRolloverTime)
		if err != nil {
			s.lruErr = fmt.Errorf("failed to create traces LRU: %w", err)
			return
		}

		s.knownFrames, err = lru.NewLRUSet(knownFramesCacheSize, minILMRolloverTime)
		if err != nil {
			s.lruErr = fmt.Errorf("failed to create frames LRU: %w", err)
			return
		}

		s.knownExecutables, err = lru.NewLRUSet(knownExecutablesCacheSize, minILMRolloverTime)
		if err != nil {
			s.lruErr = fmt.Errorf("failed to create executables LRU: %w", err)
			return
		}

		s.knownUnsymbolizedFrames, err = lru.NewLRUSet(knownUnsymbolizedFramesCacheSize, minILMRolloverTime)
		if err != nil {
			s.lruErr = fmt.Errorf("failed to create unsymbolized frames LRU: %w", err)
			return
		}

		s.knownUnsymbolizedExecutables, err = lru.NewLRUSet(knownUnsymbolizedExecutablesCacheSize, minILMRolloverTime)
		if err != nil {
			s.lruErr = fmt.Errorf("failed to create unsymbolized executables LRU: %w", err)
			return
		}

		s.knownHosts, err = lru.NewLRUSet(knownHostsCacheSize, minILMRolloverTime)
		if err != nil {
			s.lruErr = fmt.Errorf("failed to create hosts LRU: %w", err)
			return
		}
	})

	return s.lruErr
}
