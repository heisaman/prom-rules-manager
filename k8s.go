package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/alecthomas/kingpin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	rulefileConfigMap = "prometheus-rulefile-custom"
	rulefileName      = "rules.yml"
)

var (
	ruleGroupsStr string
	clientset     *kubernetes.Clientset
)

func init() {
	kingpin.Parse()
	fmt.Println("k8s.go standaloneMode: " + strconv.FormatBool(*standaloneMode))
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

	// specify namespace to get cm in particular namespace
	rulesConfig, err := clientset.CoreV1().ConfigMaps("monitoring").Get(context.TODO(), rulefileConfigMap, metav1.GetOptions{})

	if err != nil {
		panic(err.Error())
	}
	ruleGroupsStr = rulesConfig.Data[rulefileName]
}

type RulesManager struct {
	ruleGroups *RuleGroups
}

func NewRulesManager() *RulesManager {
	ruleGroups, errors := Parse([]byte(ruleGroupsStr))
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

func (manager *RulesManager) AddRule(group string, newRule Rule) error {
	fmt.Println(manager.ruleGroups)
	fmt.Println(fmt.Sprintf("AddRule...%+v", newRule))

	// Define the ConfigMap details
	// namespace := "your-namespace"
	// name := "your-configmap-name"
	// patch := `[{"op": "replace", "path": "/data/key", "value": "new-value"}]`

	// // Patch the ConfigMap
	// err := patchConfigMap(clientset, namespace, name, patch)
	// if err != nil {
	// 	fmt.Printf("Failed to patch ConfigMap: %v\n", err)
	// 	return err
	// }

	fmt.Println("ConfigMap patched successfully.")
	return nil
}

func (manager *RulesManager) RemoveRule(group string, oldRule Rule) error {
	fmt.Println(manager.ruleGroups)
	fmt.Println(fmt.Sprintf("RemoveRule...%+v", oldRule))
	return nil
}

func (manager *RulesManager) updateConfigMap() error {
	// clientset.CoreV1().ConfigMaps("monitoring").Patch()
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

func patchConfigMap(clientset *kubernetes.Clientset, namespace, name, patch string) error {
	// Prepare the patch data
	patchBytes := []byte(patch)

	// Create the PatchType object
	pt := types.JSONPatchType

	// Build the Patch object
	// patchObj := &corev1.ConfigMap{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      name,
	// 		Namespace: namespace,
	// 	},
	// }

	// Apply the patch
	_, err := clientset.CoreV1().ConfigMaps(namespace).Patch(context.TODO(), name, pt, patchBytes, metav1.PatchOptions{
		FieldManager: "client-go-patch",
	})
	if err != nil {
		return err
	}

	return nil
}
