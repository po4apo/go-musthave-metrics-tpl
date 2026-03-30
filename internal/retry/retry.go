package retry

import (
	"fmt"
	"time"
)

var defaultIntervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do выполняет fn и повторяет попытку при retriable-ошибках.
//
// :param fn: функция, которую нужно выполнить.
// :param isRetriable: предикат, определяющий, стоит ли повторять попытку при данной ошибке.
// :returns: последнюю ошибку или nil при успехе.
func Do(fn func() error, isRetriable func(error) bool) error {
	return DoWithIntervals(fn, isRetriable, defaultIntervals)
}

// DoWithIntervals выполняет fn с пользовательскими интервалами между повторами.
//
// :param fn: функция, которую нужно выполнить.
// :param isRetriable: предикат, определяющий, стоит ли повторять попытку при данной ошибке.
// :param intervals: интервалы между повторными попытками.
// :returns: последнюю ошибку или nil при успехе.
func DoWithIntervals(fn func() error, isRetriable func(error) bool, intervals []time.Duration) error {
	err := fn()
	if err == nil {
		return nil
	}

	for _, interval := range intervals {
		if !isRetriable(err) {
			return err
		}
		time.Sleep(interval)
		err = fn()
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("after %d retries: %w", len(intervals), err)
}
