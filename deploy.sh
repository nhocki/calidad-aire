#!/usr/bin/env bash

aws s3 cp static/index.html s3://www.airemedellin.com/index.html --acl public-read
aws s3 cp static/app.js s3://www.airemedellin.com/app.js --acl public-read
aws s3 cp static/style.css s3://www.airemedellin.com/style.css --acl public-read
