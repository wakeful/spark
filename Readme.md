# spark

> **Seeking Public AWS Resources and Kernels**

> [!NOTE]
> command-line tool designed to identify public AWS resources—such as backups, AMIs, snapshots, and more—associated with
> specific AWS accounts.

```shell
$ spark -h
Usage spark:
  -list-scanners
        list available resource types
  -region value
        AWS region to scan (can be specified multiple times)
  -scan value
        AWS resource type to scan (can be specified multiple times)
  -target string
        target AWS account ID (default "self")
  -verbose
        verbose log output
  -version
        show version


$ spark -list-scanners
AMI
snapshotsEBS
snapshotsRDS
ssmDocument
```

### Installation

#### From source

```shell
# via the Go toolchain
go install github.com/wakeful/spark
```

#### Using a binary release

You can download a pre-built binary from the [release page](https://github.com/wakeful/spark/releases/latest) and add it
to your user PATH.
