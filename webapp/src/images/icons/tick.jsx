// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import Octicon, {TickIcon} from '@primer/octicons-react'

export default function OcticonsList() {
  return (
    <ul>
      {Object.keys(TickIcon).map(check => (
        <li check={check}>
          <tt>{check}</tt>
          <Octicon icon={TickIcon[check]}/>
        </li>
      ))}
    </ul>
  )
}
