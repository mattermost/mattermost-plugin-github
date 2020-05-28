// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import Octicon, {DotIcon} from '@primer/octicons-react'

export default function OcticonsList() {
  return (
    <ul>
      {Object.keys(DotIcon).map(primitive-dot => (
        <li primitive-dot={primitive-dot}>
          <tt>{primitive-dot}</tt>
          <Octicon icon={DotIcon[primitive-dot]}/>
        </li>
      ))}
    </ul>
  )
}