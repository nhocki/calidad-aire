#!/usr/bin/env bash

aws s3 cp static/index.html s3://siata.picoyplaca.org/index.html --acl public-read
aws s3 cp static/app.js s3://siata.picoyplaca.org/app.js --acl public-read
aws s3 cp static/style.css s3://siata.picoyplaca.org/style.css --acl public-read
