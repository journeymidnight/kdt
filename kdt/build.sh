echo "BUIDING for win"
GOOS=windows go build
tar zcvf kdt-windows.tar.gz kdt.exe

echo "BUIDING for mac"
GOOS=darwin go build
tar zcvf kdt-macos.tar.gz kdt


echo "BUIDING for linux"
GOOS=linux go build
tar zcvf kdt-linux.tar.gz kdt
