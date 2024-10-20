# self-update: Build self-updating Go programs

[![Coverage Status](https://coveralls.io/repos/github/Jaeminst/selfupdate/badge.svg?branch=main)](https://coveralls.io/github/Jaeminst/selfupdate?branch=main)

This repository simplifies the logic from the `fynelabs/selfupdate` repository for easier use. Security-related checks such as hash values, checksums, and signature verifications, as well as scheduler-related logic, have been removed.

All you need to do is run it with the URL of the update you want. That's it!

## Unmanaged update

Example of updating from a URL:

```go
import (
    "fmt"
    "net/http"

    "github.com/Jaeminst/selfupdate"
)

func doUpdate(url string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    err = selfupdate.Apply(resp.Body, selfupdate.Options{})
    if err != nil {
        // error handling
    }
    return err
}
```

## Managed update

```go
func main() {
	done := make(chan struct{})

	// The public key above match the signature of the below file served by our CDN
	httpSource := selfupdate.NewHTTPSource(nil, url)
	config := &selfupdate.Config{
		FetchOnStart: true,
		Source:       httpSource,

		RestartConfirmCallback: func() bool {
			// Add a custom cofirm survey here.
			// or success message
			done <- struct{}{}
			return true
		},
		UpgradeConfirmCallback: func(_ string) bool {
			// Add a custom cofirm survey here.
			return true
		},
	}

	_, err = selfupdate.Manage(config)
	if err != nil {
		fmt.Printf("Error while setting up update manager: %v", err)
		return
	}
	<-done
}

```

## License
Apache
