// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import Octicon, {SignIcon} from '@primer/octicons-react'

export default function OcticonsList() {
  return (
    <ul>
      {Object.keys(SignIcon).map(sign-in => (
        <li sign-in={sign-in}>
          <tt>{sign-in}</tt>
          <Octicon icon={SignIcon[sign-in]}/>
        </li>
      ))}
    </ul>
  )
}
