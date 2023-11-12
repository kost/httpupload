<?php
# Simple multiple files uploader in single PHP file by Kost

$dirupload=".";
$uploaded = [];
$usefilename = True; # Dangerous if true
$checkext = False; # Check for extension of file in allowed?
$allowed = ['mp4', 'png'];

if ($_SERVER['REQUEST_METHOD']==="POST") {
header('Content-Type: application/json');

$succeeded = [];
$failed = [];

if(!empty($_FILES['file'])){
	foreach($_FILES['file']['name'] as $key => $name){
		if($_FILES['file']['error'][$key] === 0){
			$temp = $_FILES['file']['tmp_name'][$key];

			$ext = explode('.', $name);
			$ext = strtolower(end($ext));

			$file=$name;
			if (!$usefilename) {
				$file = md5_file($temp) . time() . '.' . $ext;
			}
			if ($checkext === True) {
				if(in_array($ext, $allowed) === False) {
					$failed[] = array(
						'name' => $name,
						'error' => 'extension'
					);
					continue;
				}
			}
			# if (move_uploaded_file($temp, "{$dirupload}/{$file}") === true) {
			if (move_uploaded_file($temp, "{$file}") === true) {
				$succeeded[] = array(
					'name' => $name,
					'file' => $file
				);
			}else{
				$failed[] = array(
					'name' => $name,
					'error' => 'move_uploaded_file'
				);
			}
		}
	}

	if(!empty($_POST['ajax'])){
		echo json_encode(array(
			'succeeded' => $succeeded,
			'failed' => $failed
		));
	}
}
} else {
?>
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
			<input type="file" id="file" name="file[]" required multiple onchange="upload_file(event)">
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
					data.append('file[]', source[i]);
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
<?php
}
?>

