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
