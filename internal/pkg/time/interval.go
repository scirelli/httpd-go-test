package time

import (
	"context"
	gotime "time"
)

//DoEvery execute a function every time interval.
func DoEvery(ctx context.Context, d gotime.Duration, f func(gotime.Time)) error {
	ticker := gotime.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case x := <-ticker.C:
			f(x)
		}
	}
}
