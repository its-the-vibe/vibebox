package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type commandRunner interface {
	Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

type gcloudRunner struct{}

func (gcloudRunner) Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(context.Background(), "gcloud", args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gcloud %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, gcloudRunner{})
}

func run(args []string, stdout, stderr io.Writer, runner commandRunner) int {
	cmd := newRootCommand(stdout, stderr, runner)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func newRootCommand(stdout, stderr io.Writer, runner commandRunner) *cobra.Command {
	root := &cobra.Command{
		Use:           "gsec",
		Short:         "Manage Google Cloud Secret Manager secrets",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(newViewCommand(runner))
	root.AddCommand(newPutCommand(runner))
	root.AddCommand(newVersionsCommand(runner))
	return root
}

func newViewCommand(runner commandRunner) *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "view <secret-name>",
		Short: "View a secret version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Run([]string{"secrets", "versions", "access", version, "--secret", args[0]}, nil, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringVar(&version, "version", "latest", "Secret version to view")
	return cmd
}

func newPutCommand(runner commandRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "put <secret-name> <key> <value>",
		Short: "Add or update a key in a JSON secret",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			var current bytes.Buffer
			if err := runner.Run([]string{"secrets", "versions", "access", "latest", "--secret", args[0]}, nil, &current, cmd.ErrOrStderr()); err != nil {
				return err
			}

			payload := make(map[string]json.RawMessage)
			if err := json.Unmarshal(current.Bytes(), &payload); err != nil || payload == nil {
				return fmt.Errorf("secret %q does not contain a JSON object", args[0])
			}
			value, err := json.Marshal(args[2])
			if err != nil {
				return err
			}
			payload[args[1]] = value
			updated, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			updated = append(updated, '\n')

			if err := runner.Run([]string{"secrets", "versions", "add", args[0], "--data-file=-"}, bytes.NewReader(updated), cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated secret %q\n", args[0])
			return nil
		},
	}
}

type secretVersion struct {
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
	State      string `json:"state"`
	number     string
	created    time.Time
}

func newVersionsCommand(runner commandRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions <secret-name>",
		Short: "List active secret versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			versions, err := listActiveVersions(runner, args[0], cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "VERSION\tCREATED")
			for _, version := range versions {
				fmt.Fprintf(out, "%s\t%s\n", version.number, version.CreateTime)
			}
			return nil
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "delete-oldest <secret-name>",
		Short: "Destroy the oldest active secret version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			versions, err := listActiveVersions(runner, args[0], cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if len(versions) == 0 {
				return fmt.Errorf("secret %q has no active versions", args[0])
			}
			oldest := versions[0]
			if err := runner.Run([]string{"secrets", "versions", "destroy", oldest.number, "--quiet", "--secret", args[0]}, nil, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Destroyed version %s of secret %q\n", oldest.number, args[0])
			return nil
		},
	})
	return cmd
}

func listActiveVersions(runner commandRunner, secret string, stderr io.Writer) ([]secretVersion, error) {
	var output bytes.Buffer
	if err := runner.Run([]string{"secrets", "versions", "list", secret, "--filter=state=ENABLED", "--format=json"}, nil, &output, stderr); err != nil {
		return nil, err
	}
	var versions []secretVersion
	if err := json.Unmarshal(output.Bytes(), &versions); err != nil {
		return nil, fmt.Errorf("parse versions for secret %q: %w", secret, err)
	}

	active := versions[:0]
	for _, version := range versions {
		if version.State != "ENABLED" {
			continue
		}
		version.number = version.Name[strings.LastIndex(version.Name, "/")+1:]
		if _, err := strconv.ParseUint(version.number, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid version name %q", version.Name)
		}
		var err error
		version.created, err = time.Parse(time.RFC3339Nano, version.CreateTime)
		if err != nil {
			return nil, fmt.Errorf("invalid creation time for version %q: %w", version.number, err)
		}
		active = append(active, version)
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].created.Equal(active[j].created) {
			left, _ := strconv.ParseUint(active[i].number, 10, 64)
			right, _ := strconv.ParseUint(active[j].number, 10, 64)
			return left < right
		}
		return active[i].created.Before(active[j].created)
	})
	return active, nil
}
