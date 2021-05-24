#!/bin/bash
echo "Start to Vano deploy"
GOOS=windows GOARCH=amd64 go build -o "VPUserver.exe" 
FILE=/home/rura/mnt/vpu/VPUserver.exe
if [ -f "$FILE" ]; then
    echo "Mounted the server drive"
else
    echo "Mounting the server drive"
    sudo mount -t cifs -o username=Kola,password=162747 \\\\192.168.115.110\\vpu /home/rura/mnt/vpu
fi
sudo cp *.exe /home/rura/mnt/vpu/
sudo cp *.toml /home/rura/mnt/vpu/
sudo cp *.json /home/rura/mnt/vpu/

