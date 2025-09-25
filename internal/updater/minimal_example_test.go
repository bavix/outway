package updater_test

import (
	"context"
	"fmt"
	"log"

	"github.com/bavix/outway/internal/updater"
)

// ExampleNew demonstrates how to create a new updater instance.
func ExampleNew() {
	config := updater.Config{
		Owner:          "golang",
		Repo:           "go",
		CurrentVersion: "go1.21.0",
		BinaryName:     "go",
	}

	u, err := updater.New(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created updater for %s/%s\n", config.Owner, config.Repo)

	_ = u
	// Output: Created updater for golang/go
}

// ExampleUpdater_CheckForUpdates demonstrates how to check for updates.
func DemoUpdater_CheckForUpdates() {
	config := updater.Config{
		Owner:          "golang",
		Repo:           "go",
		CurrentVersion: "go1.21.0",
		BinaryName:     "go",
	}

	u, err := updater.New(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	updateInfo, err := u.CheckForUpdates(ctx, false)
	if err != nil {
		log.Fatal(err)
	}

	_ = updateInfo // example without deterministic output
}

// customLogger is an example logger implementation for the updater.
type customLogger struct{}

func (l *customLogger) Debugf(format string, args ...any) {
	_ = fmt.Sprintf("DEBUG: "+format+"\n", args...)
}

func (l *customLogger) Infof(format string, args ...any) {
	_ = fmt.Sprintf("INFO: "+format+"\n", args...)
}

func (l *customLogger) Warnf(format string, args ...any) {
	_ = fmt.Sprintf("WARN: "+format+"\n", args...)
}

func (l *customLogger) Errorf(format string, args ...any) {
	_ = fmt.Sprintf("ERROR: "+format+"\n", args...)
}

// ExampleLogger demonstrates how to implement a custom logger.
func ExampleLogger() {
	config := updater.Config{
		Owner:          "myorg",
		Repo:           "myapp",
		CurrentVersion: "v1.0.0",
		BinaryName:     "myapp",
		Logger:         &customLogger{},
	}

	u, err := updater.New(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created updater with custom logger\n")

	_ = u
	// Output: Created updater with custom logger
}
