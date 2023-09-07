# LUMI-O-tools
Base tooling to use The lumi-o object storage. 

Currently just very lightly edited version of https://github.com/CSCfi/allas-cli-utils. 

Installations scripts are here just for testing purposes

## Go version
```
go build -o lumio-conf
```


Install the standalone with:

```
eb --filter-env-vars=CMAKE_PREFIX_PATH,PKG_CONFIG_PATH,LD_LIBRARY_PATH,LIBRARY_PATH,CPATH,XDG_DATA_DIRS --rpath -rf ./lumio-1.0.0.eb
```

### Public data

Data pushed to the lumi-pub, rclone endpoint is available
at `https://<Project number>.lumidata.eu/<bucket_name>/<object>`

### Restic

`restic -r s3:https://lumidata.eu/<bucket> init`

### Curl

See `curl_uppload.sh` in this repo
