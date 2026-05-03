package capabilities

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// capNetAdmin is bit 12 in the capability set.
const capNetAdmin = 12

// HasCapNetAdmin reports whether the current process has CAP_NET_ADMIN.
// On non-Linux platforms it always returns false.
func HasCapNetAdmin() bool {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "CapEff:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return false
		}
		capEff, err := strconv.ParseUint(fields[1], 16, 64)
		if err != nil {
			return false
		}
		return capEff&(1<<capNetAdmin) != 0
	}
	return false
}
