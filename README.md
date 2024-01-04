# LUMI-O-tools

Command line tool to configure s3 authentication for rclone,s3cmd awscli and boto3
By deafult only s3cmd and rclone are configured. 

## Installation

run
`make`

This will build the program and create a binary `lumio-conf`
in the root folder. Copy this anywhere to deploy.


## Usage

### Basic 

Run the command and supply the requested information, this will then automatically
create the configurations:

```bash
$ lumio-conf 
 Please login to  https://auth.lumidata.eu/
 In the web interface, choose first the project you wish to use.
 Next generate a new key or use existing valid key
 Open the Key details view and based on that give following information
 
 =========== PROMPTING USER INPUT ===========
 Lumi project number
 462000007
 Access key
 Secret key
 
 =========== CONFIGURING S3CMD ===========
 Updated s3cmd config /users/nortamoh/.s3cfg-lumi-462000001
 
 New configuration set as default
 Created s3cmd config lumi-462000001 for project_462000001
 	Other existing configurations can be accessed by adding the -c flag
 	s3cmd -c ~/.s3cfg-<profile-name> COMMAND ARGS
 
 =========== CONFIGURING RCLONE ===========
 Updated rclone config /users/nortamoh/.config/rclone/rclone.conf
 
 rclone remote lumi-462000001-private: now provides an S3 based connection to Lumi-O storage area of project_462000001
 
 rclone remote lumi-462000001-public: now provides an S3 based connection to Lumi-O storage area of project_462000001
 	Data pushed here is publicly available using the URL: https://462000001.lumidata.eu/<bucket_name>/<object>"
```

By default the generated config will also be set as the default one. This can be disbled with the `--keep-default=<tool1>,<tool2>`.
Generated configurations will also be validated before committing. This can be disabled with the `--skip-validation=<tool1>,<tool2>` flag.


## Environment variables



- `LUMIO_SKIP_PROJID_CHECK` Set to any value to disable sanity check on the project number.
By default the program verifies that the first three digits are correct and there is a corret number of digits (9) 
- `TMPDIR` is used, if not set `/tmp/<username>/` is used. The temporary
directory is used to store configs before validation and commiting them. 
- `LUMIO_PROJECTID` Can be used to supply the projectid when using the `--noninteractive` flag. If used in conjunction with `--project-number`. The command line flag value will be used.
- `LUMIO_S3_ACCESS` Used to supply the S3 access key when using the `--noninteractive` flag.
- `LUMIO_S3_SECRET` Used to supply the S3 secret key when using the `--noninteractive` flag.




## ToolBoostrap for testing

**rclone**

From https://github.com/rclone/rclone/releases download:
```
wget https://github.com/rclone/rclone/releases/download/v1.65.0/rclone-v1.65.0-linux-amd64.zip && \
unzip rclone-v1.65.0-linux-amd64.zip && \
mv rclone-v1.65.0-linux-amd64 rclone && \
chmod +x rclone
```

**s3cmd**
```
pip3 install s3cmd
```

**boto3**
```
pip3 install boto3
```

**Aws cli**
```
pip3 install awscli
```

**restic**
```
wget https://github.com/restic/restic/releases/download/v0.16.2/restic_0.16.2_linux_amd64.bz2 && \
bunzip restic_0.16.2_linux_amd64.bz2 &&\
mv restic_0.16.2_linux_amd64 restic &&\
chmod +x restic
```

## Public data

Data pushed to public rclone endpoints is available
at `https://<Lumi project number>.lumidata.eu/<bucket_name>/<object>`

## Additional examples

Boto3 uses the same config as the `aws` command.

### Boto3

```python
>>> import boto3
>>> s3 = boto3.resource('s3')
>>> for bucket in s3.buckets.all():
        print(bucket.name)
```


`lumio-conf` does not create any configuration for the following tools,
but simple examples are included here for completeness 

### Restic

```
$ export AWS_ACCESS_KEY_ID=<MY_ACCESS_KEY>
$ export AWS_SECRET_ACCESS_KEY=<MY_SECRET_ACCESS_KEY>
$ restic -r s3:https://lumidata.eu/<bucket> init
```

### Curl

```bash
file=README.md
bucket=my-nice.bucket
resource="/${bucket}/${file}"
contentType="application/x-compressed-tar"
dateValue=`date -R`
stringToSign="PUT\n\n${contentType}\n${dateValue}\n${resource}"
s3Key=$AWS_ACCESS_KEY_ID
s3Secret=$AWS_SECRET_ACCESS_KEY 
signature=`echo -en ${stringToSign} | openssl sha1 -hmac ${s3Secret} -binary | base64`
curl -X PUT -T "${file}" \
  -H "Host: https://lumidata.eu/" \
  -H "Date: ${dateValue}" \
  -H "Content-Type: ${contentType}" \
  -H "Authorization: AWS ${s3Key}:${signature}" \
    https://lumidata.eu/${bucket}/${file}
```


