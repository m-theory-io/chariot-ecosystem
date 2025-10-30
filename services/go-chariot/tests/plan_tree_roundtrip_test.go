package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

// Ensures a Plan ('P') can be set as a TreeNode attribute, saved, and loaded without losing type or fields
func TestPlanInTree_Roundtrip(t *testing.T) {
	// Initialize any environment/config used by treeSave paths
	initCouchbaseConfig()

	tests := []TestCase{
		{
			Name: "Plan-in-Tree roundtrip preserves type P",
			Script: []string{
				`setq(trig, func() { true })`,
				`setq(guard, func() { true })`,
				`setq(steps, array(func() { 1 }))`,
				`setq(drop, func() { false })`,
				`setq(root, create('planTree'))`,
				`setq(p, plan('PlanInTree', array('a','b'), trig, guard, steps, drop))`,
				`setAttribute(root, 'plan', p)`,
				`treeSave(root, 'plan_tree.json')`,
				`setq(loaded, treeLoad('plan_tree.json'))`,
				`typeOf(getAttribute(loaded, 'plan'))`,
			},
			ExpectedValue: chariot.Str("P"),
		},
		{
			Name: "Plan-in-Tree roundtrip preserves name",
			Script: []string{
				`setq(trig, func() { true })`,
				`setq(guard, func() { true })`,
				`setq(steps, array(func() { 1 }))`,
				`setq(drop, func() { false })`,
				`setq(root, create('planTree'))`,
				`setq(p, plan('PlanInTree', array('a','b'), trig, guard, steps, drop))`,
				`setAttribute(root, 'plan', p)`,
				`treeSave(root, 'plan_tree.json')`,
				`setq(loaded, treeLoad('plan_tree.json'))`,
				`getProp(getAttribute(loaded, 'plan'), 'name')`,
			},
			ExpectedValue: chariot.Str("PlanInTree"),
		},
		{
			Name: "Plan-in-Tree roundtrip preserves params length",
			Script: []string{
				`setq(trig, func() { true })`,
				`setq(guard, func() { true })`,
				`setq(steps, array(func() { 1 }))`,
				`setq(drop, func() { false })`,
				`setq(root, create('planTree'))`,
				`setq(p, plan('PlanInTree', array('a','b'), trig, guard, steps, drop))`,
				`setAttribute(root, 'plan', p)`,
				`treeSave(root, 'plan_tree.json')`,
				`setq(loaded, treeLoad('plan_tree.json'))`,
				`length(getProp(getAttribute(loaded, 'plan'), 'params'))`,
			},
			ExpectedValue: chariot.Number(2),
		},
	}

	RunTestCases(t, tests)
}
