package main

import (
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	tokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
)

func main() {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "my test program",
	}
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}

	// some change here again
}

// GenerateBootstrapToken will generate a node join token for kubeadm.
// ttl defines the time to live for this token.
func GenerateBootstrapToken(client kubernetes.Interface, ttl time.Duration) (string, error) {
	token, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", errors.Wrap(err, "generate kubeadm token")
	}

	bts, err := bootstraptokenv1.NewBootstrapTokenString(token)
	if err != nil {
		return "", errors.Wrap(err, "new kubeadm token string")
	}

	duration := &metav1.Duration{Duration: ttl}

	if err := tokenphase.UpdateOrCreateTokens(client, false, []bootstraptokenv1.BootstrapToken{
		{
			Token:  bts,
			TTL:    duration,
			Usages: []string{"authentication", "signing"},
			Groups: []string{kubeadmconstants.NodeBootstrapTokenAuthGroup},
		},
	}); err != nil {
		return "", errors.Wrap(err, "create kubeadm token")
	}

	return token, nil
}
