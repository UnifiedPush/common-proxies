#!/bin/bash

#from environment variables
#FCM_DEVICE_TOKEN=""
#GOTIFY_DEVICE_TOKEN=""

TEST_PROXY_ADDR="127.0.0.1:5000"

PAYLOAD="abc { < > } \"\" ''

\r \r\n \n


\\"

curl -d "$PAYLOAD" "http://$TEST_PROXY_ADDR/FCM?token=$FCM_DEVICE_TOKEN"
printf "\n\n"
curl -d "$PAYLOAD" "http://$TEST_PROXY_ADDR/UP?token=$GOTIFY_DEVICE_TOKEN"
echo 
