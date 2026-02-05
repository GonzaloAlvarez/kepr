/*
Copyright © 2025 Gonzalo Alvarez

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package workflow

import (
	"context"
	"fmt"
)

func ExecuteWithRetry(ctx context.Context, cfg StepConfig) error {
	if cfg.Retry == nil || cfg.Retry.MaxAttempts <= 1 {
		return cfg.Execute(ctx)
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.Retry.MaxAttempts; attempt++ {
		lastErr = cfg.Execute(ctx)
		if lastErr == nil {
			return nil
		}

		if attempt >= cfg.Retry.MaxAttempts {
			break
		}

		if cfg.Retry.PromptRetry != nil {
			retry, promptErr := cfg.Retry.PromptRetry(lastErr, attempt)
			if promptErr != nil {
				return promptErr
			}
			if !retry {
				return lastErr
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", cfg.Retry.MaxAttempts, lastErr)
}
