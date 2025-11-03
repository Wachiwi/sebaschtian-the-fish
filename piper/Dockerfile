FROM python:3.12-slim

RUN apt-get update && \
    apt-get install --yes --no-install-recommends \
      build-essential cmake ninja-build git

WORKDIR /app

RUN pip install piper-tts
RUN python3 -m pip install piper-tts[http]
RUN python3 -m piper.download_voices de_DE-karlsson-low 

ENTRYPOINT ["python3"]
CMD ["-m", "piper.http_server", "-m", "de_DE-karlsson-low"]




