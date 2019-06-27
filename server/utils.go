package server

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dapperlabs/bamboo-emulator/data"
	"github.com/dapperlabs/bamboo-emulator/nodes/access"
	"github.com/dapperlabs/bamboo-emulator/nodes/security"
)

type Config struct {
	Port               int           `default:"5000" flag:"port"`
	CollectionInterval time.Duration `default:"1s"`
	BlockInterval      time.Duration `default:"5s"`
	Verbose            bool          `default:"false" flag:"verbose"`
}

// StartServer starts up an instance of a Bamboo Emulator server.
func StartServer(log *logrus.Logger, conf Config) {
	log.WithField("port", conf.Port).Info("Starting emulator server...")

	collections := make(chan *data.Collection, 16)

	state, err := data.NewWorldState(log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize world state")
	}

	accessNode := access.NewNode(
		&access.Config{
			CollectionInterval: conf.CollectionInterval,
			Verbose: conf.Verbose,
		},
		state,
		collections,
		log,
	)
	securityNode := security.NewNode(
		&security.Config{
			BlockInterval: conf.BlockInterval,
			Verbose: conf.Verbose,
		},
		state,
		collections,
		log,
	)

	emulatorServer := NewServer(accessNode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go accessNode.Start(ctx)
	go securityNode.Start(ctx)

	emulatorServer.Start(conf.Port)
}
