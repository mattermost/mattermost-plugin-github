// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

// ReEncryptUserDataAuditParams holds request audit data for the reEncryptUserData transaction.
type ReEncryptUserDataAuditParams struct {
	TotalUsers int `json:"total_users"`
}

func (p ReEncryptUserDataAuditParams) Auditable() map[string]any {
	return map[string]any{
		"total_users": p.TotalUsers,
	}
}

// ReEncryptUserDataAuditResult holds the outcome of the reEncryptUserData transaction.
type ReEncryptUserDataAuditResult struct {
	Migrated          int `json:"migrated"`
	ForceDisconnected int `json:"force_disconnected"`
}

func (p ReEncryptUserDataAuditResult) Auditable() map[string]any {
	return map[string]any{
		"migrated":           p.Migrated,
		"force_disconnected": p.ForceDisconnected,
	}
}
