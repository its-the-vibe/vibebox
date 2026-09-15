package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const (
	defaultComposePath  = "docker-compose.yml"
	defaultOverridePath = "docker-compose.override.yml"
)

func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCommand(stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "docker-override",
		Short:         "Generate Docker Compose override files",
		SilenceErrors: true,
		SilenceUsage:  true,
		Example: "  docker-override create ghcr.io/its-the-vibe/app:latest\n" +
			"  docker-override create ghcr.io/its-the-vibe/app:latest compose.yml compose.override.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.AddCommand(newCreateCommand())
	rootCmd.AddCommand(newViewCommand())
	rootCmd.AddCommand(newDeleteCommand())
	return rootCmd
}

func newCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create <img> [docker-compose.yml] [docker-compose.override.yml]",
		Short: "Create a Docker Compose override file",
		Long:  "Create a Docker Compose override file by replacing every service definition with a shared image.",
		Example: "  docker-override create ghcr.io/its-the-vibe/app:latest\n" +
			"  docker-override create ghcr.io/its-the-vibe/app:latest compose.yml compose.override.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) > 3 {
				return fmt.Errorf("accepts between 1 and 3 arg(s), received %d", len(args))
			}

			imageName := args[0]
			inputFile := defaultComposePath
			if len(args) >= 2 {
				inputFile = args[1]
			}
			outputFile := defaultOverridePath
			if len(args) == 3 {
				outputFile = args[2]
			}

			if err := createOverrideFile(imageName, inputFile, outputFile); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Successfully generated %q with image: %s\n", outputFile, imageName)
			return nil
		},
	}
}

func newViewCommand() *cobra.Command {
	var useBase bool
	var useOverride bool
	var useCurrent bool

	cmd := &cobra.Command{
		Use:   "view",
		Short: "View the image tag of the first service",
		Long:  "View the image tag of the first service from the base, override, or current compose configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("accepts 0 arg(s), received %d", len(args))
			}

			selected := 0
			if useBase {
				selected++
			}
			if useOverride {
				selected++
			}
			if useCurrent {
				selected++
			}
			if selected > 1 {
				return errors.New("flags --base, --override, and --current are mutually exclusive")
			}

			inputFile := defaultComposePath
			if useBase {
				inputFile = defaultComposePath
			} else if useOverride {
				inputFile = defaultOverridePath
			} else if useCurrent || selected == 0 {
				if _, err := os.Stat(defaultOverridePath); err == nil {
					inputFile = defaultOverridePath
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}

			imageName, err := readFirstServiceImage(inputFile)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), imageName)
			return nil
		},
	}

	cmd.Flags().BoolVar(&useBase, "base", false, "Read from docker-compose.yml")
	cmd.Flags().BoolVar(&useOverride, "override", false, "Read from docker-compose.override.yml")
	cmd.Flags().BoolVar(&useCurrent, "current", false, "Read from override if present, otherwise base")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [docker-compose.override.yml]",
		Short: "Delete a Docker Compose override file",
		Long:  "Delete a Docker Compose override file if it exists.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts between 0 and 1 arg(s), received %d", len(args))
			}

			outputFile := defaultOverridePath
			if len(args) == 1 {
				outputFile = args[0]
			}

			if err := os.Remove(outputFile); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					fmt.Fprintf(cmd.OutOrStdout(), "No override file found at %q\n", outputFile)
					return nil
				}
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted %q\n", outputFile)
			return nil
		},
	}
}

func createOverrideFile(imageName, inputFile, outputFile string) error {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("input file %q not found", inputFile)
		}
		return err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse %q: %w", inputFile, err)
	}

	servicesNode, err := lookupTopLevelNode(&document, "services")
	if err != nil {
		return err
	}
	if servicesNode.Kind != yaml.MappingNode {
		return fmt.Errorf("services in %q must be a mapping", inputFile)
	}

	for i := 0; i < len(servicesNode.Content); i += 2 {
		servicesNode.Content[i+1] = &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "image"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: imageName},
			},
		}
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		_ = encoder.Close()
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("failed to create %q or output file is empty", outputFile)
	}

	return nil
}

func lookupTopLevelNode(document *yaml.Node, key string) (*yaml.Node, error) {
	if len(document.Content) == 0 {
		return nil, errors.New("input YAML document is empty")
	}

	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("input YAML document must be a mapping")
	}

	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("input YAML document does not contain a %q mapping", key)
}

func readFirstServiceImage(inputFile string) (string, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("input file %q not found", inputFile)
		}
		return "", err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return "", fmt.Errorf("parse %q: %w", inputFile, err)
	}

	servicesNode, err := lookupTopLevelNode(&document, "services")
	if err != nil {
		return "", err
	}
	if servicesNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("services in %q must be a mapping", inputFile)
	}
	if len(servicesNode.Content) < 2 || len(servicesNode.Content)%2 != 0 {
		return "", fmt.Errorf("services in %q must contain at least one valid service mapping", inputFile)
	}

	firstService := servicesNode.Content[1]
	if firstService.Kind != yaml.MappingNode {
		return "", fmt.Errorf("first service in %q must be a mapping", inputFile)
	}
	if len(firstService.Content)%2 != 0 {
		return "", fmt.Errorf("first service in %q must be a valid mapping", inputFile)
	}

	for i := 0; i < len(firstService.Content); i += 2 {
		if firstService.Content[i].Value == "image" {
			return firstService.Content[i+1].Value, nil
		}
	}

	return "", fmt.Errorf("first service in %q does not contain an image field", inputFile)
}
