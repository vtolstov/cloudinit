/*********************************************************
 * Copyright (C) 2006-2015 VMware, Inc. All rights reserved.
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
 * vmcheck.c --
 *
 *    Utility functions for discovering our virtualization status.
 */

#include <stdlib.h>
#include <string.h>

#ifdef WINNT_DDK
#   include <ntddk.h>
#endif

#include "vmware.h"
/* 
 * #include "vm_version.h"
 * #include "vm_tools_version.h"
 */
#define VMX_TYPE_UNSET 0
#define VERSION_MAGIC 0x6

#if !defined(WINNT_DDK)
#  include "hostinfo.h"
/* #  include "str.h" */
#  define Str_Strcmp(s1, s2) strcmp(s1, s2)
#endif

/*
 * backdoor.h includes some files which redefine constants in ntddk.h.  Ignore
 * warnings about these redefinitions for WIN32 platform.
 */
#ifdef WINNT_DDK
#pragma warning (push)
// Warning: Conditional expression is constant.
#pragma warning( disable:4127 )
#endif

#include "backdoor.h"

#ifdef WINNT_DDK
#pragma warning (pop)
#endif

#include "backdoor_def.h"
#include "debug.h"


typedef Bool (*SafeCheckFn)(void);

#if !defined(_WIN32)
#   include "vmsignal.h"
#   include "setjmp.h"

static sigjmp_buf jmpBuf;
static Bool       jmpIsSet;


/*
 *----------------------------------------------------------------------
 *
 * VmCheckSegvHandler --
 *
 *    Signal handler for segv. Return to the program state saved
 *    by a previous call to sigsetjmp, or Panic if sigsetjmp hasn't
 *    been called yet. This function never returns;
 *
 * Return Value:
 *    None.
 *
 * Side effects:
 *    See the manpage for sigsetjmp for details.
 *
 *----------------------------------------------------------------------
 */

static void
VmCheckSegvHandler(int clientData) // UNUSED
{
   if (jmpIsSet) {
      siglongjmp(jmpBuf, 1);
   } else {
      Panic("Received SEGV, exiting.");
   }
}
#endif


/*
 *----------------------------------------------------------------------
 *
 * VmCheckSafe --
 *
 *      Calls a potentially unsafe function, trapping possible exceptions.
 *
 * Results:
 *
 *      Return value of the passed function, or FALSE in case of exception.
 *
 * Side effects:
 *
 *      Temporarily suppresses signals / SEH exceptions
 *
 *----------------------------------------------------------------------
 */

static Bool
VmCheckSafe(SafeCheckFn checkFn)
{
   Bool result = FALSE;

   /*
    * On a real host this call should cause a GP and we catch
    * that and set result to FALSE.
    */

#if defined(_WIN32)
   __try {
      result = checkFn();
   } __except(EXCEPTION_EXECUTE_HANDLER) {
      /* no op */
   }
#else
   do {
      int signals[] = {
         SIGILL,
         SIGSEGV,
      };
      struct sigaction olds[ARRAYSIZE(signals)];

      if (Signal_SetGroupHandler(signals, olds, ARRAYSIZE(signals),
                                 VmCheckSegvHandler) == 0) {
         Warning("%s: Failed to set signal handlers.\n", __FUNCTION__);
         break;
      }

      if (sigsetjmp(jmpBuf, TRUE) == 0) {
         jmpIsSet = TRUE;
         result = checkFn();
      } else {
         jmpIsSet = FALSE;
      }

      if (Signal_ResetGroupHandler(signals, olds, ARRAYSIZE(signals)) == 0) {
         Warning("%s: Failed to reset signal handlers.\n", __FUNCTION__);
      }
   } while (0);
#endif

   return result;
}


/*
 *----------------------------------------------------------------------
 *
 * VmCheck_GetVersion --
 *
 *    Retrieve the version of VMware that's running on the
 *    other side of the backdoor.
 *
 * Return value:
 *    TRUE on success
 *       *version contains the VMX version
 *       *type contains the VMX type
 *    FALSE on failure
 *
 * Side effects:
 *    None
 *
 *----------------------------------------------------------------------
 */

Bool
VmCheck_GetVersion(uint32 *version, // OUT
                    uint32 *type)    // OUT
{
   Backdoor_proto bp;

   ASSERT(version);
   ASSERT(type);

   /* Make sure EBX does not contain BDOOR_MAGIC */
   bp.in.size = ~BDOOR_MAGIC;
   /* Make sure ECX does not contain any known VMX type */
   bp.in.cx.halfs.high = 0xFFFF;

   bp.in.cx.halfs.low = BDOOR_CMD_GETVERSION;
   Backdoor(&bp);
   if (bp.out.ax.word == 0xFFFFFFFF) {
      /*
       * No backdoor device there. This code is not executing in a VMware
       * virtual machine. --hpreg
       */
      return FALSE;
   }

   if (bp.out.bx.word != BDOOR_MAGIC) {
      return FALSE;
   }

   *version = bp.out.ax.word;

   /*
    * Old VMXs (workstation and express) didn't set their type. In that case,
    * our special pattern will still be there. --hpreg
    */

   /*
    * Need to expand this out since the toolchain's gcc doesn't like mixing
    * integral types and enums in the same trinary operator.
    */
   if (bp.in.cx.halfs.high == 0xFFFF)
      *type = VMX_TYPE_UNSET;
   else
      *type = bp.out.cx.word;

   return TRUE;
}


/*
 *----------------------------------------------------------------------
 *
 * VmCheck_IsVirtualWorld --
 *
 *    Verify that we're running in a VM & we're version compatible with our
 *    environment.
 *
 * Return value:
 *    TRUE if we're in a virtual machine, FALSE otherwise.
 *
 * Side effects:
 *    None.
 *
 *----------------------------------------------------------------------
 */

Bool
VmCheck_IsVirtualWorld(void)
{
   uint32 version;
   uint32 dummy;

#if !defined(WINNT_DDK)
   if (VmCheckSafe(Hostinfo_TouchXen)) {
      Debug("%s: detected Xen.\n", __FUNCTION__);
      return FALSE;
   }

   if (VmCheckSafe(Hostinfo_TouchVirtualPC)) {
      Debug("%s: detected Virtual PC.\n", __FUNCTION__);
      return FALSE;
   }

   if (!VmCheckSafe(Hostinfo_TouchBackDoor)) {
      Debug("%s: backdoor not detected.\n", __FUNCTION__);
      return FALSE;
   }

   /* It should be safe to use the backdoor without a crash handler now. */
   VmCheck_GetVersion(&version, &dummy);
#else
   /*
    * The Win32 vmwvaudio driver uses this function, so keep the old,
    * VMware-only check.
    */
   __try {
      VmCheck_GetVersion(&version, &dummy);
   } __except (GetExceptionCode() == STATUS_PRIVILEGED_INSTRUCTION) {
      return FALSE;
   }
#endif

   if (version != VERSION_MAGIC) {
      Debug("The version of this program is incompatible with your platform.\n");
      return FALSE;
   }

   return TRUE;
}

