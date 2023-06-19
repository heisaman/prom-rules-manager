package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
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
	ruleGroups, errors := ParseFile(ruleGroupsStr)
	if len(errors) > 0 {
		fmt.Println(errors)
	}
	return &RulesManager{ruleGroups}
}

func (manager *RulesManager) AddRule(group string) error {
	fmt.Println(manager.ruleGroups)
	fmt.Println("AddRule...")
	return nil
}

func (manager *RulesManager) RemoveRule(group string) error {
	fmt.Println(manager.ruleGroups)
	fmt.Println("AddRule...")
	return nil
}

func (manager *RulesManager) updateConfigMap() error {
	// clientset.CoreV1().ConfigMaps("monitoring").Patch()
	return nil
}
