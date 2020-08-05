package setup

import (
	"context"
	"fmt"
)

// common installs components required to run the operator.
func logs(ctx context.Context, config Config) error {
	if config.LogAnalyticsConfig.SharedKey == "" || config.LogAnalyticsConfig.WorkspaceID == "" {
		config.Logger.LogCtx(ctx, "level", "debug", "message", "Log shipper not configured.")
		return nil
	}
	values := fmt.Sprintf(`fluentd:
  azure:
    logAnalytics:
      enabled: true
      workspaceId: "%s"
      sharedKey: "%s"
`, config.LogAnalyticsConfig.WorkspaceID, config.LogAnalyticsConfig.SharedKey)

	return installLatestReleaseChartPackage(ctx, config, "fluent-logshipping-app", values, "https://giantswarm.github.io/giantswarm-playground-catalog")
}
