FROM ruby:2.5.1
RUN gem install rufus-scheduler
RUN mkdir /app && mkdir /data
COPY ./tools /app/tools
COPY ./bin /app/bin
WORKDIR /app
CMD ["/app/tools/crawler"]
