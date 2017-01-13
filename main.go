package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	
	"github.com/kardianos/osext"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-tools/go-xamarin/constants"
)
// ConfigsModel ...
type ConfigsModel struct {
	XamarinSolution string
	NugetUrl    string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		XamarinSolution: os.Getenv("xamarin_solution"),
		NugetUrl:    os.Getenv("nuget_url"),
	}
}

func (configs ConfigsModel) print() {
	log.Info("Configs:")

	log.Detail("- XamarinSolution: %s", configs.XamarinSolution)
	log.Detail("- NugetUrl: %s", configs.NugetUrl)
}

func (configs ConfigsModel) validate() error {
	if configs.XamarinSolution == "" {
		return errors.New("no XamarinSolution parameter specified")
	}
	if exist, err := pathutil.IsPathExists(configs.XamarinSolution); err != nil {
		return fmt.Errorf("failed to check if XamarinSolution exist at: %s, error: %s", configs.XamarinSolution, err)
	} else if !exist {
		return fmt.Errorf("xamarinSolution not exist at: %s", configs.XamarinSolution)
	}

	return nil
}

// DownloadFile ...
func DownloadFile(downloadURL, targetPath string) error {
	outFile, err := os.Create(targetPath)
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Warn("Failed to close (%s)", targetPath)
		}
	}()
	if err != nil {
		return fmt.Errorf("failed to create (%s), error: %s", targetPath, err)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download from (%s), error: %s", downloadURL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warn("failed to close (%s) body", downloadURL)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non success status code: %d", resp.StatusCode)
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download from (%s), error: %s", downloadURL, err)
	}

	return nil
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

		log.Info("Get path 1 ...")
	folderPath, err := osext.ExecutableFolder()
    if err != nil {
        fmt.Println(err)
    }
   log.Info(folderPath)
	
		log.Info("Get path 2 ...")
	pwd, err := os.Getwd()
    if err != nil {
        fmt.Println(err)
    }
    log.Info(pwd)
	
		log.Info("Get path 3 ...")
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
            fmt.Println(err)
    }
    log.Info(dir)
	
	
	nugetPth := "NuGet4.exe"
	
	nugelLocalPath := filepath.Join(folderPath, "NuGet4.exe")
	nugetRestoreCmdArgs := []string{constants.MonoPath, nugelLocalPath}
    fmt.Println(nugelLocalPath)
    fmt.Println(nugetRestoreCmdArgs)


	if configs.NugetUrl == "latest" {
		fmt.Println()
		log.Info("Updating Nuget to latest version...")
		// "sudo $nuget update -self"
		cmdArgs := []string{"sudo", nugetPth, "update", "-self"}
		cmd, err := cmdex.NewCommandFromSlice(cmdArgs)
		if err != nil {
			log.Error("Failed to create command from args (%v), error: %s", cmdArgs, err)
			os.Exit(1)
		}

		cmd.SetStdout(os.Stdout)
		cmd.SetStderr(os.Stderr)

		log.Done("$ %s", cmdex.PrintableCommandArgs(false, cmdArgs))

		if err := cmd.Run(); err != nil {
			log.Error("Failed to update nuget, error: %s", err)
			os.Exit(1)
		}
	} else if configs.NugetUrl != "" {
		fmt.Println()
		log.Info("Downloading Nuget from %s ...", configs.NugetUrl)
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__nuget__")
		if err != nil {
			log.Error("Failed to create tmp dir, error: %s", err)
			os.Exit(1)
		}

		downloadPth := filepath.Join(tmpDir, "nuget.exe")

		// https://dist.nuget.org/win-x86-commandline/v3.3.0/nuget.exe
		nugetURL := configs.NugetUrl

		log.Detail("Download URL: %s", nugetURL)

		if err := DownloadFile(nugetURL, downloadPth); err != nil {
			log.Warn("Download failed, error: %s", err)

			// https://dist.nuget.org/win-x86-commandline/v3.4.4/NuGet.exe
			nugetURL = configs.NugetUrl

			log.Detail("Retry download URl: %s", nugetURL)

			if err := DownloadFile(nugetURL, downloadPth); err != nil {
				log.Error("Failed to download nuget, error: %s", err)
				os.Exit(1)
			}
		}

		nugetRestoreCmdArgs = []string{constants.MonoPath, downloadPth}
	}

	fmt.Println()
	log.Info("Restoring Nuget packages...")

	nugetRestoreCmdArgs = append(nugetRestoreCmdArgs, "restore", configs.XamarinSolution)

	if err := retry.Times(1).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warn("Attempt %d failed, retrying...", attempt)
		}

		log.Done("$ %s", cmdex.PrintableCommandArgs(false, nugetRestoreCmdArgs))

		cmd, err := cmdex.NewCommandFromSlice(nugetRestoreCmdArgs)
		if err != nil {
			log.Error("Failed to create Nuget command, error: %s", err)
			os.Exit(1)
		}

		cmd.SetStdout(os.Stdout)
		cmd.SetStderr(os.Stderr)

		if err := cmd.Run(); err != nil {
			log.Error("Restore failed, error: %s", err)
			return err
		}
		return nil
	}); err != nil {
		log.Error("Nuget restore failed, error: %s", err)
		os.Exit(1)
	}
}
