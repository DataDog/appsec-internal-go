FROM golang:1.21 as go-format

RUN apt update && apt install -y jq curl
COPY writer/ /home/
WORKDIR /home/

ARG version
ARG GITHUB_TOKEN

RUN if [ "${version}" = "latest" ]; then \
  curl \
    --header "Authorization: Bearer ${GITHUB_TOKEN}" \
    --header "X-Github-Api-Version: 2022-11-28" \
    -fSsL 'https://api.github.com/repos/DataDog/appsec-event-rules/releases/latest' \
  | jq -r '.tag_name' > .version; \
  else \
  echo "${version}" > .version; \
  fi

RUN curl \
    --header "Authorization: Bearer ${GITHUB_TOKEN}" \
    --header "X-Github-Api-Version: 2022-11-28" \
    -fSsL "https://raw.githubusercontent.com/DataDog/appsec-event-rules/$(cat .version)/build/recommended.json" \
    -o /home/rules.json
RUN go run writer.go $(cat .version) > embed.go

FROM scratch
COPY --from=go-format /home/embed.go embed.go
COPY --from=go-format /home/rules.json rules.json
