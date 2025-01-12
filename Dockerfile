FROM ayankousky/go-base:1.2025-01-12 as development

WORKDIR /srv
COPY . .

RUN adduser -s /bin/sh -D -u 1000 app && chown -R app:app /home/app
USER app

CMD ["air", "-c", ".air.toml"]