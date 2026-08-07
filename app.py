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

import os


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

def upload_chunk(chunk_hash, data, parent_video):

    now = datetime.now()

    metadata = [
        now.strftime("%H:%M:%S"),
        now.strftime("%Y-%m-%d")
    ]
    # TODO metadata = tag_chunk()

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

# -------------------------------
# Upload Video
# -------------------------------

# def upload_video(filename):

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

def upload_video(filename):
    path = Path(filename)

    if not path.exists():
        raise FileNotFoundError(filename)

    video_id = str(uuid.uuid4())
    video_name = path.name

    print("Uploading:", video_name)
    print("Video ID:", video_id)

    # Reading + hashing is local/CPU-bound and fast — do this part
    # sequentially, it's not the bottleneck. Only the network upload
    # needs to be parallelized.
    chunks = list(split_file(filename))
    chunk_hashes = [None] * len(chunks)

    with ThreadPoolExecutor(max_workers=WINDOW_SIZE) as executor:

        futures = {}

        for index, chunk in chunks:
            chunk_hash = sha256(chunk)
            chunk_hashes[index] = chunk_hash
            fut = executor.submit(upload_chunk, chunk_hash, chunk, filename)
            futures[fut] = index

        for fut in futures:
            index = futures[fut]
            fut.result()
            print(f"Uploaded chunk {index}/{len(chunks) - 1}")

    VIDEO_CHUNK_MAP[video_name] = chunk_hashes

    print("\nUpload complete")

# this is the slow sequential version
# def download_video(filename):

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
def download_video(filename):

    video_name = Path(filename).name

    if video_name not in VIDEO_CHUNK_MAP:
        raise KeyError(f"No chunk mapping found for {video_name}")

    output_dir = Path("output")
    output_dir.mkdir(exist_ok=True)
    output_path = output_dir / video_name

    chunk_hashes = VIDEO_CHUNK_MAP[video_name]

    print("Downloading:", video_name)

    with output_path.open("wb") as f, ThreadPoolExecutor(max_workers=WINDOW_SIZE) as executor:

        # Submitting all at once is fine — max_workers caps how many
        # actually run concurrently; the rest just sit queued.
        futures = [executor.submit(download_chunk, h) for h in chunk_hashes]

        for index, fut in enumerate(futures):
            data = fut.result()  # blocks only until *this* chunk is ready
            f.write(data)
            # print(f"Wrote chunk {index}/{len(futures) - 1}")

    print("\nDownload complete")
    print("Saved to:", output_path)

def clear_output():

    output_dir = Path("output")

    if not output_dir.exists():
        print("Nothing to clear — output/ doesn't exist")
        return

    count = 0

    for p in output_dir.iterdir():
        if p.is_file():
            p.unlink()
            count += 1

    print(f"Cleared {count} file(s) from output/")


# -------------------------------
# CLI
# -------------------------------

def main():

    while True:

        command = input("> ").strip()

        if command == "exit":
            break


        parts = command.split(
            maxsplit=1
        )


        if parts[0] == "put":

            if len(parts) != 2:
                print(
                    "Usage: put <file>"
                )
                continue
            
            t = time.time()
            upload_video(
                parts[1]
            )
            print("Upload took: " + str(time.time() - t))

        elif parts[0] == "get":

            if len(parts) != 2:
                print(
                    "Usage: get <file>"
                )
                continue
            t = time.time()
            download_video(
                parts[1]
            )
            print("Download took: " + str(time.time() - t))
        
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
