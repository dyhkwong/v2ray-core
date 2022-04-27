package singbridge

import (
	"github.com/sagernet/sing/common/exceptions"
)

func ReturnError(err error) error {
	if exceptions.IsClosed(err) {
		return nil
	}
	return err
}
