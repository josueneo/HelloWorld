# Improve security in your CI/CD workflows

In the development world, continuous integration is where members of a team integrate all their work frequently, for example, think of a team all working on the same code base, they are fixing bugs, implementing new features, so to prevent conflicts, all the code is merged together with automation.

For this use case, a free singularity container service account is required, one click and you are done, 11gb for your images and 500 build time minutes. For more information, please see [part 1 of the Singularity Container Workflow](https://sylabs.io/2022/08/singularity-container-workflow-part-1-introduction-to-singularity-container-services/). Upcoming releases will include a SBOM (Software Bill of Materials), this is an important contribution to security.

If you are not familiar with SBOM, let me put it in simple words, it is a list of dependencies: modules, libraries and components required to either run or build a piece of software. Like the ingredients you read on a can of food, if you are allergic to some specific ingredient and you may want to stay away, the same happens with an SBOM, you will probably want to stay away from a flagged or vulnerable component. Some questions may arise, How is this collection built? Does it have a specific format? What kind and type of information contains?

There are many SBOM specifications, the most widely used today are [SPDX](https://spdx.dev) and [CycloneDX](https://cyclonedx.org/), being CycloneDX the format selected by default. And now, what can we do with an SBOM?, we can audit the components.

The tool of choice is [grype](https://github.com/anchore/grype), it is a vulnerability scanner for container images and filesystems, is very easy to use, it got a extensive vulnerability database for many major linux operating systems like: CentOS, Debian, Oracle Linux, Red Hat (RHEL), Ubuntu, and others, and also supports SIF images out of the box.

## Use case

Build a SIF image and Improve security by documenting and analyzing packages and its dependencies, for this, the following requirements are:
* Github public repository.
* A free singularity container service account.

We will create a simple Hello World program in golang. Remember, this is a demonstration on how to build a singularity image, generate a Software Bill of Materials (SBOM), and check to see if any of the software in our image has known vulnerabilities..

The structure of our source code is as follows:

```bash
$ tree
.
├── cmd
│   └── main.go      <-- Command line program, imports hello package.
├── go.mod           <-- Standard module definition **no deps here**
├── hello.go         <-- Source code of the program
├── hello_test.go    <-- Testing source code of the program
├── helloworld.def   <-- SIF image definition file
├── main             <-- Compiled program
└── README.md        <-- This file
```

Our program is very basic, contents of the **cmd/main.go** file:

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

Contents of the hello.go file:
```go
package hello
 
func SayHello(a string) string {
   return "Hello " + a + "!"
}
```

The contents of the source code for testing are basic as well, so the contents of the hello_test.go file are:
```go
package hello
 
import (
   "strings"
   "testing"
)
 
func TestLength(t *testing.T) {
   msg := SayHello("World")
   length := len(msg)
   if length != 12 {
       t.Errorf("SayHello(\"World\") length is %d; want 12", length)
   }
}
 
func TestContainsUTF(t *testing.T) {
   msg := SayHello("嗨")
   if !strings.Contains(msg, "嗨") {
       t.Error("SayHello(\"嗨\") doesn't support UTF8")
   }
}
```

Then, we will create a definition file for this image, this is the content of the helloworld.def file:
```go
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

## Deep analysis

At a very low level the SIF is made up of a global header, metadata, and the actual data. Our interest is in the metadata part, which in the SIF standard are called descriptors. The descriptors are an important part of the image because it tells how the image was created using a definition file, it stores data partitions, labels, signing data, cryptographic data, and our data type of interest: SBOM data.

With the use of siftool, we can list these descriptors for example:

```bash
$ siftool list ubuntu-nosbom.sif
--------------------------------------------------------------------------
ID   |GROUP   |LINK    |SIF POSITION (start-end)  |TYPE
--------------------------------------------------------------------------
1    |1       |NONE    |32768-32836               |Def.FILE
2    |1       |NONE    |36864-37662               |JSON.Generic
3    |1       |NONE    |40960-41052               |JSON.Generic
4    |1       |NONE    |45056-29814784            |FS (Squashfs/*System/am
5    |NONE    |1   (G) |29814784-29816581         |Signature (SHA-256)
```

However, as you can see, there is no SBOM bundled into this image, here is an example of an image with SBOM:

```bash
$ siftool list ubuntu-sbom.sif
--------------------------------------------------------------------------
ID   |GROUP   |LINK    |SIF POSITION (start-end)  |TYPE
--------------------------------------------------------------------------
1    |1       |NONE    |32768-32806               |Def.FILE
2    |1       |NONE    |36864-38769               |JSON.Generic
3    |1       |NONE    |40960-41052               |JSON.Generic
4    |1       |NONE    |45056-27750400            |FS (Squashfs/*System/am
5    |1       |NONE    |27750400-27975279         |SBOM
```

The SBOM is marked as the ID 5 in this SIF file, so, save that ID for future reference because is going to be needed for the next command, now that we know the SIF image contains an SBOM, we are going to dump it like so:

```bash
$ siftool dump 5 ubuntu-sbom.sif
{
 "bomFormat": "CycloneDX",
 "specVersion": "1.4",
 "serialNumber": "urn:uuid:e54f3e42-2f90-4563-bfd5-9b9ae6988d66",
 "version": 1,
...
}
```

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
8. Perform the actual scan of the image using `grype`.
9.  Save the generated SIF file in the Github's artifact list.

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
