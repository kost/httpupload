#!/usr/bin/env python
# HTTP upload server by Kost

import os

AllowOverwrite=False
UploadDir=os.getcwd()

import cgi
import argparse
import json

try:
    import http.server as server
except ImportError:
    # Handle Python 2.x
    import SimpleHTTPServer as server

html_page = '''
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8" />
	<title>Multiple File Uploader</title>
<style>
body{
	font-family: "Georgia", serif;
}

.upload{
	width: 500px;
	background: #f0f0f0;
	border: 1px solid #ddd;
	padding: 20px;
}

.upload fieldset{
	border: 0;
	padding: 0;
	margin-bottom: 10px;
}

.upload fieldset legend{
	font-size: 1.2em;
	margin-bottom: 10px;
}

.bar{
	width: 100%;
	background: #eee;
	padding: 3px;
	margin-bottom: 10px;
	box-shadow: inset 0 1px 3px rgba(0, 0, 0, .3);
	border-radius: 3px;
	box-sizing: border-box;
	-webkit-box-sizing: border-box;
	-moz-box-sizing: border-box;
}

.bar-fill{
	height: 20px;
	display: block;
	background: cornflowerblue;
	width: 0;
	border-radius: 3px;

	transition: width 0.8s ease;
	-webkit-transition: width 0.8s ease;
	-moz-transition: width 0.8s ease;
}

.bar-fill-text{
	color: #fff;
	padding: 3px;
}

.uploads a, .uploads span{
	display: block;
}
</style>
</head>
<body>
	<form method="post" enctype="multipart/form-data" id="upload" class="upload">
		<div id="drop_file_zone" ondrop="upload_file(event)" ondragover="return false">
		<div id="drag_upload_file">
			<p>Drop file(s) here</p>
			<p>or</p>
		<fieldset>
			<legend>Upload files</legend>
			<input type="file" id="file" name="file" required multiple onchange="upload_file(event)">
			<input type="submit" id="submit" name="submit" value="Upload">
		</fieldset>
		</div>
		</div>

		<div class="bar">
			<span class="bar-fill" id="pb"><span class="bar-fill-text" id="pt"></span></span>
		</div>
		<div id="uploads" class="uploads">
			Uploaded file links will appear here.
		</div>

		<script>
		var app = app || {};

		(function(o){
			"use strict";

			//Private methods
			var ajax, getFormData, setProgress;

			ajax = function(data){
				// var xmlhttp = new XMLHttpRequest();
				var xmlhttp = (window.XMLHttpRequest) ? new window.XMLHttpRequest() : new window.ActiveXObject("Microsoft.XMLHTTP");
				var uploaded;

				xmlhttp.addEventListener('readystatechange', function(){
					if(this.readyState === 4){
						if(this.status === 200){
							uploaded = JSON.parse(this.response);
							if(typeof o.options.finished === 'function'){
								o.options.finished(uploaded);
							}
						}else{
							if(typeof o.options.error === 'function'){
								o.options.error();
							}
						}
					}
				});

				xmlhttp.upload.addEventListener('progress', function(event){
					var percent;

					if(event.lengthComputable === true){
						percent = Math.round((event.loaded / event.total) * 100);
						setProgress(percent);
					}
				});

				xmlhttp.open('post', o.options.processor);
				xmlhttp.send(data);
			};

			getFormData = function(source){
				var data = new FormData(), i;

				for(i = 0; i < source.length; i = i + 1){
					data.append('file', source[i]);
				}

				data.append('ajax', true);

				return data;
			};

			setProgress = function(value){
				if(o.options.progressBar !== undefined){
					o.options.progressBar.style.width = value ? value + '%' : 0;
				}

				if(o.options.progressText !== undefined){
					o.options.progressText.innerText = value ? value + '%' : '';
				}
			};

			o.uploader = function(options){
				o.options = options;

				if(o.options.files !== undefined){
					ajax(getFormData(o.options.files.files));
				}
			}

		}(app));

		function upload_file(e) {
			e.preventDefault();

			var f = document.getElementById('file'),
				pb = document.getElementById('pb'),
				pt = document.getElementById('pt');

			app.uploader({
				files: f,
				progressBar: pb,
				progressText: pt,
				// processor: 'upload.php',
				processor: window.location,

				finished: function(data){
					var uploads = document.getElementById('uploads'),
						succeeded = document.createElement('div'),
						failed = document.createElement('div'),

						anchor,
						span,
						x;

					if(data.failed.length){
						failed.innerHTML = '<p>Unfortunately, the following files failed to upload:</p>'
					}

					uploads.innerText = '';

					for(x = 0; x < data.succeeded.length; x = x + 1){
						anchor = document.createElement('a');
						anchor.href = 'uploads/' + data.succeeded[x].file;
						anchor.innerText = data.succeeded[x].name;
						anchor.target = '_blank';

						succeeded.appendChild(anchor);
					}

					for(x = 0; x < data.failed.length; x = x + 1){
						span = document.createElement('span');
						span.innerText = data.failed[x].name;

						failed.appendChild(span);
					}

					uploads.appendChild(succeeded);
					uploads.appendChild(failed);
				},

				error: function(){
					console.log('Not working.');
				}
			});
		};
		// document.getElementById('submit').addEventListener('click', upload_file(event));
		document.getElementById('submit').addEventListener('click', function(e){
			upload_file(e)
		});

		</script>
	</form>
</body>
</html>
'''.encode('utf-8')

success=[]
failed=[]

class HTTPRequestHandler(server.SimpleHTTPRequestHandler):
    """Extend SimpleHTTPRequestHandler to handle PUT requests"""
    def do_PUT(self):
        """Save a file following a HTTP PUT request"""
        filename = UploadDir+"/"+os.path.basename(self.path)

        # Don't overwrite files
        if not AllowOverwrite and os.path.exists(filename):
            self.send_response(409, 'Conflict')
            self.end_headers()
            reply_body = '"%s" already exists\n' % filename
            self.wfile.write(reply_body.encode('utf-8'))
            return

        file_length = int(self.headers['Content-Length'])
        with open(filename, 'wb') as output_file:
            output_file.write(self.rfile.read(file_length))
        self.send_response(201, 'Created')
        self.end_headers()
        reply_body = 'Saved "%s"\n' % filename
        self.wfile.write(reply_body.encode('utf-8'))

    def do_GET(self):
        self.send_response(200, 'OK')
        self.end_headers()
        self.wfile.write(html_page)

    def do_POST(self):
        success=[]
        failed=[]
        ctype, pdict = cgi.parse_header(self.headers['Content-Type'])
        r, info = self.deal_post_data()
        print(r, info, "by: ", self.client_address)
        retctype="text/html"
        length=0
        if ctype == 'application/json':
            retctype='application/json'
            retjson={"succeeded" : success, "failed" : failed}
            jsonstr=json.dumps(retjson)
            jsonb=bytes(jsonstr, 'utf-8')
            length=len(jsonb)
        else:
            length=len(html_page)
        self.send_response(200)
        self.send_header("Content-type", retctype)
        self.send_header("Content-Length", str(length))
        self.end_headers()
        if ctype == 'application/json':
            self.wfile.write(jsonb)
        else:
            self.wfile.write(html_page)

    def deal_post_data(self):
        ctype, pdict = cgi.parse_header(self.headers['Content-Type'])
        pdict['boundary'] = bytes(pdict['boundary'], "utf-8")
        pdict['CONTENT-LENGTH'] = int(self.headers['Content-Length'])
        if ctype == 'multipart/form-data' or ctype == 'application/json':
            form = cgi.FieldStorage( fp=self.rfile, headers=self.headers, environ={'REQUEST_METHOD':'POST', 'CONTENT_TYPE':self.headers['Content-Type'], })
            print (type(form))
            try:
                if isinstance(form["file"], list):
                    for record in form["file"]:
                        try:
                            open("%s/%s"%(UploadDir,record.filename), "wb").write(record.file.read())
                            success.append({"name": record.filename, "file": record.filename})
                        except IOError:
                            failed.append({"name": record.filename, "error": "IOerror"})
                else:
                    try:
                        open("%s/%s"%(UploadDir,form["file"].filename), "wb").write(form["file"].file.read())
                    except IOError:
                        failed.append({"name": record.filename, "file": record.filename})
            except IOError:
                    failed.append({"name": "name.pdf", "error": "IOerror"})
                    return (False, "Can't create file to write, do you have permission to write?")
        return (True, "Files uploaded")

if __name__ == '__main__':
    example_text = '''Examples:
  curl -F 'file=@file.txt' http://localhost:8000/
  curl -X PUT --upload-file file.txt http://localhost:8000
  wget -O- --method=PUT --body-file=file.txt http://localhost:8000/file.txt
 '''
    parser = argparse.ArgumentParser(prog='httpupload',description='HTTP Upload Server', epilog=example_text, formatter_class=argparse.RawDescriptionHelpFormatter, )
    parser.add_argument('--port', '-p', type=int, default=8000, help='Listening port for HTTP Server')
    parser.add_argument('--directory', '-d', default=UploadDir,
        help='Specify alternative directory [default:current directory]')
    args = parser.parse_args()
    UploadDir = args.directory
    server.test(HandlerClass=HTTPRequestHandler, port=args.port)

