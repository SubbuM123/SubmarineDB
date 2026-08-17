#!/usr/bin/env python3

import base64
import hashlib
import time
import uuid
from datetime import datetime
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import requests
import random
import numpy as np

from generate_files import generate_single_file

import os
import pandas as pd
# from ultralytics import YOLO
# import cv2

# model = YOLO("yolov8n.pt")


# -------------------------------
# Configuration
# -------------------------------

SERVER = ["http://localhost:7067", "http://localhost:7150", "http://localhost:7233"]

PUT_ENDPOINT = SERVER[0] + "/put_chunk"
GET_ENDPOINT = SERVER[0] + "/get_chunk"
DELETE_ENDPOINT = SERVER[0] + "/delete_chunk"

CHUNK_SIZE = 1024 * 1024  # 1 MB

# MANIFEST_DIR = Path("./manifests")
VIDEO_CHUNK_MAP = {}

WINDOW_SIZE = 16
WARMUP = 5
ITERATIONS = 30

# -------------------------------
# Helpers
# -------------------------------

def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def split_file(filename):
    """
    Split file into chunks.
    Returns:
        chunk index
        chunk bytes
    """

    with open(filename, "rb") as f:

        index = 0

        while True:

            chunk = f.read(CHUNK_SIZE)

            if not chunk:
                break

            yield index, chunk

            index += 1

def get_node():
    """
    For now just use first node.
    Later replace with node discovery.
    """

    # return SERVER_NODES[0]
    return random.choice(SERVER)


# -------------------------------
# DHT Communication
# -------------------------------

def upload_chunk(chunk_hash, data, parent_video, tags):

    now = datetime.now()

    metadata = [
        now.strftime("%H:%M:%S"),
        now.strftime("%Y-%m-%d")
    ]
    # metadata = tags

    value = {
        "data": base64.b64encode(data).decode("ascii"),
        "parent_video" : parent_video,
        "metadata": metadata
    }

    payload = {
        "key": chunk_hash,
        "value": value
    }

    url = (
        get_node() + "/put_chunk"
    )

    response = requests.put(
        url,
        json=payload
    )

    if response.status_code != 200:
        raise Exception(
            f"Chunk upload failed: {response.text}"
        )


def download_chunk(chunk_hash):
    node = get_node()
    response = requests.get(
        node + "/get_chunk",
        params={
            "key": chunk_hash
        }
    )

    if response.status_code != 200:
        raise Exception(
            f"Chunk download failed: {response.text}"
        )

    encoded_data = response.json()

    if not isinstance(encoded_data, str):
        raise Exception(
            f"Unexpected chunk payload: {encoded_data}"
        )

    return base64.b64decode(encoded_data)

def delete_chunk(chunk_hash):
    node = get_node()
    response = requests.delete(
        node + "/delete_chunk",
        params={"key": chunk_hash}
    )
    if response.status_code != 200:
        raise Exception(f"Chunk delete failed: {response.text}")
    return


# -------------------------------
# Upload Video
# -------------------------------

# def upload_video_slow(filename):

#     path = Path(filename)

#     if not path.exists():
#         raise FileNotFoundError(filename)


#     video_id = str(uuid.uuid4())
#     video_name = path.name
#     chunk_hashes = []


#     print("Uploading:", video_name)
#     print("Video ID:", video_id)


#     for index, chunk in split_file(filename):

#         chunk_hash = sha256(chunk)
#         print(
#             f"Uploading chunk {index}: {chunk_hash}"
#         )
#         upload_chunk(
#             chunk_hash,
#             chunk
#         )
#         chunk_hashes.append(chunk_hash)

#     VIDEO_CHUNK_MAP[video_name] = chunk_hashes
#     print("\nUpload complete")

def upload_video(filename, window_size=1):
    path = Path(filename)
    tags = []
    # tags = detect_objects(filename)
    if not path.exists():
        raise FileNotFoundError(filename)

    video_id = str(uuid.uuid4())
    video_name = path.name

    print("Uploading:", video_name)
    # print("Video ID:", video_id)

    # Reading + hashing is local/CPU-bound and fast — do this part
    # sequentially, it's not the bottleneck. Only the network upload
    # needs to be parallelized.
    chunks = list(split_file(filename))
    chunk_hashes = [None] * len(chunks)

    with ThreadPoolExecutor(max_workers=window_size) as executor:

        futures = {}

        for index, chunk in chunks:
            chunk_hash = sha256(chunk)
            chunk_hashes[index] = chunk_hash
            fut = executor.submit(upload_chunk, chunk_hash, chunk, filename, tags)
            futures[fut] = index

        for fut in futures:
            index = futures[fut]
            fut.result()
            # print(f"Uploaded chunk {index}/{len(chunks) - 1}")

    VIDEO_CHUNK_MAP[video_name] = chunk_hashes

    # print("\nUpload complete")

# this is the slow sequential version
# def download_video_slow(filename):

#     video_name = Path(filename).name

#     if video_name not in VIDEO_CHUNK_MAP:
#         raise KeyError(
#             f"No chunk mapping found for {video_name}"
#         )

#     output_dir = Path("output")
#     output_dir.mkdir(exist_ok=True)

#     output_path = output_dir / video_name

#     print("Downloading:", video_name)

#     with output_path.open("wb") as f:

#         for chunk_hash in VIDEO_CHUNK_MAP[video_name]:

#             # print(
#             #     f"Downloading chunk: {chunk_hash}"
#             # )

#             chunk_data = download_chunk(chunk_hash)
#             f.write(chunk_data)

#     print("\nDownload complete")
#     print("Saved to:", output_path)

# this version parallelizes the download
def download_video(filename, window_size = 1):

    video_name = Path(filename).name

    if video_name not in VIDEO_CHUNK_MAP:
        raise KeyError(f"No chunk mapping found for {video_name}")

    output_dir = Path("output")
    output_dir.mkdir(exist_ok=True)
    output_path = output_dir / video_name

    chunk_hashes = VIDEO_CHUNK_MAP[video_name]

    print("Downloading:", video_name)

    with output_path.open("wb") as f, ThreadPoolExecutor(max_workers=window_size) as executor:

        # Submitting all at once is fine — max_workers caps how many
        # actually run concurrently; the rest just sit queued.
        futures = [executor.submit(download_chunk, h) for h in chunk_hashes]

        for index, fut in enumerate(futures):
            data = fut.result()  # blocks only until *this* chunk is ready
            f.write(data)
            # print(f"Wrote chunk {index}/{len(futures) - 1}")

    # print("\nDownload complete")
    # print("Saved to:", output_path)

def delete_video(filename):

    video_name = Path(filename).name

    if video_name not in VIDEO_CHUNK_MAP:
        raise KeyError(f"No chunk mapping found for {video_name}")

    chunk_hashes = VIDEO_CHUNK_MAP[video_name]

    # print("Deleting:", video_name)

    with ThreadPoolExecutor(max_workers=WINDOW_SIZE) as executor:

        futures = [executor.submit(delete_chunk, h) for h in chunk_hashes]

        for fut in futures:
            fut.result()  # raises if a chunk failed to delete after retries

    del VIDEO_CHUNK_MAP[video_name]

    # print("\nDelete complete")

def clear_output():

    output_dir = Path("output")

    if not output_dir.exists():
        # print("Nothing to clear — output/ doesn't exist")
        return

    count = 0

    for p in output_dir.iterdir():
        if p.is_file():
            p.unlink()
            count += 1

    # print(f"Cleared {count} file(s) from output/")

def test_speed(iter = ITERATIONS, thread_count = WINDOW_SIZE):
    filename = "speed_input/test"

    sizes = [8, 16, 32, 64, 128]

    for i in range(WARMUP):
        upload_video("speed_input/8", window_size=WINDOW_SIZE)
        # delete_video(filename)

    data = []
    for i in range(iter):
        for size in sizes:
            generate_single_file(size)
            t = time.time()
            upload_video(filename, window_size=thread_count)
            t2 = time.time() - t

            data.append({"operation": "upload", "threads": thread_count, "size": size, "time": t2})

            t = time.time()
            download_video(filename, window_size=thread_count)
            t2 = time.time() - t
            data.append({"operation": "download", "threads": thread_count, "size": size, "time": t2})

            delete_video(filename)

        df = pd.DataFrame(data)
        df.to_csv(f"speed_data_{thread_count}.csv", index=False)


# def test_fast(size, iter = ITERATIONS):
#     filename = "speed_input/test"

#     for i in range(WARMUP):
#         generate_single_file(size)
#         upload_video("speed_input/8", window_size=WINDOW_SIZE)
#         # delete_video(filename)

#     put_times = []
#     get_times = []
#     for i in range(iter):
#         generate_single_file(size)
#         t = time.time()
#         upload_video(filename, window_size=WINDOW_SIZE)
#         t2 = time.time() - t
#         put_times.append(t2)

#         t = time.time()
#         download_video(filename, window_size=WINDOW_SIZE)
#         t2 = time.time() - t
#         get_times.append(t2)
#         # TODO  REMOVE
#         with open("put_data.txt", "a") as file:
#             file.write(f"{size}: {str(put_times[-1])}")
#         with open("get_data.txt", "a") as file:
#             file.write(f"{size}: {str(get_times[-1])}")

#         delete_video(filename)

#     np.save(f"data/put_data_{size}", np.array(put_times))
#     np.save(f"data/get_data_{size}", np.array(get_times))

#     print("Average upload time: ", sum(put_times)/iter)
#     print("Average download time: ", sum(get_times)/iter)

# def test_slow(size, iter = ITERATIONS):
#     filename = "speed_input/test"

#     for i in range(WARMUP):
#         generate_single_file(size)
#         upload_video("speed_input/8", window_size=WINDOW_SIZE)
#         # delete_video(filename)

#     put_times = []
#     get_times = []
#     for i in range(iter):
#         generate_single_file(size)
#         t = time.time()
#         upload_video(filename, window_size=1)
#         t2 = time.time() - t
#         put_times.append(t2)

#         t = time.time()
#         download_video(filename, window_size=1)
#         t2 = time.time() - t
#         get_times.append(t2)
#         # TODO  REMOVE
#         with open("put_data.txt", "a") as file:
#             file.write(f"{size}: {str(put_times[-1])}")
#         with open("get_data.txt", "a") as file:
#             file.write(f"{size}: {str(get_times[-1])}")

#         delete_video(filename)

#     np.save(f"data/put_data_{size}", np.array(put_times))
#     np.save(f"data/get_data_{size}", np.array(get_times))

#     print("Average upload time: ", sum(put_times)/iter)
#     print("Average download time: ", sum(get_times)/iter)



# -------------------------------
# CLI
# -------------------------------

def main():

    while True:

        command = input("> ").strip()

        if command == "exit":
            break


        parts = command.split(
            maxsplit=2
        )

        if parts[0] == "f":
            test_speed(int(parts[1]), int(parts[2]))
            # if len(parts) == 2:
            #     test_fast(parts[1])
            # else:
            #     test_fast(parts[1], int(parts[2]))
        # elif parts[0] == "s":
        #     if len(parts) == 2:
        #         test_slow(parts[1])
        #     else:
        #         test_slow(parts[1], int(parts[2]))
        
        elif parts[0] == "view":
            os.startfile(str(parts[1]))

        elif parts[0] == "clear":
            clear_output()

        else:

            print(
                "Commands:"
            )
            print(
                " put <file>"
            )
            print(
                " get <file>"
            )
            print(
                " view <file>"
            )
            print(
                " exit"
            )


if __name__ == "__main__":
    main()



# if parts[0] == "put_fast":

#             if len(parts) != 2:
#                 print(
#                     "Usage: put <file>"
#                 )
#                 continue

#             for i in range(WARMUP):
#                 upload_video(
#                     parts[1],
#                     window_size=WINDOW_SIZE
#                 )
#                 delete_video(parts[1])

#             times = []
#             for i in range(ITERATIONS):
#                 t = time.time()
#                 upload_video(
#                     parts[1],
#                     window_size=WINDOW_SIZE
#                 )
#                 t2 = time.time() - t

#                 times.append(t2)
#                 delete_video(parts[1])
#             print("Average upload time: " + str(sum(times)/ITERATIONS))

#         elif parts[0] == "get_fast":

#             if len(parts) != 2:
#                 print(
#                     "Usage: put <file>"
#                 )
#                 continue

#             for i in range(WARMUP):
#                 upload_video(
#                     parts[1],
#                     window_size=WINDOW_SIZE
#                 )
#                 delete_video(parts[1])

#             times = []
#             for i in range(ITERATIONS):
#                 t = time.time()
#                 download_video(
#                     parts[1],
#                     window_size=WINDOW_SIZE
#                 )
#                 t2 = time.time() - t

#                 times.append(t2)
#                 clear_output()
#             print("Average download time: ", times)

#         elif parts[0] == "put_slow":

#             if len(parts) != 2:
#                 print(
#                     "Usage: put <file>"
#                 )
#                 continue
#             times = []
#             for i in range(ITERATIONS):
#                 t = time.time()
#                 upload_video(
#                     parts[1]
#                 )
#                 t2 = time.time() - t

#                 times.append(t2)
#                 delete_video(parts[1])
#             print("Average slow upload time: ", times)

#         elif parts[0] == "get_slow":

#             if len(parts) != 2:
#                 print(
#                     "Usage: put <file>"
#                 )
#                 continue
#             times = []
#             for i in range(ITERATIONS):
#                 t = time.time()
#                 download_video(
#                     parts[1]
#                 )
#                 t2 = time.time() - t

#                 times.append(t2)
#                 clear_output()
#             print("Average slow download time: " + str(sum(times)/ITERATIONS))
