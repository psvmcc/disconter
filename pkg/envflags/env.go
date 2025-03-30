package envflags

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func SetFlagsFromEnvironment() (err error) {
	flag.VisitAll(func(f *flag.Flag) {
		name := strings.ToUpper(strings.ReplaceAll(f.Name, ".", "_"))
		if value, ok := os.LookupEnv(name); ok {
			err2 := flag.Set(f.Name, value)
			if err2 != nil {
				err = fmt.Errorf("failed setting flag from environment: %w", err2)
			}
		}
	})
	return err
}
