package main

import (
	"fmt"
	"log"
	"time"

	klog "github.com/go-kit/log"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/rules"
)

var (
	groupLoader    = rules.FileLoader{}
	filename       = "rules.yaml"
	interval       = 10 * time.Second
	externalLabels labels.Labels
	externalURL    string
	logger         klog.Logger
)

func main() {
	rgs, errs := groupLoader.Load(filename)
	if errs != nil {
		log.Fatal(errs)
	}

	// opts := &ManagerOptions{
	// 	QueryFunc:       EngineQueryFunc(suite.QueryEngine(), suite.Storage()),
	// 	Appendable:      suite.Storage(),
	// 	Queryable:       suite.Storage(),
	// 	Context:         context.Background(),
	// 	Logger:          log.NewNopLogger(),
	// 	NotifyFunc:      func(ctx context.Context, expr string, alerts ...*Alert) {},
	// 	OutageTolerance: 30 * time.Minute,
	// 	ForGracePeriod:  10 * time.Minute,
	// }
	opts := &rules.ManagerOptions{}

	groups := make(map[string]*rules.Group)
	for _, rg := range rgs.Groups {
		itv := interval
		if rg.Interval != 0 {
			itv = time.Duration(rg.Interval)
		}

		groupRules := make([]rules.Rule, 0, len(rg.Rules))
		for _, r := range rg.Rules {
			expr, err := groupLoader.Parse(r.Expr.Value)
			if err != nil {
				log.Fatal(fmt.Errorf("%s: %w", filename, err))
			}

			if r.Alert.Value != "" {
				groupRules = append(groupRules, rules.NewAlertingRule(
					r.Alert.Value,
					expr,
					time.Duration(r.For),
					time.Duration(r.KeepFiringFor),
					labels.FromMap(r.Labels),
					labels.FromMap(r.Annotations),
					externalLabels,
					externalURL,
					false,
					klog.With(logger, "alert", r.Alert),
				))

				fmt.Println(fmt.Sprintf("%+v", groupRules[0]))
				continue
			}
			groupRules = append(groupRules, rules.NewRecordingRule(
				r.Record.Value,
				expr,
				labels.FromMap(r.Labels),
			))
		}

		groups[rules.GroupKey(filename, rg.Name)] = rules.NewGroup(rules.GroupOptions{
			Name:              rg.Name,
			File:              filename,
			Interval:          itv,
			Limit:             rg.Limit,
			Rules:             groupRules,
			ShouldRestore:     false,
			Opts:              opts,
			EvalIterationFunc: nil,
		})
	}

	fmt.Println(fmt.Sprintf("%+v", groups))
}
