// Copyright 2023 Massimo Comuzzo, Michele Pasqua and Marino Miculan
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"testing"

	"github.com/abu-lang/goabu/config"
)

func TestStop(t *testing.T) {
	const port = 11100
	a := NewBuilder().WithPort(port).WithLoggerOptions(config.TestsLogConfig).WithMdns("TestStop").Build()
	if a.Stop() == nil {
		t.Error("should return error when agent is not running")
	}
	start(t, a, port)
	restart(t, a, port)
	restart(t, a, port)
	stop(t, a)
}

func start(t *testing.T, a *Agent, p int) {
	t.Helper()
	err := a.Start()
	if err != nil {
		t.Fatal(err.Error())
	}
	checkCorrectStart(t, a, p)
}

func stop(t *testing.T, a *Agent) {
	t.Helper()
	err := a.Stop()
	if err != nil {
		t.Fatal(err.Error())
	}
	checkCorrectStop(t, a)
}

func restart(t *testing.T, a *Agent, p int) {
	t.Helper()
	stop(t, a)
	start(t, a, p)
}

func checkCorrectStart(t *testing.T, a *Agent, p int) {
	t.Helper()
	if !a.IsRunning() {
		t.Error("should be running")
	}
	switch {
	case a.mDns == nil:
		t.Error("mDns should not be nil")
	case a.topic == nil:
		t.Error("topic should not be nil")
	case a.subscription == nil:
		t.Error("subscription should not be nil")
	case a.stopReceive == nil:
		t.Error("stopReceive should not be nil")
	}
}

func checkCorrectStop(t *testing.T, a *Agent) {
	t.Helper()
	if a.IsRunning() {
		t.Error("should not be running")
	}
	switch {
	case a.stopReceive != nil:
		t.Error("stopReceive should be nil")
	case a.subscription != nil:
		t.Error("subscription should be nil")
	case a.topic != nil:
		t.Error("topic should be nil")
	case a.mDns != nil:
		t.Error("mDns should be nil")
	}
}
