// Copyright (c) 2018-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, type Locator} from '@playwright/test';

// Sub-pixel rounding differs between fonts and zoom levels.
const tolerance = 1;

async function box(locator: Locator) {
    const b = await locator.boundingBox();
    if (!b) {
        throw new Error(`${locator} has no bounding box; is it visible?`);
    }
    return b;
}

export async function expectInside(child: Locator, parent: Locator) {
    const [c, p] = await Promise.all([box(child), box(parent)]);
    expect({
        left: c.x >= p.x - tolerance,
        top: c.y >= p.y - tolerance,
        right: c.x + c.width <= p.x + p.width + tolerance,
        bottom: c.y + c.height <= p.y + p.height + tolerance,
    }, `${child} should sit inside ${parent}`).toEqual({left: true, top: true, right: true, bottom: true});
}

export async function expectNoOverlap(a: Locator, b: Locator) {
    const [x, y] = await Promise.all([box(a), box(b)]);
    const overlaps = x.x + tolerance < y.x + y.width && y.x + tolerance < x.x + x.width &&
        x.y + tolerance < y.y + y.height && y.y + tolerance < x.y + x.height;
    expect(overlaps, `${a} and ${b} should not overlap`).toBe(false);
}

export async function expectNothingClipped(root: Locator) {
    const clipped = await root.evaluate((el, tol) => {
        const offenders: string[] = [];
        for (const node of [el, ...Array.from(el.querySelectorAll('*'))]) {
            const style = getComputedStyle(node);
            const clips = style.overflowX !== 'visible' || style.overflowY !== 'visible' || style.textOverflow === 'ellipsis';
            if (clips && (node.scrollWidth > node.clientWidth + tol || node.scrollHeight > node.clientHeight + tol)) {
                offenders.push(`<${node.tagName.toLowerCase()} class="${node.className}"> ${node.scrollWidth}x${node.scrollHeight} in ${node.clientWidth}x${node.clientHeight}`);
            }
        }
        return offenders;
    }, tolerance);
    expect(clipped, `content under ${root} should not be clipped`).toEqual([]);
}
