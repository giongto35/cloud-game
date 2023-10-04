/*
 *  Copyright 2012 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_CONVERT_ARGB_H_
#define INCLUDE_LIBYUV_CONVERT_ARGB_H_

#include "basic_types.h"

// Conversion matrix for YVU to BGR
LIBYUV_API extern const struct YuvConstants kYvuI601Constants;   // BT.601
LIBYUV_API extern const struct YuvConstants kYvuJPEGConstants;   // BT.601 full
LIBYUV_API extern const struct YuvConstants kYvuH709Constants;   // BT.709
LIBYUV_API extern const struct YuvConstants kYvuF709Constants;   // BT.709 full
LIBYUV_API extern const struct YuvConstants kYvu2020Constants;   // BT.2020
LIBYUV_API extern const struct YuvConstants kYvuV2020Constants;  // BT.2020 full

#endif  // INCLUDE_LIBYUV_CONVERT_ARGB_H_
