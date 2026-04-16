package settings

import "testing"

func TestDefaultGatewaySettingsHeartbeatDefaults(t *testing.T) {
	t.Parallel()

	defaults := DefaultGatewaySettings()
	if defaults.Runtime.DebugMode != GatewayDebugModeOff {
		t.Fatalf("expected runtime.debugMode default off, got %q", defaults.Runtime.DebugMode)
	}
	if defaults.Runtime.CallRecords.SaveStrategy != GatewayCallRecordSaveStrategyAll {
		t.Fatalf("expected runtime.callRecords.saveStrategy default all, got %q", defaults.Runtime.CallRecords.SaveStrategy)
	}
	if defaults.Runtime.CallRecords.RetentionDays != DefaultGatewayCallRecordRetentionDays {
		t.Fatalf(
			"expected runtime.callRecords.retentionDays default %d, got %d",
			DefaultGatewayCallRecordRetentionDays,
			defaults.Runtime.CallRecords.RetentionDays,
		)
	}
	if defaults.Runtime.CallRecords.AutoCleanup != GatewayCallRecordAutoCleanupHourly {
		t.Fatalf("expected runtime.callRecords.autoCleanup default hourly, got %q", defaults.Runtime.CallRecords.AutoCleanup)
	}
	if !defaults.Heartbeat.Periodic.Enabled {
		t.Fatalf("expected periodic.enabled default true")
	}
	if defaults.Heartbeat.ActiveHours.Start == "" {
		t.Fatalf("expected active start default to be set")
	}
	if defaults.Heartbeat.ActiveHours.End == "" {
		t.Fatalf("expected active end default to be set")
	}
	if defaults.Heartbeat.ActiveHours.Timezone == "" {
		t.Fatalf("expected active timezone default to be set")
	}
}

func TestResolveGatewaySettingsHeartbeatExtendedFields(t *testing.T) {
	t.Parallel()

	params := GatewaySettingsParams{
		Heartbeat: &GatewayHeartbeatSettingsParams{
			RunSession:   stringPtr(" hb-session "),
			PromptAppend: stringPtr(" append prompt "),
			Periodic: &GatewayHeartbeatPeriodicSettingsParams{
				Enabled: boolPtr(true),
				Every:   stringPtr(" 15m "),
			},
			Delivery: &GatewayHeartbeatDeliverySettingsParams{
				Periodic: &GatewayHeartbeatSurfacePolicyParams{
					Center:           boolPtr(false),
					PopupMinSeverity: stringPtr(" warning "),
					ToastMinSeverity: stringPtr(" error "),
					OSMinSeverity:    stringPtr(" critical "),
				},
				EventDriven: &GatewayHeartbeatSurfacePolicyParams{
					Center:           boolPtr(false),
					PopupMinSeverity: stringPtr(" info "),
					ToastMinSeverity: stringPtr(" warning "),
					OSMinSeverity:    stringPtr(" error "),
				},
				ThreadReplyMode: stringPtr(" inline "),
			},
			Events: &GatewayHeartbeatEventSettingsParams{
				CronWakeMode:     stringPtr(" now "),
				ExecWakeMode:     stringPtr(" next-heartbeat "),
				SubagentWakeMode: stringPtr(" now "),
			},
		},
	}

	resolved := ResolveGatewaySettings(params)

	if resolved.Heartbeat.RunSession != "hb-session" {
		t.Fatalf("expected runSession to be persisted, got %q", resolved.Heartbeat.RunSession)
	}
	if resolved.Heartbeat.PromptAppend != "append prompt" {
		t.Fatalf("expected promptAppend to be persisted, got %q", resolved.Heartbeat.PromptAppend)
	}
	if !resolved.Heartbeat.Periodic.Enabled {
		t.Fatalf("expected periodic.enabled=true")
	}
	if resolved.Heartbeat.Periodic.Every != "15m" {
		t.Fatalf("expected periodic.every=15m, got %q", resolved.Heartbeat.Periodic.Every)
	}
	if resolved.Heartbeat.Delivery.Periodic.Center {
		t.Fatalf("expected periodic delivery center=false")
	}
	if resolved.Heartbeat.Delivery.Periodic.PopupMinSeverity != "warning" {
		t.Fatalf(
			"expected periodic delivery popupMinSeverity=warning, got %q",
			resolved.Heartbeat.Delivery.Periodic.PopupMinSeverity,
		)
	}
	if resolved.Heartbeat.Delivery.Periodic.ToastMinSeverity != "error" {
		t.Fatalf(
			"expected periodic delivery toastMinSeverity=error, got %q",
			resolved.Heartbeat.Delivery.Periodic.ToastMinSeverity,
		)
	}
	if resolved.Heartbeat.Delivery.Periodic.OSMinSeverity != "critical" {
		t.Fatalf(
			"expected periodic delivery osMinSeverity=critical, got %q",
			resolved.Heartbeat.Delivery.Periodic.OSMinSeverity,
		)
	}
	if resolved.Heartbeat.Delivery.EventDriven.Center {
		t.Fatalf("expected eventDriven delivery center=false")
	}
	if resolved.Heartbeat.Delivery.EventDriven.PopupMinSeverity != "info" {
		t.Fatalf(
			"expected eventDriven delivery popupMinSeverity=info, got %q",
			resolved.Heartbeat.Delivery.EventDriven.PopupMinSeverity,
		)
	}
	if resolved.Heartbeat.Delivery.EventDriven.ToastMinSeverity != "warning" {
		t.Fatalf(
			"expected eventDriven delivery toastMinSeverity=warning, got %q",
			resolved.Heartbeat.Delivery.EventDriven.ToastMinSeverity,
		)
	}
	if resolved.Heartbeat.Delivery.EventDriven.OSMinSeverity != "error" {
		t.Fatalf(
			"expected eventDriven delivery osMinSeverity=error, got %q",
			resolved.Heartbeat.Delivery.EventDriven.OSMinSeverity,
		)
	}
	if resolved.Heartbeat.Delivery.ThreadReplyMode != "inline" {
		t.Fatalf(
			"expected threadReplyMode=inline, got %q",
			resolved.Heartbeat.Delivery.ThreadReplyMode,
		)
	}
	if resolved.Heartbeat.Events.CronWakeMode != "now" {
		t.Fatalf("expected cronWakeMode=now, got %q", resolved.Heartbeat.Events.CronWakeMode)
	}
	if resolved.Heartbeat.Events.ExecWakeMode != "next-heartbeat" {
		t.Fatalf("expected execWakeMode=next-heartbeat, got %q", resolved.Heartbeat.Events.ExecWakeMode)
	}
	if resolved.Heartbeat.Events.SubagentWakeMode != "now" {
		t.Fatalf("expected subagentWakeMode=now, got %q", resolved.Heartbeat.Events.SubagentWakeMode)
	}
}

func TestResolveGatewaySettingsDebugModeSetsRecordPrompt(t *testing.T) {
	t.Parallel()

	resolved := ResolveGatewaySettings(GatewaySettingsParams{
		Runtime: &GatewayRuntimeSettingsParams{
			DebugMode: stringPtr("full"),
		},
	})

	if resolved.Runtime.DebugMode != GatewayDebugModeFull {
		t.Fatalf("expected runtime.debugMode=full, got %q", resolved.Runtime.DebugMode)
	}
	if !resolved.Runtime.RecordPrompt {
		t.Fatalf("expected runtime.recordPrompt=true when debugMode=full")
	}
}

func TestResolveGatewaySettingsMigratesLegacyRecordPrompt(t *testing.T) {
	t.Parallel()

	resolved := ResolveGatewaySettings(GatewaySettingsParams{
		Runtime: &GatewayRuntimeSettingsParams{
			RecordPrompt: boolPtr(true),
		},
	})

	if resolved.Runtime.DebugMode != GatewayDebugModeFull {
		t.Fatalf("expected legacy recordPrompt=true to resolve debugMode=full, got %q", resolved.Runtime.DebugMode)
	}
	if !resolved.Runtime.RecordPrompt {
		t.Fatalf("expected runtime.recordPrompt=true after migration")
	}
}

func TestResolveGatewaySettingsCallRecordOptions(t *testing.T) {
	t.Parallel()

	resolved := ResolveGatewaySettings(GatewaySettingsParams{
		Runtime: &GatewayRuntimeSettingsParams{
			CallRecords: &GatewayCallRecordsSettingsParams{
				SaveStrategy:  stringPtr(" errors "),
				RetentionDays: intPtr(400),
				AutoCleanup:   stringPtr(" on_write "),
			},
		},
	})

	if resolved.Runtime.CallRecords.SaveStrategy != GatewayCallRecordSaveStrategyErrors {
		t.Fatalf(
			"expected runtime.callRecords.saveStrategy=errors, got %q",
			resolved.Runtime.CallRecords.SaveStrategy,
		)
	}
	if resolved.Runtime.CallRecords.RetentionDays != MaxGatewayCallRecordRetentionDays {
		t.Fatalf(
			"expected runtime.callRecords.retentionDays=%d, got %d",
			MaxGatewayCallRecordRetentionDays,
			resolved.Runtime.CallRecords.RetentionDays,
		)
	}
	if resolved.Runtime.CallRecords.AutoCleanup != GatewayCallRecordAutoCleanupOnWrite {
		t.Fatalf(
			"expected runtime.callRecords.autoCleanup=on_write, got %q",
			resolved.Runtime.CallRecords.AutoCleanup,
		)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
