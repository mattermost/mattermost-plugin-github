// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const MS_PER_DAY = 24 * 60 * 60 * 1000;

export type ReviewTargetDayType = 'calendar' | 'business';

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

export function normalizeReviewTargetDayType(dayType?: string | null): ReviewTargetDayType {
    if (typeof dayType === 'string' && dayType.trim().toLowerCase() === 'business') {
        return 'business';
    }
    return 'calendar';
}

/** Advance a UTC Y/M/D by n weekdays (Mon–Fri). Returns UTC midnight ms of the due day. */
export function addBusinessDaysUTC(year: number, month: number, date: number, n: number): number {
    let y = year;
    let m = month;
    let d = date;
    let remaining = n;
    while (remaining > 0) {
        const next = new Date(Date.UTC(y, m, d + 1));
        y = next.getUTCFullYear();
        m = next.getUTCMonth();
        d = next.getUTCDate();
        const wd = next.getUTCDay(); // 0=Sun … 6=Sat
        if (wd !== 0 && wd !== 6) {
            remaining -= 1;
        }
    }
    return Date.UTC(y, m, d);
}

/**
 * Computes the SLA status for a review item, or null when no useful answer is
 * possible (no target configured, no start date, unparsable date). Due date
 * uses calendar or business days per dayType; daysFromDue is always calendar
 * days against today's UTC date, matching the server's digest math.
 */
export function getReviewSLAStatus(
    item: {review_sla_start?: string | null; created_at?: string | null},
    targetDays: number,
    dayType: ReviewTargetDayType = 'calendar',
    now: Date = new Date(),
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

    const y = start.getUTCFullYear();
    const m = start.getUTCMonth();
    const d = start.getUTCDate();
    const dueUTC = normalizeReviewTargetDayType(dayType) === 'business' ?
        addBusinessDaysUTC(y, m, d, targetDays) :
        Date.UTC(y, m, d + targetDays);

    const todayUTC = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
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
    dayType: ReviewTargetDayType = 'calendar',
): boolean {
    if (!targetDays || !reviews || reviews.length === 0) {
        return false;
    }
    for (const pr of reviews) {
        const status = getReviewSLAStatus(pr, targetDays, dayType);
        if (status && status.overdue) {
            return true;
        }
    }
    return false;
}
