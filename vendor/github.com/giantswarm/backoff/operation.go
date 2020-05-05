package backoff

// An Operation is executing by Retry() or RetryNotify(). The operation will be
// retried using a backoff policy if it returns an error.
type Operation func() error
