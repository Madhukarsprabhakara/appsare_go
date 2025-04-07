FROM golang

RUN mkdir -p /var/www/go_src
ADD ./src/ /var/www/go_src
