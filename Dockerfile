FROM golang:latest

RUN curl -sL https://deb.nodesource.com/setup_8.x | bash -
RUN apt-get update && apt-get install -y --no-install-recommends nodejs

#https://github.com/electron-userland/electron-packager/issues/654

#use 32bit wine for rcedit
RUN dpkg --add-architecture i386 && apt-get update && apt-get install -y --no-install-recommends wine32 wine
RUN npm install electron-packager -g
