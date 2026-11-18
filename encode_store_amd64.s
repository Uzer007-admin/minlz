//go:build amd64 && !appengine && !noasm && gc && !purego

#include "textflag.h"

// CPUID.07H.0H:EDX[15] identifies an Intel hybrid. Leaf 1AH is checked
// before it is ever queried. Unknown or hidden hybrid types use runtime copy.
// https://www.intel.com/content/www/us/en/developer/articles/guide/12th-gen-intel-core-processor-gamedev-guide.html
TEXT ·cpuidStoredCopyMode(SB), NOSPLIT, $0-1
 XORL AX, AX
 CPUID
 MOVL AX, R8
 CMPL AX, $7
 JB conventional
 MOVL $7, AX
 XORL CX, CX
 CPUID
 BTL $15, DX
 JCC conventional
 CMPL R8, $0x1a
 JB unknown
 MOVB $2, ret+0(FP)
 RET
conventional:
 MOVB $1, ret+0(FP)
 RET
unknown:
 MOVB $0, ret+0(FP)
 RET

// Called only when the maximum CPUID leaf and hybrid bit permit leaf 1AH.
TEXT ·cpuidStoreCoreType(SB), NOSPLIT, $0-1
 MOVL $0x1a, AX
 XORL CX, CX
 CPUID
 SHRL $24, AX
 MOVB AL, ret+0(FP)
 RET
