VERSION 0.7

FROM tochemey/docker-go:1.22.0-3.0.0

protogen:
    # copy the proto files to generate
    COPY --dir protos/ ./
    COPY buf.work.yaml buf.gen.yaml ./

    # generate the pbs
    RUN buf generate \
            --template buf.gen.yaml \
            --path protos/goaktwspb

    # save artifact to
    SAVE ARTIFACT gen/goaktwspb AS LOCAL goaktwspb
