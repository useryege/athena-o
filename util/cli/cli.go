// Package cmd provides functionally common to various athena CLIs

package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/errors"
	utillog "github.com/useryege/athena/util/log"
)

// NewVersionCmd returns a new `version` command to be used as a sub-command to root
func NewVersionCmd(cliName string) *cobra.Command {
	var short bool
	versionCmd := cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			version := common.GetVersion()
			fmt.Printf("%s: %s\n", cliName, version)
			if short {
				return
			}
			fmt.Printf("  BuildDate: %s\n", version.BuildDate)
			fmt.Printf("  GitCommit: %s\n", version.GitCommit)
			fmt.Printf("  GitTreeState: %s\n", version.GitTreeState)
			if version.GitTag != "" {
				fmt.Printf("  GitTag: %s\n", version.GitTag)
			}
			fmt.Printf("  GoVersion: %s\n", version.GoVersion)
			fmt.Printf("  Compiler: %s\n", version.Compiler)
			fmt.Printf("  Platform: %s\n", version.Platform)
			if version.ExtraBuildInfo != "" {
				fmt.Printf("  ExtraBuildInfo: %s\n", version.ExtraBuildInfo)
			}
		},
	}
	versionCmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	return &versionCmd
}

// SetLogFormat sets a logrus log format
func SetLogFormat(logFormat string) {
	switch strings.ToLower(logFormat) {
	case utillog.JsonFormat:
		os.Setenv(common.EnvLogFormat, utillog.JsonFormat)
	case utillog.TextFormat, "":
		os.Setenv(common.EnvLogFormat, utillog.TextFormat)
	default:
		log.Fatalf("Unknown log format '%s'", logFormat)
	}

	log.SetFormatter(utillog.CreateFormatter(logFormat))
}

// SetLogLevel parses and sets a logrus log level
func SetLogLevel(logLevel string) {
	if logLevel == "" {
		logLevel = log.InfoLevel.String()
	}
	level, err := log.ParseLevel(logLevel)
	errors.CheckError(err)
	os.Setenv(common.EnvLogLevel, level.String())
	log.SetLevel(level)
}

// SetGLogLevel set the glog level for the k8s go-client
func SetGLogLevel(glogLevel int) {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "true")
	_ = flag.Set("v", strconv.Itoa(glogLevel))
}
