package main

import (
	goflag "flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"github.com/songgao/water"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var gid uint
var uid uint
var tapName string

func createTapDevice(name string, uid uint, gid uint, isMultiqueue bool) error {
	var err error = nil
	config := water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:    name,
			Persist: true,
			Permissions: &water.DevicePermissions{
				Owner: uid,
				Group: gid,
			},
			MultiQueue: isMultiqueue,
		},
	}

	_, err = water.New(config)
	return err
}

func createTapDeviceOnPIDNetNs(launcherPid string, tapName string, uid uint, gid uint) {
	netns, err := ns.GetNS(fmt.Sprintf("/proc/%s/ns/net", launcherPid))

	if err != nil {
		glog.Fatalf("Could not load netns: %+v", err)
	} else if netns != nil {
		glog.V(4).Info("Successfully loaded netns ...")

		err = netns.Do(func(_ ns.NetNS) error {
			if err := createTapDevice(tapName, uid, gid, false); err != nil {
				glog.Fatalf("error creating tap device: %v", err)
			}

			glog.V(4).Infof("Managed to create the tap device in pid %s", launcherPid)
			return nil
		})
	}
}

func init() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func main() {
	rootCmd := &cobra.Command{
		Use: "tap-maker",
		Run: func(cmd *cobra.Command, args []string) {
			goflag.Parse()
		},
	}

	rootCmd.PersistentFlags().StringVar(&tapName, "tap-name", "tap0", "the name of the tap device")
	rootCmd.PersistentFlags().UintVar(&gid, "gid", 0, "the owner GID of the tap device")
	rootCmd.PersistentFlags().UintVar(&uid, "uid", 0, "the owner UID of the tap device")

	createTapCmd := &cobra.Command{
		Use:   "create-tap",
		Short: "create a tap device in a given PID net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := cmd.Flag("tap-name").Value.String()
			launcherPID := cmd.Flag("launcher-pid").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			gidStr := cmd.Flag("gid").Value.String()

			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return err
			}
			gid, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return err
			}

			glog.V(4).Infof("Executing in netns of pid %s", launcherPID)
			createTapDeviceOnPIDNetNs(launcherPID, tapName, uint(uid), uint(gid))

			return nil
		},
	}

	createTapCmd.Flags().StringP("launcher-pid", "p", "", "specify the PID holding the netns where the tap device will be created")
	if err := createTapCmd.MarkFlagRequired("launcher-pid"); err != nil {
		os.Exit(1)
	}

	consumeTapCmd := &cobra.Command{
		Use:   "consume-tap",
		Short: "consume a tap device in the current net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := cmd.Flag("tap-name").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			gidStr := cmd.Flag("gid").Value.String()

			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return err
			}
			gid, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return err
			}

			glog.V(4).Info("Will consume tap device named: ")
			err = createTapDevice(tapName, uint(uid), uint(gid), false)
			if err != nil {
				glog.Fatalf("Could not open the tapsy-thingy: %v", err)
			}

			glog.V(4).Infof("Opened the tap device on pid %d", os.Getpid())
			for {
				time.Sleep(time.Second)
			}
		},
	}

	rootCmd.AddCommand(createTapCmd, consumeTapCmd)
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
