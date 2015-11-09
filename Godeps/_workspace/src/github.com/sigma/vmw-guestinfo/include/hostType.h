/*********************************************************
 * Copyright (C) 1998-2015 VMware, Inc. All rights reserved.
 *
 * This program is free software; you can redistribute it and/or modify it
 * under the terms of the GNU Lesser General Public License as published
 * by the Free Software Foundation version 2.1 and no later version.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
 * or FITNESS FOR A PARTICULAR PURPOSE.  See the Lesser GNU General Public
 * License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin St, Fifth Floor, Boston, MA  02110-1301 USA.
 *
 *********************************************************/

/*
 * hostType.h --
 *
 *      Interface to host-specific information functions
 *   
 */

#ifndef _HOSTTYPE_H_
#define _HOSTTYPE_H_

#include "vm_basic_types.h"

Bool HostType_OSIsVMK(void);
Bool HostType_OSIsPureVMK(void);
Bool HostType_OSIsVMK64(void);
Bool HostType_OSIsSimulator(void);

#endif /* ifndef _HOSTTYPE_H_ */
