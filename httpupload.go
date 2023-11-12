//usr/bin/go run $0 $@ ; exit
// httpupload in Go. Copyright (C) Kost. Distributed under MIT.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"io/ioutil"
	"log"
	"strings"

	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	indexhtml []byte
	success   []map[string]string
	failed    []map[string]string
)

type AppOptions struct {
	allowOverwrite bool
	cert           string
	usetls         bool
	quiet          bool
	uploadDir      string
	limitMultiPart int64
	limitMultiArg  int
	listenstr      string
}

var CurOptions AppOptions

// RandString generates random string of n size
// It returns the generated random string.and any write error encountered.
func RandString(n int) string {
	r := make([]byte, n)
	_, err := rand.Read(r)
	if err != nil {
		return ""
	}

	b := make([]byte, n)
	l := len(letters)
	for i := range b {
		b[i] = letters[int(r[i])%l]
	}
	return string(b)
}

// RandBytes generates random bytes of n size
// It returns the generated random bytes
func RandBytes(n int) []byte {
	r := make([]byte, n)
	_, err := rand.Read(r)
	if err != nil {
	}

	return r
}

// RandBigInt generates random big integer with max number
// It returns the generated random big integer
func RandBigInt(max *big.Int) *big.Int {
	r, _ := rand.Int(rand.Reader, max)
	return r
}

func GenPair(keysize int) (cacert []byte, cakey []byte, cert []byte, certkey []byte) {

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	ca := &x509.Certificate{
		SerialNumber: RandBigInt(serialNumberLimit),
		Subject: pkix.Name{
			Country:            []string{RandString(16)},
			Organization:       []string{RandString(16)},
			OrganizationalUnit: []string{RandString(16)},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		SubjectKeyId:          RandBytes(5),
		BasicConstraintsValid: true,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, _ := rsa.GenerateKey(rand.Reader, keysize)
	pub := &priv.PublicKey
	caBin, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	if err != nil {
		log.Println("create ca failed", err)
		return
	}

	cert2 := &x509.Certificate{
		SerialNumber: RandBigInt(serialNumberLimit),
		Subject: pkix.Name{
			Country:            []string{RandString(16)},
			Organization:       []string{RandString(16)},
			OrganizationalUnit: []string{RandString(16)},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: RandBytes(6),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	priv2, _ := rsa.GenerateKey(rand.Reader, keysize)
	pub2 := &priv2.PublicKey
	cert2Bin, err2 := x509.CreateCertificate(rand.Reader, cert2, ca, pub2, priv)
	if err2 != nil {
		log.Println("create cert2 failed", err2)
		return
	}

	privBin := x509.MarshalPKCS1PrivateKey(priv)
	priv2Bin := x509.MarshalPKCS1PrivateKey(priv2)

	return caBin, privBin, cert2Bin, priv2Bin

}

func VerifyCert(cacert []byte, cert []byte) bool {
	caBin, _ := x509.ParseCertificate(cacert)
	cert2Bin, _ := x509.ParseCertificate(cert)
	err3 := cert2Bin.CheckSignatureFrom(caBin)
	if err3 != nil {
		return false
	}
	return true
}

func GetPEMs(cert []byte, key []byte) (pemcert []byte, pemkey []byte) {
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: key,
	})

	return certPem, keyPem
}

func GetTLSPair(certPem []byte, keyPem []byte) (tls.Certificate, error) {
	tlspair, errt := tls.X509KeyPair(certPem, keyPem)
	if errt != nil {
		return tlspair, errt
	}
	return tlspair, nil
}

func GetRandomTLS(keysize int) (tls.Certificate, error) {
	_, _, cert, certkey := GenPair(keysize)
	certPem, keyPem := GetPEMs(cert, certkey)
	tlspair, err := GetTLSPair(certPem, keyPem)
	return tlspair, err
}

func init() {
	indexhtml = []byte(`
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
					xmlhttp.setRequestHeader("X-Requested-With", "XMLHttpRequest");
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
	`)
	flag.StringVar(&CurOptions.cert, "cert", "", "Specify [cert].crt and [cert].key to use for TLS")
	flag.StringVar(&CurOptions.listenstr, "listen", "0.0.0.0:8000", "Listen on address:port")
	flag.BoolVar(&CurOptions.allowOverwrite, "overwrite", false, "Allow overwriting existing files")
	flag.BoolVar(&CurOptions.usetls, "tls", false, "Listen on TLS")
	flag.BoolVar(&CurOptions.quiet, "q", false, "Be quiet")
	flag.StringVar(&CurOptions.uploadDir, "dir", ".", "Specify the upload directory")
	flag.IntVar(&CurOptions.limitMultiArg, "limit", -1, "Specify maximum (in MB) for parsing multiform post data")
	flag.Parse()

	if CurOptions.quiet {
		log.SetOutput(ioutil.Discard)
	}

	if CurOptions.limitMultiArg != -1 {
		CurOptions.limitMultiPart = int64(CurOptions.limitMultiArg) << 20 // in MB
	} else {
		CurOptions.limitMultiPart = -1 // default: no limit
	}

	http.HandleFunc("/", handleRequest)
}

func main() {
	var err error
	var cer tls.Certificate

	log.Printf("Server is listening on %s\n", CurOptions.listenstr)
	srv := &http.Server{
		Addr: CurOptions.listenstr,
	}

	if CurOptions.usetls {
		if CurOptions.cert == "" {
			log.Printf("No TLS certificate. Generated random one.")
			cer, err = GetRandomTLS(2048)
		} else {
			cer, err = tls.LoadX509KeyPair(CurOptions.cert+".crt", CurOptions.cert+".key")
		}
		if err != nil {
			log.Printf("Error creating/loading certificate file %s: %v", CurOptions.cert, err)
		}
		srv = &http.Server{
			Addr:      CurOptions.listenstr,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{cer}},
		}
		err = srv.ListenAndServeTLS("", "")
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Printf("Error: %s", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got request %s: %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodPut:
		handlePut(w, r)
	case http.MethodGet:
		handleGet(w, r)
	case http.MethodPost:
		handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Join(CurOptions.uploadDir, filepath.Base(r.URL.Path))

	if !CurOptions.allowOverwrite {
		if _, err := os.Stat(filename); err == nil {
			http.Error(w, fmt.Sprintf(`"%s" already exists`, filename), http.StatusConflict)
			return
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating file: %s", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error writing to file: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `Saved "%s"`, filename)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(indexhtml)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	success = []map[string]string{}
	failed = []map[string]string{}

	contentType := r.Header.Get("Content-Type")
	xRequestedWith := r.Header.Get("X-Requested-With")
	var retContentType string
	var length int

	log.Printf("Got content %s for %s: %s", contentType, r.Method, r.URL.Path)

	switch {
	case contentType == "application/json" || xRequestedWith == "XMLHttpRequest":
		retContentType = "application/json"
		err := parseMultipartForm(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error handling multipart form: %s", err), http.StatusInternalServerError)
			return
		}
		retJSON := map[string]interface{}{"succeeded": success, "failed": failed}
		jsonStr, err := json.Marshal(retJSON)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error encoding JSON: %s", err), http.StatusInternalServerError)
			return
		}
		length = len(jsonStr)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", retContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
		w.Write(jsonStr)
	case strings.HasPrefix(contentType, "multipart/form-data"):
		err := parseMultipartForm(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error handling multipart form: %s", err), http.StatusInternalServerError)
			return
		}
		retContentType = "text/html"
		length = len(indexhtml)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", retContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
		w.Write(indexhtml)
	default:
		length = len(indexhtml)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", retContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
		w.Write(indexhtml)
	}
}

func parseMultipartForm(r *http.Request) error {
	err := r.ParseMultipartForm(CurOptions.limitMultiPart)
	if err != nil {
		return err
	}

	form := r.MultipartForm
	files := form.File["file"]

	for _, file := range files {
		dstFile, err := os.Create(filepath.Join(CurOptions.uploadDir, file.Filename))
		if err != nil {
			failed = append(failed, map[string]string{"name": file.Filename, "error": err.Error()})
			continue
		}
		defer dstFile.Close()

		srcFile, err := file.Open()
		if err != nil {
			failed = append(failed, map[string]string{"name": file.Filename, "error": err.Error()})
			continue
		}
		defer srcFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			failed = append(failed, map[string]string{"name": file.Filename, "error": err.Error()})
			continue
		}

		success = append(success, map[string]string{"name": file.Filename, "file": file.Filename})
	}

	return nil
}
