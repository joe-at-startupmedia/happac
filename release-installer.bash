#!/usr/bin/env bash

set -x

RELEASE=$1
PROJECT_NAME="happac"
PROJECT_URL="https://github.com/joe-at-startupmedia/$PROJECT_NAME"
RELEASE_ARCHIVE="$PROJECT_NAME-$RELEASE"

if [ "$RELEASE" == "" ]; then
    echo "Please enter release version"
    exit 1
fi

clean_downloads() {
  rm -f "$RELEASE_ARCHIVE.tar.gz"
}

download_from_project() {
  SRC=$1
  DST=$2
  wget "$SRC" -O "$DST"
  wgetreturn=$?
  if [[ $wgetreturn -ne 0 ]]; then
    echo "Could not wget: $SRC"
    clean_downloads
    exit 1
  fi
}

systemd_install() {
  sudo cp -R bin/happac /usr/local/bin/
  sudo cp "systemd/happac.service" /usr/lib/systemd/system/
  sudo cp "systemd/happac.env" /etc/haproxy/
  systemctl enable happac
  systemctl start happac
  echo "Modify /etc/haproxy/happac.env variable specifc to your needs"
}


echo "Installing $PROJECT_NAME from release: $RELEASE"

#the extracted folder isnt prepended by the letter v
download_from_project "$PROJECT_URL/archive/refs/tags/v$RELEASE.tar.gz" "$RELEASE_ARCHIVE.tar.gz"

rm -rf "$RELEASE_ARCHIVE" && \
  tar -xvzf "$RELEASE_ARCHIVE.tar.gz" && \
  rm -f "$RELEASE_ARCHIVE.tar.gz" && \
  cd "$RELEASE_ARCHIVE" && \
  mkdir bin && \
  download_from_project "$PROJECT_URL/releases/download/v$RELEASE/happac" "bin/happac" && \
  chmod +x bin/* && \
  systemd_install

clean_downloads
