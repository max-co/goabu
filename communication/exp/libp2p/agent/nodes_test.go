// Copyright 2023 Massimo Comuzzo, Michele Pasqua and Marino Miculan
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"testing"
	"time"

	"github.com/abu-lang/goabu"
	"github.com/abu-lang/goabu/config"
	"github.com/abu-lang/goabu/memory"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/max-co/goabu/communication/exp/libp2p/agent"
)

func TestTwoNodes(t *testing.T) {
	memory := memory.MakeResources()
	memory.Integer["lorem"] = 5
	r := "rule r on lorem for all this.lorem > ext.lorem do ext.lorem = this.lorem, "
	rules := []string{r}
	t.Run("TestTwoNodes#1", func(t *testing.T) {
		e1, err := goabu.NewExecuter(memory, rules, NewTestMdnsAgent(9001, "TestTwoNodes", 2), config.TestsLogConfig)
		if err != nil {
			t.Fatal(err)
		}
		t.Parallel()
		for e1.DoIfStable(func() {}) {
		}
		e1.Exec()
		if !e1.DoIfStable(func() {}) {
			t.Error("should be stable")
		}
		mem1, _ := e1.TakeState()
		if mem1.Integer["lorem"] != 10 {
			t.Error("lorem should be 10")
		}
	})
	t.Run("TestTwoNodes#2", func(t *testing.T) {
		t.Parallel()
		a2 := NewTestMdnsAgent(9002, "TestTwoNodes", 2)
		e2, err := goabu.NewExecuter(memory, rules, a2, config.TestsLogConfig)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Second)
		e2.Input("lorem = 10, ")
		if !e2.DoIfStable(func() {}) {
			t.Error("should be stable")
		}
		mem2, _ := e2.TakeState()
		if mem2.Integer["lorem"] != 10 {
			t.Error("lorem should be 10")
		}
	})
}

func TestThreeNodes(t *testing.T) {
	memory := memory.MakeResources()
	memory.Float["ipsum"] = 3.0
	memory.Bool["involved"] = false
	r1 := "rule r1 on ipsum default involved = true for all ipsum != ext.ipsum do involved = true"
	r2 := "rule r2 on involved for all involved && ipsum > ext.ipsum do ipsum = this.ipsum"
	rules := []string{r1, r2}
	t.Run("TestThreeNodes#1", func(t *testing.T) {
		a1 := NewTestMdnsAgent(10001, "TestThreeNodes", 3)
		e1, err := goabu.NewExecuter(memory, rules, a1, config.TestsLogConfig)
		if err != nil {
			t.Fatal(err)
		}
		t.Parallel()
		mem1, _ := e1.TakeState()
		for mem1.Float["ipsum"] != 6.5 {
			e1.Exec()
			mem1, _ = e1.TakeState()
		}
		if mem1, _ = e1.TakeState(); !mem1.Bool["involved"] {
			t.Error("involved should be true")
		}
	})
	t.Run("TestThreeNodes#2", func(t *testing.T) {
		a2 := NewTestMdnsAgent(10002, "TestThreeNodes", 3)
		memory.Float["ipsum"] = 6.5
		e2, err := goabu.NewExecuter(memory, rules, a2, config.TestsLogConfig)
		if err != nil {
			t.Fatal(err)
		}
		t.Parallel()
		for e2.DoIfStable(func() {}) {
		}
		e2.Exec()
		mem2, _ := e2.TakeState()
		if !mem2.Bool["involved"] {
			t.Error("involved should be true")
		}
		if mem2.Float["ipsum"] != 6.5 {
			t.Error("ipsum should be 6.5")
		}
	})
	t.Run("TestThreeNodes#3", func(t *testing.T) {
		t.Parallel()
		a3 := NewTestMdnsAgent(10003, "TestThreeNodes", 3)
		memory.Float["ipsum"] = 3.0
		e3, err := goabu.NewExecuter(memory, rules, a3, config.TestsLogConfig)
		if err != nil {
			t.Fatal(err)
		}
		e3.Input("ipsum = 6.0,")
		mem3, _ := e3.TakeState()
		for mem3.Float["ipsum"] != 6.5 {
			e3.Exec()
			mem3, _ = e3.TakeState()
		}
		if mem3, _ = e3.TakeState(); !mem3.Bool["involved"] {
			t.Error("involved should be true")
		}
	})
}

// NewTestMdnsAgent creates a new [Agent] with mDNS discovery listening on the specified port and waiting for a number of connected nodes before publishing.
func NewTestMdnsAgent(port int, tag string, participants int) *agent.Agent {
	return agent.NewBuilder().WithPort(port).WithMdns(tag).WithLoggerOptions(config.TestsLogConfig).WithPublishOptions(pubsub.WithReadiness(pubsub.MinTopicSize(participants - 1))).Build()
}
