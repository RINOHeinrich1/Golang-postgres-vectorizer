#!/bin/bash
go build main.go
mv main postgres_vectorizer; scp  ./postgres_vectorizer  rino@onirtech.com:/home/rino/