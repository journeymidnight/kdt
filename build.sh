echo "BUIDING for win"
GOOS=windows go build -o bin/kdt.exe kdt/main.go
tar zcvf kdt-windows.tar.gz bin/kdt.exe

echo "BUIDING for mac"
GOOS=darwin go build -o bin/kdt-mac kdt/main.go
tar zcvf kdt-macos.tar.gz bin/kdt-mac

echo "BUIDING for linux"
GOOS=linux go build -o bin/kdt kdt/main.go
tar zcvf kdt-linux.tar.gz bin/kdt

mkdir gui/vendor && cp bin/kdt-mac bin/kdt bin/kdt.exe gui/vendor/

pushd gui
npm install
npm run package:mac
npm run package:win
popd


