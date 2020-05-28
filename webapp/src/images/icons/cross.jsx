// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import Octicon, {CrossIcon} from '@primer/octicons-react'

export default function OcticonsList() {
  return (
    <ul>
      {Object.keys(CrossIcon).map(x => (
        <li x={x}>
          <tt>{x}</tt>
          <Octicon icon={CrossIcon[x]}/>
        </li>
      ))}
    </ul>
  )
}
