/*
Copyright 2022 k0s authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package singlenode

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/k0sproject/k0s/inttest/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SingleNodeSuite struct {
	common.FootlooseSuite
}

func (s *SingleNodeSuite) TestK0sGetsUp() {
	s.NoError(s.InitController(0, "--single"))

	kc, err := s.KubeClient(s.ControllerNode(0))
	s.Require().NoError(err)

	err = s.WaitForNodeReady(s.ControllerNode(0), kc)
	s.NoError(err)

	pods, err := kc.CoreV1().Pods("kube-system").List(s.Context(), v1.ListOptions{
		Limit: 100,
	})
	s.NoError(err)

	podCount := len(pods.Items)

	s.T().Logf("found %d pods in kube-system", podCount)
	s.Greater(podCount, 0, "expecting to see few pods in kube-system namespace")

	s.T().Log("waiting to see CNI pods ready")
	s.NoError(common.WaitForKubeRouterReadyWithContext(s.Context(), kc), "CNI did not start")

	s.T().Run("verify", func(t *testing.T) {
		ssh, err := s.SSH(s.ControllerNode(0))
		require.NoError(t, err, "failed to SSH into controller")
		defer ssh.Disconnect()

		t.Run(("kineIsDefaultStorage"), func(t *testing.T) {
			_, err = ssh.ExecWithOutput("test -e /var/lib/k0s/bin/kine && ps xa | grep kine")
			assert.NoError(t, err)
		})

		t.Run(("noControllerJoinTokens"), func(t *testing.T) {
			noToken, err := ssh.ExecWithOutput(fmt.Sprintf("'%s' token create --role=controller", s.K0sFullPath))
			assert.Error(t, err)
			assert.Equal(t, "Error: refusing to create token: cannot join into a single node cluster", noToken)
		})

		t.Run(("noWorkerJoinTokens"), func(t *testing.T) {
			noToken, err := ssh.ExecWithOutput(fmt.Sprintf("'%s' token create --role=worker", s.K0sFullPath))
			assert.Error(t, err)
			assert.Equal(t, "Error: refusing to create token: cannot join into a single node cluster", noToken)
		})
	})
}

func TestSingleNodeSuite(t *testing.T) {
	s := SingleNodeSuite{
		common.FootlooseSuite{
			ControllerCount: 1,
			ControllerUmask: 027,
		},
	}
	suite.Run(t, &s)
}
