// Copyright 2023 Massimo Comuzzo, Michele Pasqua and Marino Miculan
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"github.com/abu-lang/goabu/config"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// NewBuilder creates a new [Agent] builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Builder can be used for [Agent] creation.
type Builder struct {
	// port specifies the listening port.
	port int
	// discoveryTag specifies the string used for the mDNS discovery.
	discoveryTag string
	// pubOpts contains the message publishing options.
	pubOpts []pubsub.PubOpt
	// logConf contains the logger's configuration.
	logConf config.LogConfig
}

// WithPort configures the [Agent] to use a given listening port.
func (b *Builder) WithPort(port int) *Builder {
	b.port = port
	return b
}

// WithPublishOptions configures the [Agent] to pass the given options when a message is published.
func (b *Builder) WithPublishOptions(opts ...pubsub.PubOpt) *Builder {
	b.pubOpts = append(b.pubOpts, opts...)
	return b
}

// WithMdns set ups mDNS discovery with the passed non empty discovery tag string.
func (b *Builder) WithMdns(tag string) *Builder {
	b.discoveryTag = tag
	return b
}

// WithLoggerOptions configures the [Agent]'s logger.
func (b *Builder) WithLoggerOptions(conf config.LogConfig) *Builder {
	b.logConf = conf
	return b
}
