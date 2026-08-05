// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {addBusinessDaysUTC, getReviewSLAStatus, normalizeReviewTargetDayType} from './sla';

describe('normalizeReviewTargetDayType', () => {
    it('defaults to calendar', () => {
        expect(normalizeReviewTargetDayType()).toBe('calendar');
        expect(normalizeReviewTargetDayType('')).toBe('calendar');
        expect(normalizeReviewTargetDayType('other')).toBe('calendar');
    });

    it('accepts business case-insensitively', () => {
        expect(normalizeReviewTargetDayType('business')).toBe('business');
        expect(normalizeReviewTargetDayType('Business')).toBe('business');
    });
});

describe('addBusinessDaysUTC', () => {
    it('skips weekends from Friday', () => {
        // 2025-03-14 is Friday
        expect(addBusinessDaysUTC(2025, 2, 14, 1)).toBe(Date.UTC(2025, 2, 17)); // Mon
        expect(addBusinessDaysUTC(2025, 2, 14, 2)).toBe(Date.UTC(2025, 2, 18)); // Tue
        expect(addBusinessDaysUTC(2025, 2, 14, 5)).toBe(Date.UTC(2025, 2, 21)); // next Fri
    });

    it('starts from weekend and lands on Monday for +1', () => {
        expect(addBusinessDaysUTC(2025, 2, 15, 1)).toBe(Date.UTC(2025, 2, 17)); // Sat + 1
        expect(addBusinessDaysUTC(2025, 2, 16, 1)).toBe(Date.UTC(2025, 2, 17)); // Sun + 1
    });
});

describe('getReviewSLAStatus business days', () => {
    const friItem = {created_at: '2025-03-14T18:00:00.000Z'};

    it('Friday + 2 business days is still due later on Monday', () => {
        const status = getReviewSLAStatus(friItem, 2, 'business', new Date('2025-03-17T12:00:00.000Z'));
        expect(status).toEqual({daysFromDue: 1, overdue: false});
    });

    it('Friday + 2 calendar days is overdue on Monday', () => {
        const status = getReviewSLAStatus(friItem, 2, 'calendar', new Date('2025-03-17T12:00:00.000Z'));
        expect(status).toEqual({daysFromDue: -1, overdue: true});
    });
});
