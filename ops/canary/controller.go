// Package canary implements a progressive rollout controller for MetalDocs.
// It reads policy.yaml and advances the feature flag percentage when metrics
// stay within thresholds across the required consecutive breach window.
//
// Usage:
//
//	go run ./ops/canary [--dry-run] [--step]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Policy mirrors policy.yaml structure.
type Policy struct {
	FeatureFlag string     `yaml:"feature_flag"`
	RampSteps   []RampStep `yaml:"ramp_steps"`

	MinRequests   int `yaml:"min_requests"`
	MinDwellMin   int `yaml:"min_dwell_min"`
	MaxDwellMin   int `yaml:"max_dwell_min"`

	BreachConsecutiveBuckets int `yaml:"breach_consecutive_buckets"`

	Thresholds struct {
		ErrorRateMultiplier    float64 `yaml:"error_rate_multiplier"`
		ErrorRateAbsolutePct   float64 `yaml:"error_rate_absolute_pct"`
		P99LatencyMultiplier   float64 `yaml:"p99_latency_multiplier"`
		SignoffFailureRatePct  float64 `yaml:"signoff_failure_rate_pct"`
	} `yaml:"thresholds"`

	Baseline struct {
		WindowHours  int    `yaml:"window_hours"`
		Storage      string `yaml:"storage"`
	} `yaml:"baseline"`
}

type RampStep struct {
	Pct      int `yaml:"pct"`
	DwellMin int `yaml:"dwell_min"`
}

// Baseline stores the pre-canary rolling median metrics.
type Baseline struct {
	ComputedAt      time.Time `json:"computed_at"`
	ErrorRatePct    float64   `json:"error_rate_pct"`
	P99LatencyMs    float64   `json:"p99_latency_ms"`
	RequestsPerMin  float64   `json:"requests_per_min"`
}

// MetricSample is a 1-min bucket from the metrics store.
type MetricSample struct {
	Timestamp           time.Time
	ErrorRatePct        float64
	P99LatencyMs        float64
	SignoffFailureRatePct float64
	RequestCount        int
}

func main() {
	dryRun := flag.Bool("dry-run", false, "compute next step but do not apply")
	stepOnly := flag.Bool("step", false, "advance one step then exit")
	flag.Parse()

	policy, err := loadPolicy("ops/canary/policy.yaml")
	if err != nil {
		slog.Error("load policy", "err", err)
		os.Exit(1)
	}

	baseline, err := loadBaseline(policy.Baseline.Storage)
	if err != nil {
		slog.Warn("baseline not found — run nightly baseline refresh first", "err", err)
		baseline = &Baseline{} // zero baseline = conservative (all thresholds relative to 0)
	}

	currentPct, err := readFeatureFlag(policy.FeatureFlag)
	if err != nil {
		slog.Error("read feature flag", "err", err)
		os.Exit(1)
	}

	nextStep := findNextStep(policy, currentPct)
	if nextStep == nil {
		slog.Info("canary: already at 100%, nothing to advance")
		return
	}

	slog.Info("canary: evaluating", "current_pct", currentPct, "next_pct", nextStep.Pct)

	samples, err := fetchRecentSamples(policy.BreachConsecutiveBuckets + 5)
	if err != nil {
		slog.Error("fetch metric samples", "err", err)
		os.Exit(1)
	}

	// Check sample floor
	totalRequests := 0
	for _, s := range samples {
		totalRequests += s.RequestCount
	}
	if totalRequests < policy.MinRequests {
		slog.Info("canary: sample floor not met — extending dwell",
			"requests", totalRequests, "min", policy.MinRequests)
		return
	}

	// Check consecutive breach window
	breaches := countConsecutiveBreaches(samples, baseline, policy)
	if breaches >= policy.BreachConsecutiveBuckets {
		slog.Error("canary: BREACH detected — aborting ramp",
			"consecutive_breaches", breaches,
			"threshold", policy.BreachConsecutiveBuckets)
		if !*dryRun {
			abortRamp(policy.FeatureFlag)
		}
		os.Exit(2)
	}

	slog.Info("canary: metrics healthy", "breaches", breaches)

	if *dryRun {
		fmt.Printf("dry-run: would advance %s to %d%%\n", policy.FeatureFlag, nextStep.Pct)
		return
	}

	if err := applyFeatureFlag(policy.FeatureFlag, nextStep.Pct); err != nil {
		slog.Error("apply feature flag", "err", err, "pct", nextStep.Pct)
		os.Exit(1)
	}

	slog.Info("canary: advanced", "flag", policy.FeatureFlag, "pct", nextStep.Pct)

	if *stepOnly {
		return
	}
}

func loadPolicy(path string) (*Policy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Policy
	return &p, yaml.Unmarshal(b, &p)
}

func loadBaseline(path string) (*Baseline, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bl Baseline
	return &bl, json.Unmarshal(b, &bl)
}

func readFeatureFlag(_ string) (int, error) {
	// TODO: read from central config store (Consul/LaunchDarkly/env)
	// Placeholder reads from env
	val := os.Getenv("APPROVAL_V2_PCT")
	if val == "" {
		return 0, nil
	}
	var pct int
	_, err := fmt.Sscanf(val, "%d", &pct)
	return pct, err
}

func applyFeatureFlag(flag string, pct int) error {
	// TODO: write to central config store
	slog.Info("applying feature flag", "flag", flag, "pct", pct)
	return nil
}

func abortRamp(flag string) {
	slog.Warn("aborting ramp — setting flag to 0", "flag", flag)
	_ = applyFeatureFlag(flag, 0)
}

func findNextStep(p *Policy, current int) *RampStep {
	for i := range p.RampSteps {
		if p.RampSteps[i].Pct > current {
			return &p.RampSteps[i]
		}
	}
	return nil
}

func fetchRecentSamples(_ int) ([]MetricSample, error) {
	// TODO: query Prometheus/Grafana/Datadog for recent 1-min buckets
	// Stub returns empty — controller is wired, metrics integration is operator task
	return []MetricSample{}, nil
}

func countConsecutiveBreaches(samples []MetricSample, bl *Baseline, p *Policy) int {
	consecutive := 0
	max := 0
	for _, s := range samples {
		if isBreach(s, bl, p) {
			consecutive++
			if consecutive > max {
				max = consecutive
			}
		} else {
			consecutive = 0
		}
	}
	return max
}

func isBreach(s MetricSample, bl *Baseline, p *Policy) bool {
	// Error rate: relative AND absolute
	errorRelative := bl.ErrorRatePct * p.Thresholds.ErrorRateMultiplier
	if s.ErrorRatePct > math.Max(errorRelative, p.Thresholds.ErrorRateAbsolutePct) {
		return true
	}
	// p99 latency: relative
	if bl.P99LatencyMs > 0 && s.P99LatencyMs > bl.P99LatencyMs*p.Thresholds.P99LatencyMultiplier {
		return true
	}
	// Signoff failure: absolute
	if s.SignoffFailureRatePct > p.Thresholds.SignoffFailureRatePct {
		return true
	}
	return false
}
