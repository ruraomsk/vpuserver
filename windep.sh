#!/bin/bash
echo "Start to Semen deploy"
#GOOS=windows GOARCH=amd64 go build -o "VPUserver.exe"
go build
FILE=/home/rura/mnt/Semen/VPUserver
if [ -f "$FILE" ]; then
    echo "Mounted the server drive"
else
    echo "Mounting the server drive"
sudo mount -t cifs -o username=semen,password=1,vers=2.1  \\\\192.168.115.134\\vpu-server /home/rura/mnt/Semen
fi
sudo cp VPUserver /home/rura/mnt/Semen/
sudo cp *.toml /home/rura/mnt/Semen/
sudo cp *.json /home/rura/mnt/Semen/
sudo cp license.key /home/rura/mnt/Semen/

