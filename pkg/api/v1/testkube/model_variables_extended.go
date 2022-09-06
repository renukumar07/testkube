package testkube

import (
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/secret"
)

type Variables map[string]Variable

func VariablesToMap(v Variables) map[string]string {
	vars := make(map[string]string, len(v))

	for _, v := range v {
		vars[v.Name] = v.Value
	}

	return vars
}

func ObfuscateSecrets(output string, variables Variables, testName string) string {
	// TODO: this is ugly, does anybody have a better idea?
	namespace := "testkube"
	if ns, ok := os.LookupEnv("TESTKUBE_NAMESPACE"); ok {
		namespace = ns
	}
	secretClient, err := secret.NewClient(namespace)

	for _, v := range variables {
		secretValue := ""
		if *v.Type_ == SECRET_VariableType {
			secretValue = v.Value
			if v.SecretRef != nil {
				skv := getSecretKeyValue(err, secretClient, v.SecretRef.Name)
				secretValue = skv[v.SecretRef.Key]
			}
		}

		if secretValue != "" {
			output = strings.ReplaceAll(output, `'`+v.Value+`'`, "********")
			output = strings.ReplaceAll(output, `"`+v.Value+`"`, "********")
		}
	}
	return output
}

func getSecretKeyValue(err error, secretClient *secret.Client, secretName string) map[string]string {
	var secretKeyValues map[string]string
	if err == nil {
		secretKeyValues, err = secretClient.Get(secretName)
		if err != nil && !errors.IsNotFound(err) {
			log.DefaultLogger.Warnw("error getting secret", "error", err)
		}
	}
	return secretKeyValues
}
