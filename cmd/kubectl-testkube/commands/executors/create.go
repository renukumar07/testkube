package executors

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateExecutorCmd() *cobra.Command {
	var (
		types, command, executorArgs, imagePullSecretNames []string
		name, executorType, image, uri, jobTemplate        string
		labels                                             map[string]string
	)

	cmd := &cobra.Command{
		Use:     "executor",
		Aliases: []string{"exec", "ex"},
		Short:   "Create new Executor",
		Long:    `Create new Executor Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			ui.ExitOnError("parsing flag value", err)

			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			var client apiClient.Client
			if !crdOnly {
				client, namespace = common.GetClient(cmd)

				executor, _ := client.GetExecutor(name)
				if name == executor.Name {
					ui.Failf("Executor with name '%s' already exists in namespace %s", name, namespace)
				}
			}

			jobTemplateContent := ""
			if jobTemplate != "" {
				b, err := os.ReadFile(jobTemplate)
				ui.ExitOnError("reading job template", err)
				jobTemplateContent = string(b)
			}

			var imageSecrets []testkube.LocalObjectReference
			for _, secretName := range imagePullSecretNames {
				imageSecrets = append(imageSecrets, testkube.LocalObjectReference{Name: secretName})
			}

			options := apiClient.CreateExecutorOptions{
				Name:             name,
				Namespace:        namespace,
				Types:            types,
				ExecutorType:     executorType,
				Image:            image,
				ImagePullSecrets: imageSecrets,
				Command:          command,
				Args:             executorArgs,
				Uri:              uri,
				JobTemplate:      jobTemplateContent,
				Labels:           labels,
			}

			if !crdOnly {

				_, err = client.CreateExecutor(options)
				ui.ExitOnError("creating executor "+name+" in namespace "+namespace, err)

				ui.Success("Executor created", name)
			} else {
				if options.JobTemplate != "" {
					options.JobTemplate = fmt.Sprintf("%q", options.JobTemplate)
				}

				data, err := crd.ExecuteTemplate(crd.TemplateExecutor, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test name - mandatory")
	cmd.Flags().StringArrayVarP(&types, "types", "t", []string{}, "test types handled by executor")
	cmd.Flags().StringVar(&executorType, "executor-type", "job", "executor type, container or job (defaults to job)")

	cmd.Flags().StringVarP(&uri, "uri", "u", "", "if resource need to be loaded from URI")
	cmd.Flags().StringVar(&image, "image", "", "image used for container executor")
	cmd.Flags().StringArrayVar(&imagePullSecretNames, "image-pull-secrets", []string{}, "secret name used to pull the image in container executor")
	cmd.Flags().StringArrayVar(&command, "command", []string{}, "command passed to image in container executor")
	cmd.Flags().StringArrayVar(&executorArgs, "args", []string{}, "args passed to image in container executor")
	cmd.Flags().StringVarP(&jobTemplate, "job-template", "j", "", "if executor needs to be launched using custom job specification, then a path to template file should be provided")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
