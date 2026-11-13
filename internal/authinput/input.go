package authinput

import "fmt"

// Handler is installed once in an isolated worker before authentication starts.
var Handler func(kind string, image []byte) (string, error)

func Read(kind string, image []byte, out *string) (int, error) {
	if Handler != nil {
		v, err := Handler(kind, image)
		*out = v
		if err != nil {
			return 0, err
		}
		return 1, nil
	}
	return fmt.Scan(out)
}
