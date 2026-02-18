#!/bin/bash

# stop the service

if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

ssh $USER@$IP "systemctl --user stop flight-bot-slack"

echo "building for linux..."
GOOS=linux GOARCH=amd64 go build -o flight-bot-slack .
echo "copying to server..."
scp flight-bot-slack $USER@$IP:/home/$USER/flight-bot-slack/

# if --assets or --data is passed, copy those folders as well
if [[ "$@" == *"--assets"* ]]; then
    echo "copying assets/ to server..."
    # zip assets/ and copy the zip file, then unzip on the server to avoid copying unnecessary files
    zip -r assets.zip assets/
    scp assets.zip $USER@$IP:/home/$USER/flight-bot-slack/
    ssh $USER@$IP "unzip -o /home/$USER/flight-bot-slack/assets.zip -d /home/$USER/flight-bot-slack/ && rm /home/$USER/flight-bot-slack/assets.zip"
    rm assets.zip
fi
if [[ "$@" == *"--data"* ]]; then
    echo "copying data/ to server..."
    # zip data/ and copy the zip file, then unzip on the server to avoid copying unnecessary files
    zip -r data.zip data/
    scp data.zip $USER@$IP:/home/$USER/flight-bot-slack/
    ssh $USER@$IP "unzip -o /home/$USER/flight-bot-slack/data.zip -d /home/$USER/flight-bot-slack/ && rm /home/$USER/flight-bot-slack/data.zip"
    rm data.zip
fi
echo "deploying on server..."
ssh $USER@$IP "systemctl --user restart flight-bot-slack"
echo "done!"
echo "tailing logs..."
echo "--------------------------------"
ssh $USER@$IP "journalctl -u flight-bot-slack --user -f"
