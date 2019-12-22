// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function formatDate(date, useMilitaryTime = false) {
    const monthNames = [
        'Jan', 'Feb', 'Mar',
        'Apr', 'May', 'Jun', 'Jul',
        'Aug', 'Sep', 'Oct',
        'Nov', 'Dec',
    ];

    const day = date.getDate();
    const monthIndex = date.getMonth();
    let hours = date.getHours();
    let minutes = date.getMinutes();

    let ampm = '';
    if (!useMilitaryTime) {
        ampm = ' AM';
        if (hours >= 12) {
            ampm = ' PM';
        }

        hours %= 12;
        if (!hours) {
            hours = 12;
        }
    }

    if (minutes < 10) {
        minutes = '0' + minutes;
    }

    return monthNames[monthIndex] + ' ' + day + ' at ' + hours + ':' + minutes + ampm;
}

export function formatTimeSince(date) {
    const secondsSince = Math.trunc((Date.now() - (new Date(date)).getTime())/1000);
    if (secondsSince < 60) {
        return secondsSince + 's';
    }
    const minutesSince = Math.trunc(secondsSince/60);
    if (minutesSince < 60) {
        return minutesSince + 'm';
    }
    const hoursSince = Math.trunc(minutesSince/60);
    if (hoursSince < 24) {
        return hoursSince + 'h';
    }
    const daysSince = Math.trunc(hoursSince/24);
    return daysSince + 'd';
}