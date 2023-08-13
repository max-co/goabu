// Copyright 2023 Massimo Comuzzo, Michele Pasqua and Marino Miculan
// SPDX-License-Identifier: Apache-2.0

// Package agent provides an experimental and incomplete implementation of [github.com/abu-lang/goabu.Agent]
// based on [github.com/libp2p/go-libp2p-pubsub].
package agent

import (
	"context"
	"errors"
	"strconv"

	logconf "github.com/abu-lang/goabu/config"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Agent is an implementation of [github.com/abu-lang/goabu.Agent] based on [github.com/libp2p/go-libp2p-pubsub].
type Agent struct {
	listenAddr   config.Option
	discoveryTag string
	pubOpts      []pubsub.PubOpt

	ctx          context.Context
	host         host.Host
	mDns         mdns.Service
	pubsub       *pubsub.PubSub
	topic        *pubsub.Topic
	subscription *pubsub.Subscription
	stopReceive  func() <-chan struct{}

	running           bool
	operations        chan chan []byte
	operationCommands chan chan string
	logLevel          zap.AtomicLevel
	logger            *zap.Logger
}

// Build creates a new [Agent]
func (b *Builder) Build() *Agent {
	res := &Agent{listenAddr: libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/" + strconv.Itoa(b.port)),
		discoveryTag:      b.discoveryTag,
		operations:        make(chan chan []byte),
		operationCommands: make(chan chan string),
		pubOpts:           b.pubOpts,
	}
	if b.logConf.Encoding == "" {
		b.logConf.Encoding = "console"
	}
	zapCfg, ok := logconf.LogPreset(b.logConf.Encoding).(zap.Config)
	if !ok {
		zapCfg = zap.NewProductionConfig()
	}
	res.logLevel = zapCfg.Level
	var err error
	res.logger, err = zapCfg.Build()
	if err != nil {
		if ok { // fallback to zap default
			ok = false
			zapCfg = zap.NewProductionConfig()
			res.logLevel = zapCfg.Level
			res.logger, err = zapCfg.Build()
		}
		if err != nil {
			panic(err)
		}
	}
	res.SetLogLevel(b.logConf.Level)
	return res
}

// Start brings the [Agent] to a state where it is ready to accept incoming connections from other nodes and possibly
// starts mDNS node discovery.
func (a *Agent) Start() error {
	if a.running {
		return errors.New("agent is already running")
	}
	a.ctx = context.Background()
	var err error
	if a.host == nil {
		a.host, err = libp2p.New(a.listenAddr)
		if err != nil {
			return err
		}
		a.pubsub, err = pubsub.NewGossipSub(a.ctx, a.host)
		if err != nil {
			return err
		}
	}
	if a.discoveryTag != "" {
		a.mDns = mdns.NewMdnsService(a.host, a.discoveryTag, notifee{a})
		err = a.mDns.Start()
		if err != nil {
			return err
		}
	}
	a.topic, err = a.pubsub.Join("allUpdates")
	if err != nil {
		return err
	}
	a.subscription, err = a.topic.Subscribe()
	if err != nil {
		return err
	}
	cancelCtx, cancelFunc := context.WithCancel(a.ctx)
	done := a.receiveUpdates(cancelCtx)
	a.stopReceive = func() <-chan struct{} {
		cancelFunc()
		return done
	}
	a.running = true
	return nil
}

// receiveUpdates handles the reception of updates coming from other nodes.
// The function returns a channel which signals if update handling is stopped.
func (a *Agent) receiveUpdates(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func(done chan<- struct{}) {
		defer func() {
			done <- struct{}{}
		}()
		for {
			msg, err := a.subscription.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, pubsub.ErrSubscriptionCancelled) {
					a.logger.Debug("stopping receiving updates: " + err.Error())
				} else {
					a.logger.Error(err.Error())
				}
			}
			if msg == nil {
				return
			}
			if err != nil || msg.ReceivedFrom == a.host.ID() {
				continue
			}
			actionsCh := make(chan []byte)
			commandsCh := make(chan string)
			a.operations <- actionsCh
			a.operationCommands <- commandsCh
			actionsCh <- msg.GetData()
			if <-commandsCh == "interested" {
				commandsCh <- "can_commit?"
				if <-commandsCh == "prepared" {
					commandsCh <- "do_commit"
					<-commandsCh
				}
			}
		}
	}(done)
	return done
}

// IsRunning checks if the [Agent] is operational.
func (a *Agent) IsRunning() bool {
	return a.running
}

// Join is a no-op.
func (a *Agent) Join() error {
	return nil
}

// ForAll publishes the given payload.
func (a *Agent) ForAll(payload []byte) error {
	if !a.running {
		return errors.New("agent is not running")
	}
	return a.topic.Publish(a.ctx, payload, a.pubOpts...)
}

// ReceivedActions returns two channels passing a channel each for every received update.
// The []byte channel passes the marshalled update while the string channel is used for interacting
// with the [github.com/abu-lang/goabu.Executer].
func (a *Agent) ReceivedActions() (<-chan chan []byte, <-chan chan string) {
	return a.operations, a.operationCommands
}

// Stop brings the [Agent] to an halted stated by bringing down the connections to the other nodes.
func (a *Agent) Stop() error {
	if !a.running {
		return errors.New("agent is not running")
	}
	if a.stopReceive != nil {
		done := a.stopReceive()
		defer func() {
			<-done
			a.stopReceive = nil
			a.subscription = nil
		}()
	}
	a.subscription.Cancel()
	err := a.topic.Close()
	if err != nil {
		return err
	}
	a.topic = nil
	err = a.mDns.Close()
	if err != nil {
		return err
	}
	a.mDns = nil
	a.running = false
	return nil
}

// SetLogLevel sets the log level of the [Agent]'s logger.
func (a *Agent) SetLogLevel(l int) {
	if l < logconf.LogDebug {
		l = logconf.LogDebug
	} else if l > logconf.LogFatal {
		l = logconf.LogFatal
	}
	zapLevel := zapcore.InfoLevel
	switch l {
	case logconf.LogDebug:
		zapLevel = zapcore.DebugLevel
	case logconf.LogWarning:
		zapLevel = zapcore.WarnLevel
	case logconf.LogError:
		zapLevel = zapcore.ErrorLevel
	case logconf.LogFatal:
		zapLevel = zapcore.DPanicLevel
	}
	a.logLevel.SetLevel(zapLevel)
}

// notifee handles connection to discovered other nodes by implementing the [mdns.Notifee] interface.
type notifee struct {
	*Agent
}

// HandlePeerFound starts a connection with the found peer.
func (a notifee) HandlePeerFound(found peer.AddrInfo) {
	a.logger.Info("Found new node through mDNS", zap.Stringer("nodeID", found.ID))
	err := a.host.Connect(a.ctx, found)
	if err != nil {
		a.logger.Error("Could not connect to node", zap.Stringer("nodeID", found.ID))
	}
}
