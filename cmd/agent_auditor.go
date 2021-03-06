/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, n.A.
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

package cmd

import (
	"github.com/bbva/qed/gossip"
	"github.com/bbva/qed/gossip/auditor"
	"github.com/bbva/qed/gossip/member"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/util"
	"github.com/spf13/cobra"
)

func newAgentAuditorCommand(ctx *cmdContext, config *gossip.Config) *cobra.Command {

	auditorConfig := auditor.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "auditor",
		Short: "Start a QED auditor",
		Long:  `Start a QED auditor that reacts to snapshot batches propagated by QED servers and periodically executes membership queries to verify the inclusion of events`,
		Run: func(cmd *cobra.Command, args []string) {

			log.SetLogger("QedAuditor", ctx.logLevel)

			config.Role = member.Auditor
			auditorConfig.APIKey = ctx.apiKey

			auditor, err := auditor.NewAuditor(*auditorConfig)
			if err != nil {
				log.Fatalf("Failed to start the QED monitor: %v", err)
			}

			agent, err := gossip.NewAgent(config, []gossip.Processor{auditor})
			if err != nil {
				log.Fatalf("Failed to start the QED auditor: %v", err)
			}

			contacted, err := agent.Join(config.StartJoin)
			if err != nil {
				log.Fatalf("Failed to join the cluster: %v", err)
			}
			log.Debugf("Number of nodes contacted: %d (%v)", contacted, config.StartJoin)

			defer agent.Shutdown()
			util.AwaitTermSignal(agent.Leave)
		},
	}

	cmd.Flags().StringSliceVarP(&auditorConfig.QEDUrls, "qedUrls", "", []string{}, "Comma-delimited list of QED servers ([host]:port), through which an auditor can make queries")
	cmd.Flags().StringSliceVarP(&auditorConfig.PubUrls, "pubUrls", "", []string{}, "Comma-delimited list of QED servers ([host]:port), through which an auditor can make queries")
	cmd.MarkFlagRequired("qedUrls")
	cmd.MarkFlagRequired("pubUrls")

	return cmd
}
