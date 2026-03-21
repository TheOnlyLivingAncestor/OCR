# Use an official Python runtime as a parent image
FROM python:3.8-slim

# Set the working directory in the container
WORKDIR /Worker

# Copy the local directory contents into the container at /Worker
COPY . /Worker

# Run Python script when the container launches
CMD ["python", "main.py"]