## build package 

# windows

Clone source code

```
git clone https://github.com/journeymidnight/kdt.git 

```


Install KDT binary to vender

```

cd kdt/gui
mkdir vender
cd vender
wget https://github.com/journeymidnight/kdt/releases/download/v1.0/kdt-win32-x64.zip
unzip kdt-win32-x64.zip

```
Install electron and other nodejs modules

```
set ELECTRON_MIRROR=https://npm.taobao.org/mirrors/electron npm install
```


Run electron packager to get the final package

```
npm run package:win
```


# mac

Clone source code

```
git clone https://github.com/journeymidnight/kdt.git 

```


Install KDT binary to vender

```
cd kdt/gui
mkdir vender
cd vender
wget https://github.com/journeymidnight/kdt/releases/download/v1.0/kdt-macos.tar.gz
tar zxvf kdt-macos.tar.gz


```
Install electron and other nodejs modules

```
export ELECTRON_MIRROR=https://npm.taobao.org/mirrors/electron npm install
```


Run electron packager to get the final package

```
npm run package:mac
```
