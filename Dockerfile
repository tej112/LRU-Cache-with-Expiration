# Use a minimal base image
FROM alpine:latest  

WORKDIR /root/

# Copy the pre-built binary file into the image
COPY main .

# Expose port 8080 to the outside world (if needed)
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
