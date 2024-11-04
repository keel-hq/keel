#!/usr/bin/powershell -Command

##################################################
# Main build script, this is used both for local
# development and in continuous integration
# to build and push the images
##################################################

param (
    [switch]$Push = $false,
    [switch]$RunTests = $false,
    [switch]$StartContainers = $false,
    [switch]$StartDebugContainers = $false
)

$global:ErrorActionPreference = 'Stop'

. .\helpers.ps1

$TESTDIR = $Env:TESTDIR;
if ([string]::IsNullOrWhiteSpace($TESTDIR)) {
    $TESTDIR = Get-Location;
}

# If there is a local environment envsettings
# file, load it. In pipelines, these are all comming
# from environment variables.
if (Test-Path "envsettings.ps1") {
  .\envsettings.ps1;
}

# Ensure we are in LINUX containers
if (-not(Test-Path $Env:ProgramFiles\Docker\Docker\DockerCli.exe)) {
  Get-Command docker
  Write-Warning "Docker cli not found at $Env:ProgramFiles\Docker\Docker\DockerCli.exe"
}
else {
  Write-Warning "Switching to Linux Engine"
  & $Env:ProgramFiles\Docker\Docker\DockerCli.exe -SwitchLinuxEngine
}

$Env:SERVICEACCOUNT = Join-Path (split-path -parent $MyInvocation.MyCommand.Definition) "serviceaccount"

# Define the array of environment variable names to check
$envVarsToCheck = @(
    "IMAGE_VERSION",
    "REGISTRY_PATH"
)

foreach ($envVarName in $envVarsToCheck) {
    $envVarValue = [System.Environment]::GetEnvironmentVariable($envVarName)
    if ([string]::IsNullOrWhiteSpace($envVarValue)) {
        throw "Environment variable '$envVarName' is empty or not set. Rename envsettings.ps1.template to envsettings.ps1 and complete the environment variables or set them for the current environment."
    }
}

$version = $ENV:IMAGE_VERSION;
$containerregistry = $ENV:REGISTRY_PATH;

Write-Host "Environment IMAGE_VERSION: $($version)"
Write-Host "Environment REGISTRY_PATH: $($containerregistry)"

if (-not $containerregistry.EndsWith('/')) {
    # Add a slash to the end of $containerregistry
    $containerregistry = "$containerregistry/"
}

# Image names
$Env:IMG_KEEL = "$($containerregistry)keel:$($version)";
$Env:IMG_KEEL_DEBUG = "$($containerregistry)keel-debug:$($version)";
$Env:IMG_KEEL_TESTS = "$($containerregistry)keel-tests:$($version)";

docker compose build
ThrowIfError

if ($StartContainers) {
    docker compose up
    ThrowIfError
}

if ($StartDebugContainers) {
    docker compose -f compose.debug.yml up
    ThrowIfError
}

if ($RunTests -eq $true) {
    Write-Host "Running tests..."
    docker compose -f compose.tests.yml up -d --build
    ThrowIfError
    $testResultsFile = "/go/src/github.com/keel-hq/keel/test_results.json"
    $localResultsPath = Join-Path $TESTDIR "test_results.xml"
    $containerName = "keel_tests"
    docker exec $containerName sh -c "make test"
    # This one is to export the results
    docker exec $containerName sh -c "go test -v `$(go list ./... | grep -v /tests) -cover 2>&1 | go-junit-report > $testResultsFile"
    docker cp "$($containerName):$($testResultsFile)" $localResultsPath
    Write-Host "Test results copied to $localResultsPath"
    docker compose -f compose.tests.yml down
}

if ($Push) {
    if ($Env:REGISTRY_USER -and $Env:REGISTRY_PWD) {
        Write-Output "Container registry credentials through environment provided."

        # Identify the registry
        $registryHost = $ENV:REGISTRY_SERVER;
        Write-Output "Remote registry host: $($registryHost)";
        docker login "$($registryHost)" -u="$($Env:REGISTRY_USER)" -p="$($Env:REGISTRY_PWD)"
        ThrowIfError
    }

    docker push "$($Env:IMG_KEEL)"
}

