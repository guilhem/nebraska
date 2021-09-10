/*
Package updater aims to simplify omaha-powered updates.

Its goal is to abstract many of the omaha-protocol details, so users can
perform updates without having to understand the omaha protocal
internals.

Since the omaha protocol is very powerful, it supports many options that
are beyond the scope of this package. So the updater package assumes that
there's only one single application involved in the update, and that the
update is being performed for one single instance of that application.

It also simplifies the update information that is represented by an omaha
response, but allows to retrieve the actual OmahaResponse object if needed.

Basics:

Each update has four main parts involved:
  1. The application: represents the software getting updated;
  2. The instance: represents the instance of the application that's getting
     updated.
  3. The instance version: this is used by the server to decide how to respond
     (whether there's a new version available or not).
  4. The channel: an application may have different channels of releases (this
	 is typical "beta", "stable", etc.).

The way omaha managed updates work is that omaha responds to update checks, and
relies on the information given by the application's instance to keep track of
each update's state. So the basic workflow for using this updater package is:
  1. Check for an update, if there's one available then. if there is not, try again later.
  2. Inform the server we're started it (this is done by sending a progress
	 report of "download started").
  3. Actually perform the download or whatever represents fetching the
     update's parts.
  4. Inform the server that the download is finished it and that we're applying
     the update (this is done by sending a progress report of "installation
	 started").
  5. Apply the update (this deeply depends on what each application does, but
	 may involve extracting files into locations, migrating configuration,
	 etc.).
  6. Inform the server that the update installation is finished; run the new
    version of the application and report that the update is now complete
	(these are two different progress reports and may involve running).

Note that if your update requires a restart, then there's a specific progress
report for that.
The caller is also responsible for keeping any local state the update
implementation needs (like e.g. knowing that a restart has happened, or that the
version if now running).

Initialization:

An instance of the updater needs to be initialized.

	import (
		"context"
		"fmt"

		"github.com/kinvolk/nebraska/updater"
	)

	func getInstanceID() string {
		// Your own implementation here...
		return os.Getenv("MACHINE_NAME")
	}

	func getAppVersion() string {
		// Your own implementation here...
		return os.Getenv("MACHINE_VERSION")
	}

	func main(){
		conf := updater.Config{
			OmahaURL:        "http://test.omahaserver.com/v1/update/",
			AppID:           "application_id",
			Channel:         "stable",
			InstanceID:      getInstanceID(),
			InstanceVersion: getAppVersion(),
		}

		appUpdater, err := updater.New(conf)
		if err != nil {
			fmt.Println("error setting up updater",err)
			os.Exit(1)
		}

Performing updates manually:

After we have the updater instance, we can try updating:

		ctx := context.TODO()

		updateInfo, err := appUpdater.CheckForUpdates(ctx)
		if err != nil {
			fmt.Printf("oops, something didn't go well... %v\n", err)
			return
		}

		if !updateInfo.HasUpdate() {
			fmt.Printf("no updates, try next time...")
			return
		}

		// So we got an update, let's report we'll start downloading it.
		if err := appUpdater.ReportProgress(ctx, updater.ProgressDownloadStarted); err != nil {
			if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
				fmt.Println("reporting progress error:", progressErr)
			}
			fmt.Println("error reporting progress download started:",err)
			return
		}

		// This should be implemented by the user.
		// download, err := someFunctionThatDownloadsAFile(ctx, info.GetURL())
		// if err != nil {
		// 	// Oops something went wrong
		// 	if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
		// 		fmt.Println("reporting error:", progressErr)
		// 	}
		// 	return err
		// }

		// The download was successful, let's inform that to the omaha server
		if err := appUpdater.ReportProgress(ctx, updater.ProgressDownloadFinished); err != nil {
			if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
				fmt.Println("reporting progress error:", progressErr)
			}
		return err
		}

		// let's install the update!
		if err := appUpdater.ReportProgress(ctx, updater.ProgressInstallationStarted); err != nil {
			if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
				fmt.Println("reporting progress error:", progressErr)
			}
			return err
		}

		// This should be users own implementation
		// err := someFunctionThatExtractsTheUpdateAndInstallIt(ctx, download)
		// if err != nil {
		// 	// Oops something went wrong
		// 	if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
		// 		fmt.Println("reporting error:", progressErr)
		// 	}
		// 	return err
		// }

		if err := appUpdater.CompleteUpdate(ctx, updateInfo); err != nil {
			if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
				fmt.Println("reporting progress error:", progressErr)
			}
			return err
		}

		return nil
	}

If instead of rerunning the application in the example above, we'd perform a
restart, then upon running the logic again and detecting that we're running a
new version, we could report that we did so:
    u.ReportProgress(ctx, updater.ProgressUpdateCompleteAndRestarted)


Performing updates, simplified:

It may be that our update process is very straightforward (doesn't need a
restart not a lot of state checks in between) and that it can be well divided
in two parts: getting the update, applying the update.

For this use-case, updater offers a simpler way to update that sends the
progress reports automatically: TryUpdate

	// After initializing our Updater instance...

	import (
		"context"
		"fmt"

		"github.com/kinvolk/nebraska/updater"
	)

	func getInstanceID() string {
		// Your own implementation here...
		return os.Getenv("MACHINE_NAME")
	}

	func getAppVersion() string {
		// Your own implementation here...
		return os.Getenv("MACHINE_VERSION")
	}

	type exampleUpdateHandler struct {
	}

	func (e exampleUpdateHandler) FetchUpdate(ctx context.Context, info updater.UpdateInfo) error {
		// download, err := someFunctionThatDownloadsAFile(ctx, info.GetURL())
		// if err != nil {
		// 	return err
		// }
		return nil
	}

	func (e exampleUpdateHandler) ApplyUpdate(ctx context.Context, info updater.UpdateInfo) error {
		// err := someFunctionThatExtractsTheUpdateAndInstallIt(ctx, getDownloadFile(ctx))
		// if err != nil {
		// 	// Oops something went wrong
		// 	return err
		// }

		// err := someFunctionThatExitsAndRerunsTheApp(ctx)
		// if err != nil {
		// 	// Oops something went wrong
		// 	return err
		// }
		return nil
	}

	func main() {

		conf := updater.Config{
			OmahaURL:        "http://test.omahaserver.com/v1/update/",
			AppID:           "application_id",
			Channel:         "stable",
			InstanceID:      getInstanceID(),
			InstanceVersion: getAppVersion(),
		}

		appUpdater, err := updater.New(conf)
		if err != nil {
			return err
		}

		ctx := context.TODO()

		if err := appUpdater.TryUpdate(ctx, exampleUpdateHandler{}); err != nil {
			return err
		}

		return nil
	}

	// If the update succeeded, then u.GetInstanceVersion() should be set to the new version

*/
package updater
