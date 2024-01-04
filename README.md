# LUMI-O-tools

Command line tool to configure s3 authentication for rclone,s3cmd awscli and boto3
By deafult only s3cmd and rclone are configured 

## Installation

run
`make`

This will build the program and create a binary `lumio-conf`
in the root folder. Copy this anywhere to deploy.


## Usage

## Environment variables



- `LUMIO_SKIP_PROJID_CHECK` Set to any value to disable sanity check on the project number.
By default the program verifies that the first three digits are correct and there is a corret number of digits (9) 
- `TMPDIR` is used, if not set `/tmp/<username>/` is used. The temporary
directory is used to store configs before validation and commiting them. 
- `LUMIO_PROJECTID` Can be used to supply the projectid when using the `--noninteractive` flag. If used in conjunction with `--project-number`. The command line flag value will be used.
- `LUMIO_S3_ACCESS` Used to supply the S3 access key when using the `--noninteractive` flag.
- `LUMIO_S3_SECRET` Used to supply the S3 secret key when using the `--noninteractive` flag.




### Public data

Data pushed to public rclone endpoints is available
at `https://<Lumi project number>.lumidata.eu/<bucket_name>/<object>`

## Additional examples

`lumio-conf` does not create any configuration for these tools,
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


