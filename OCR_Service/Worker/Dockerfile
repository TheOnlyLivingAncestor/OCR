# Use an official Python runtime as a parent image
FROM python:3.12-slim-bookworm

RUN apt-get update && apt-get install -y wget unzip && rm -rf /var/lib/apt/lists/* && \
    mkdir -p /home/app/EasyOCR/model && \
    wget https://github.com/JaidedAI/EasyOCR/releases/download/v1.3/english_g2.zip -O /home/app/EasyOCR/english_g2.zip && \
    wget https://github.com/JaidedAI/EasyOCR/releases/download/pre-v1.1.6/craft_mlt_25k.zip -O /home/app/EasyOCR/craft_mlt_25k.zip && \
    unzip /home/app/EasyOCR/english_g2.zip -d /home/app/EasyOCR/model && \
    unzip /home/app/EasyOCR/craft_mlt_25k.zip -d /home/app/EasyOCR/model && \
    rm /home/app/EasyOCR/*.zip


RUN pip install --no-cache-dir -r requirements.txt    
# Set the working directory in the container
WORKDIR /Worker

# Copy the local directory contents into the container at /Worker
COPY . /Worker

# Run Python script when the container launches
CMD ["python", "main.py"]