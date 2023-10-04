/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_BASIC_TYPES_H_
#define INCLUDE_LIBYUV_BASIC_TYPES_H_

#include <stddef.h>  // For size_t and NULL

#if !defined(INT_TYPES_DEFINED) && !defined(GG_LONGLONG)
#define INT_TYPES_DEFINED

#include <stdint.h>  // for uintptr_t and C99 types

#endif  // INT_TYPES_DEFINED

#if !defined(LIBYUV_API)
#define LIBYUV_API
#endif  // LIBYUV_API

#define LIBYUV_BOOL int

#endif  // INCLUDE_LIBYUV_BASIC_TYPES_H_
