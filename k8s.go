package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kingpin"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	namespace         = "monitoring"
	rulefileConfigMap = "prometheus-rulefile-custom"
	rulefileName      = "rules.yml"
)

var (
	clientset *kubernetes.Clientset
)

func init() {
	kingpin.Parse()
	// Get the clientset
	var err error
	if *standaloneMode {
		clientset, err = getOutOfClusterClient()
	} else {
		clientset, err = getInClusterClient()
	}
	if err != nil {
		panic(fmt.Sprintf("Failed to get clientset: %v\n", err))
	}
}

type RulesManager struct {
	ruleGroups *RuleGroups
}

func NewRulesManager() *RulesManager {
	// specify namespace to get cm in particular namespace
	rulesConfig, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), rulefileConfigMap, metav1.GetOptions{})

	if err != nil {
		panic(err.Error())
	}
	ruleGroups, errors := Parse([]byte(rulesConfig.Data[rulefileName]))
	if len(errors) > 0 {
		fmt.Println(errors)
	}

	// opts := &rules.ManagerOptions{}

	// groups := make(map[string]*rules.Group)
	// for _, rg := range rgs.Groups {
	// 	itv := interval
	// 	if rg.Interval != 0 {
	// 		itv = time.Duration(rg.Interval)
	// 	}

	// 	groupRules := make([]rules.Rule, 0, len(rg.Rules))
	// 	for _, r := range rg.Rules {
	// 		expr, err := groupLoader.Parse(r.Expr.Value)
	// 		if err != nil {
	// 			panic(fmt.Errorf("%s: %w", filename, err))
	// 		}

	// 		if r.Alert.Value != "" {
	// 			groupRules = append(groupRules, rules.NewAlertingRule(
	// 				r.Alert.Value,
	// 				expr,
	// 				time.Duration(r.For),
	// 				time.Duration(r.KeepFiringFor),
	// 				labels.FromMap(r.Labels),
	// 				labels.FromMap(r.Annotations),
	// 				externalLabels,
	// 				externalURL,
	// 				false,
	// 				log.With(logger, "alert", r.Alert),
	// 			))

	// 			fmt.Println(fmt.Sprintf("%+v", groupRules[0]))
	// 			continue
	// 		}
	// 		groupRules = append(groupRules, rules.NewRecordingRule(
	// 			r.Record.Value,
	// 			expr,
	// 			labels.FromMap(r.Labels),
	// 		))
	// 	}

	// 	groups[rules.GroupKey(filename, rg.Name)] = rules.NewGroup(rules.GroupOptions{
	// 		Name:              rg.Name,
	// 		File:              filename,
	// 		Interval:          itv,
	// 		Limit:             rg.Limit,
	// 		Rules:             groupRules,
	// 		ShouldRestore:     false,
	// 		Opts:              opts,
	// 		EvalIterationFunc: nil,
	// 	})
	// }

	// fmt.Println(fmt.Sprintf("%+v", groups))

	return &RulesManager{ruleGroups}
}

func (manager *RulesManager) AddRules(newRuleGroup SimpleRuleGroup) error {
	fmt.Println(fmt.Sprintf("AddRules: %+v\n", newRuleGroup))

	for i, ruleGroup := range manager.ruleGroups.Groups {
		fmt.Println(fmt.Sprintf("In forrange, ruleGroup %s address: %p\n", ruleGroup.Name, &ruleGroup))
		if ruleGroup.Name == newRuleGroup.Name {
			for _, existingRule := range ruleGroup.Rules {
				for j, newRule := range newRuleGroup.Rules {
					if existingRule.Alert.Value == newRule.Alert && existingRule.Expr.Value == newRule.Expr {
						// Update an old rule
						existingRule.For = newRule.For
						existingRule.KeepFiringFor = newRule.KeepFiringFor
						existingRule.Labels = newRule.Labels
						existingRule.Annotations = newRule.Annotations
						newRuleGroup.Rules = slices.Delete(newRuleGroup.Rules, j, j+1)
						break
					}
				}
			}
			for _, newRule := range newRuleGroup.Rules {
				// Add a new rule
				newNodeRule := RuleNode{
					For:           newRule.For,
					KeepFiringFor: newRule.KeepFiringFor,
					Labels:        newRule.Labels,
					Annotations:   newRule.Annotations,
				}
				var recordNode, alertNode, exprNode yaml.Node
				exprNode.SetString(newRule.Expr)
				newNodeRule.Expr = exprNode
				if newRule.Alert != "" {
					alertNode.SetString(newRule.Alert)
					newNodeRule.Alert = alertNode
				}
				if newRule.Record != "" {
					recordNode.SetString(newRule.Record)
					newNodeRule.Record = recordNode
				}
				// ruleGroup.Rules = append(ruleGroup.Rules, newNodeRule)
				manager.ruleGroups.Groups[i].Rules = append(manager.ruleGroups.Groups[i].Rules, newNodeRule)
				fmt.Println(fmt.Sprintf("ruleGroup appended a newNodeRule: %+v\n", ruleGroup))
			}
		}
	}

	// Patch the ConfigMap
	rulesData, err := yaml.Marshal(manager.ruleGroups)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("manager.ruleGroups: %+v", manager.ruleGroups))
	dataValue := map[string]string{
		rulefileName: string(rulesData),
	}
	err = manager.updateConfigMap(map[string]interface{}{"data": dataValue})
	if err != nil {
		fmt.Printf("Failed to patch ConfigMap: %v\n", err)
		return err
	}

	fmt.Println("Custom rules configmap patched successfully.")
	return nil
}

func (manager *RulesManager) RemoveRules(newRuleGroup SimpleRuleGroup) error {
	fmt.Println(fmt.Sprintf("RemoveRules: %+v\n", newRuleGroup))

	for i, ruleGroup := range manager.ruleGroups.Groups {
		fmt.Println(fmt.Sprintf("In forrange, ruleGroup %s address: %p\n", ruleGroup.Name, &ruleGroup))
		if ruleGroup.Name == newRuleGroup.Name {
			for j, existingRule := range ruleGroup.Rules {
				for _, newRule := range newRuleGroup.Rules {
					if existingRule.Alert.Value == newRule.Alert && existingRule.Expr.Value == newRule.Expr {
						// Delete an old rule
						manager.ruleGroups.Groups[i].Rules = slices.Delete(manager.ruleGroups.Groups[i].Rules, j, j+1)
						break
					}
				}
			}
		}
	}

	// Patch the ConfigMap
	rulesData, err := yaml.Marshal(manager.ruleGroups)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("manager.ruleGroups: %+v", manager.ruleGroups))
	dataValue := map[string]string{
		rulefileName: string(rulesData),
	}
	err = manager.updateConfigMap(map[string]interface{}{"data": dataValue})
	if err != nil {
		fmt.Printf("Failed to patch ConfigMap: %v\n", err)
		return err
	}

	fmt.Println("Custom rules configmap patched successfully.")
	return nil
}

func (manager *RulesManager) updateConfigMap(patchData map[string]interface{}) error {
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return err
	}

	// Create the PatchType object
	patchType := types.MergePatchType

	// Apply the patch
	_, err = clientset.CoreV1().ConfigMaps(namespace).Patch(context.TODO(), rulefileConfigMap, patchType, patchBytes, metav1.PatchOptions{
		FieldManager: "client-go-patch",
	})
	if err != nil {
		return err
	}
	return nil
}

func getInClusterClient() (*kubernetes.Clientset, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func getOutOfClusterClient() (*kubernetes.Clientset, error) {
	// Assumes the kubeconfig file is present at the default location (~/.kube/config)
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	if err != nil {
		return nil, err
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
