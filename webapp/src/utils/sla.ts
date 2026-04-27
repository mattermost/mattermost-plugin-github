// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const MS_PER_DAY = 24 * 60 * 60 * 1000;

// daysFromDue is negative when overdue, 0 when due today, positive when in the future.
export type ReviewSLAStatus = {
    daysFromDue: number;
    overdue: boolean;
};

/**
 * Returns the ISO timestamp the review SLA clock should start from. Prefers
 * review_sla_start (the most recent review request the plugin recorded) and
 * falls back to created_at. Returns null when neither is set.
 */
export function getReviewSLAStartIso(item: {review_sla_start?: string | null; created_at?: string | null}): string | null {
    if (item.review_sla_start) {
        return item.review_sla_start;
    }
    if (item.created_at) {
        return item.created_at;
    }
    return null;
}

/**
 * Computes the SLA status for a review item, or null when no useful answer is
 * possible (no target configured, no start date, unparsable date). The "days"
 * are calendar days computed against today's UTC date, matching the server's
 * digest math.
 */
export function getReviewSLAStatus(
    item: {review_sla_start?: string | null; created_at?: string | null},
    targetDays: number,
): ReviewSLAStatus | null {
    if (!targetDays || targetDays <= 0) {
        return null;
    }

    const startIso = getReviewSLAStartIso(item);
    if (!startIso) {
        return null;
    }

    const start = new Date(startIso);
    if (Number.isNaN(start.getTime())) {
        return null;
    }

    const dueUTC = Date.UTC(
        start.getUTCFullYear(),
        start.getUTCMonth(),
        start.getUTCDate() + targetDays,
    );
    const today = new Date();
    const todayUTC = Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate());
    const daysFromDue = Math.round((dueUTC - todayUTC) / MS_PER_DAY);

    return {
        daysFromDue,
        overdue: daysFromDue < 0,
    };
}

/**
 * True when at least one review in the list is overdue against the target.
 * Used to drive the red "needs review" indicator on the sidebar button.
 */
export function reviewsHaveOverdue(
    reviews: Array<{review_sla_start?: string | null; created_at?: string | null}> | null | undefined,
    targetDays: number,
): boolean {
    if (!targetDays || !reviews || reviews.length === 0) {
        return false;
    }
    for (const pr of reviews) {
        const status = getReviewSLAStatus(pr, targetDays);
        if (status && status.overdue) {
            return true;
        }
    }
    return false;
}
