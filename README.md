# Improve security in your CI/CD workflows

In the development world, continuous integration is where members of a team integrate all their work frequently, for example, think of a team all working on the same code base, they are fixing bugs, implementing new features, so to prevent conflict with the code, they basically merge all the code together.

The outcome is great, with multiple people collaborating at the same time there is the possibility of integration errors and the need to detect them as quickly as possible increases quality, so a good practice is to integrate testing to make sure it all works together.

When we perform this integration process, it is usually in a “CI Server” or a “build server” and in github they call it a “runner”. Jenkins, Circle CI, Travis CI (and others) are examples of integration tools and are considered also CI servers, they also can be in the cloud or on-prem, whatever works better, fulfills the code integration idea.

As a requirement, you will require a free singularity container service account, one click and you are done, 11gb for your images and 500 build time minutes. For more information, please see part 1 of the Singularity Container Workflow. Upcoming releases will include a SBOM (Software Bill of Materials) so let's take advantage of it.

## Use case

Improve code quality by bringing a code scanner and build a SIF image, for this, the following requirements are:
* Github public repository.
* A free singularity container service account.

We will create a simple Hello World program in golang. Remember, this is a demonstration on how to implement a singularity image and perform a SBOM scan.

The structure of our source code is as follows:
```
$ tree
.
├── go.mod         <-- Standard module definition **no deps here**
├── helloworld.def <-- SIF image definition file
├── main           <-- Compiled program
├── main.go        <-- Source code of the program
└── README.md      <-- This file
```

Our program is very basic, contents of the **main.go** file:

```go
package main

import (
	"fmt"
	"hello"
)

func main() {
	message := hello.SayHello("World")
	fmt.Println(message)
}
```
First, we will create a definition file for this image, this is the content of the **helloworld.def** file:

```
Bootstrap: library
From: alpine:latest

%files
    /app/main /usr/bin
%runscript
    /usr/bin/main
```

If you are new, let's explain:

* **Bootstrap**. is telling the image builder where to get the core image, options are, library, dockerhub and others. At this time we are going to download the alpine image from the singularity library.
* **From**. this is the name of our image, we are going to use latest alpine image, wether this is not a good for production, the scope of this post is a demonstration.
* **%files**. This section is going to copy our helloworld program which is named: main.
* **%runscript**. This tells the singularity engine to run our program when called via the `run` subcommand.

Secondly, here comes the CI/CD part let’s create a github action, I will explain it section by section.

In this section, lets name this workflow "Build Image" is triggering on a push to the repository.
```yaml
name: Build Image

on:
  push:
    branches: [ "master" ]
```

Next section are variable definition, in this part, the definition file is set, as well as the name of the SIF file to be created. As an option, we can also keep a copy of the image file on the build server.

```yaml
env:
  DEF_FILE: "helloworld.def"
  OUTPUT_SIF: "helloworld.sif"
  LIBRARY_PATH: "library:josueneo/howto/helloworld:latest"
```

These are the steps the runnner executes when a push occurs.

1. Setup the standard "checkout" step.
2. Setup go, for building our program.
3. Actually build the program
4. Test the program, not implemented here, but a good practice.
5. Prepare the environment, here we set some variables in order for the `scs-build` program to work, create temporary directories, download  and install `siftool` to generate an SBOM and `grype` to analyze it.
6. Build the actual SIF file **remotely** and create an image, this is important to understand, this is where you need a Singularity Container Services account already mentioned before. So, create a secret token and call it `SYLABS_AUTH_TOKEN`, the value is given by the Singularity Container Services when you create the account there.
7. When the build is complete, in order to generate the SBOM, the SIF must be moved outside the container and in a temporary directory.
8. Extract the SBOM with the already installed `siftool`.
9. Perform the actual scan of the image using `grype`.
10. Save the generated SIF file in the Github's artifact list.

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
#1
    - uses: actions/checkout@v3
#2
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.18
#3
    - name: Build program
      run: go build -v ./...
#4
    - name: Test program
      run: go test -v ./...
#5
    - name: Prepare environment
      run: |
        echo "USER_ID=$(id -u ${USER}):$(id -g ${USER})" >> $GITHUB_ENV
        mkdir -p ${{ github.workspace }}/tmp-sif-path
        mkdir -p ${{ github.workspace }}/sif/
        wget -P ${{ github.workspace }}/sif/ https://github.com/sylabs/sif/releases/download/v2.8.0/sif_2.8.0_linux_amd64.tar.gz
        tar -zxf ${{ github.workspace }}/sif/sif_2.8.0_linux_amd64.tar.gz -C ${{ github.workspace }}/sif/
#6
    - name: Build SIF File
      env:
        SYLABS_AUTH_TOKEN: ${{ secrets.SYLABS_AUTH_TOKEN }}
      uses: addnab/docker-run-action@v3
      with:
        image: sylabsio/scs-build:latest
        options: -v ${{ github.workspace }}:/app -e SYLABS_AUTH_TOKEN -u ${{ env.USER_ID }}
        run: |
          /scs-build build /app/helloworld.def /app/${{ env.OUTPUT_SIF }}
#7
    - name: Move SIF
      run: mv ${{ github.workspace }}/$OUTPUT_SIF ${{ github.workspace }}/tmp-sif-path/$OUTPUT_SIF
#8
    - name: Extract SBOM
      run: |
        ${{ github.workspace }}/sif/siftool dump 5 ${{ github.workspace }}/tmp-sif-path/$OUTPUT_SIF > ${{ github.workspace }}/tmp-sif-path/sbom.json
#9
    - name: Scan SBOM
      uses: anchore/scan-action@v3
      with:
        sbom: "${{ github.workspace }}/tmp-sif-path/sbom.json"
#10
    - name: Save image to github
      uses: actions/upload-artifact@v2
      with:
        name: ${{ env.OUTPUT_SIF }}
        path: ${{ github.workspace }}/tmp-sif-path
```

By default, if any vulnerability at medium or higher the build is going to fail. To modify the behavior, set the `severity-cutoff` to `low`, `high` or `critical`:

```yaml
    - name: Scan SBOM
      uses: anchore/scan-action@v3
      with:
        sbom: "${{ github.workspace }}/tmp-sif-path/sbom.json"
        severity-cutoff: critical
```

You could inspect the output of the report, for example in SARIF, like so:

```yaml
    - name: Inspect the report
      run: cat ${{ steps.scan.outputs.sarif }}
```

## Summary
Scanning for vulnerabilities with the help of Singularity's SBOM and `grype` in a CI/CD workflow is a good start. Using many open source tools provides value, saves time and efforts, but every day a new bug comes and is not always easy to see how critical they are.