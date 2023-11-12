# httpupload
Simple HTTP server for quick uploads in different programming languages

Note: you should run it in trusted environments. No authentication and sanitization of inputs on purpose.

# Quick Usage

Start web server:
```
python3 httpupload.py
php -S 0.0.0.0:8000 -f -t httupload.php
go run httpupload.go
```

You can use following commands to upload file (or just use browser):
```
curl -F 'file=@file.txt' http://localhost:8000/
curl -X PUT --upload-file file.txt http://localhost:8000
wget -O- --method=PUT --body-file=file.txt http://localhost:8000/file.txt
```

For obvious reasons, you have to use following for PHP files:
```
curl -F 'file[]=@file.txt' http://localhost:8000/
```

# Reference

## Python

```
optional arguments:
  -h, --help            show this help message and exit
  --port PORT, -p PORT  Listening port for HTTP Server
  --directory DIRECTORY, -d DIRECTORY
                        Specify alternative directory [default:current directory]
```

## Go

```
 -cert string
    	Specify [cert].crt and [cert].key to use for TLS
  -dir string
    	Specify the upload directory (default ".")
  -limit int
    	Specify maximum (in MB) for parsing multiform post data (default -1)
  -listen string
    	Listen on address:port (default "0.0.0.0:8000")
  -overwrite
    	Allow overwriting existing files
  -q	Be quiet
  -tls
    	Listen on TLS
```



