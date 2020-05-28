// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import Octicon, {ChangesRequestedIcon} from '@primer/octicons-react'

export default function OcticonsList() {
  return (
    <ul>
      {Object.keys(ChangesRequestedIcon).map(request-changes => (
        <li request-changes={request-changes}>
          <tt>{request-changes}</tt>
          <Octicon icon={ChangesRequestedIcon[request-changes]}/>
        </li>
      ))}
    </ul>
  )
}