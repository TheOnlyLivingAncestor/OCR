import cv2
import easyocr
import base64
import numpy as np
import json
import requests
import os
import logging
from rabbitmq_amqp_python_client import (
    Environment,
    Message,
    AddressHelper
)

reader = easyocr.Reader(['en'], download_enabled=False,
                        model_storage_directory="/home/app/EasyOCR/model",
                        gpu=False)


def get_env(name: str) -> str:
    value = os.getenv(name)
    if not value:
        raise RuntimeError(f"Missing required env var: {name}")
    return value


def download_image(presigned_url):
    response = requests.get(presigned_url)
    response.raise_for_status()
    image_bytes = np.frombuffer(response.content, np.uint8)
    image = cv2.imdecode(image_bytes, cv2.IMREAD_COLOR)
    return image


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)

    rabbitAddress = get_env("RABBITMQ_URL")
    publisherQueue = get_env("RABBITMQ_PUBLISHER_QUEUE")
    downloadUrl = get_env("DOWNLOAD_URL")
    uploadUrl = get_env("UPLOAD_URL")
    jobID = get_env("JOBID")

    logging.info(f"rabbitAddress: {rabbitAddress}")
    logging.info(f"publisherQueue: {publisherQueue}")
    logging.info(f"downloadUrl: {downloadUrl}")
    logging.info(f"uploadUrl: {uploadUrl}")
    logging.info(f"jobID: {jobID}")
    try:
        # Letöltjük a képet és elvégezzük rajta a szövegfelismerést
        image = download_image(downloadUrl)
        logging.info("Successfully downloaded image")
        results = reader.readtext(image,
                                  paragraph=True,
                                  slope_ths=0.4,
                                  width_ths=0.7)

        # Berajzoljuk a képen a a dobozokat
        for (boundary, _) in results:
            pts = np.array(boundary, dtype=np.int32)
            cv2.polylines(image, [pts], True, (0, 255, 0), 2)

        # Az annotált kép
        buf = cv2.imencode('.jpg', image)[1]
        annotated_b64 = base64.b64encode(buf).decode('utf-8')

        # detektált szövegek összegyűjtése
        results_json = []
        for (_, text) in results:
            results_json.append(text)

        ocr_result = {
            "jobID": jobID,
            "image": annotated_b64,
            "results": results_json
        }

        logging.info(f"OCR result: {ocr_result}")

        ocr_result_json = json.dumps(ocr_result).encode('utf-8')
        response = requests.put(uploadUrl,
                                data=ocr_result_json,
                                headers={'Content-Type': 'application/json'})

        if response.status_code == 200:
            logging.info("OCR result successfully uploaded.")
        else:
            errormsg = {response.status_code} + "-" + {response.text}
            logging.error(f"Error during upload: {errormsg}")

        # Publish to rabbitMQ
        environment = Environment(rabbitAddress)
        connection = environment.connection()
        connection.dial()

        management = connection.management()

        queue_address = AddressHelper.queue_address(publisherQueue)
        publisher = connection.publisher(queue_address)

        message_body = f"OCR recognition finished with ID:{jobID}"
        encoded_message_body = message_body.encode("utf-8")
        msg = Message(body=encoded_message_body)
        publisher.publish(msg)

        print("OCR result published in RabbitMQ")

        publisher.close()
        connection.close()
        environment.close()
    except Exception as e:
        logging.error("Something happened: {e}")
        raise RuntimeError(f"Failed to process task {e}")
