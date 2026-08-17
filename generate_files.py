import os

CHUNK_SIZE = 1024 * 1024

def generate_test_files(path = "speed_input/"):
    sizes = [8, 16, 32, 64, 128]

    for s in sizes:
        with open(path + str(s) , "wb") as f:
            for i in range(s):
                f.write(os.urandom(CHUNK_SIZE))

def generate_single_file(size, path = "speed_input/test"):
    # sizes = [8, 16, 32, 64, 128]

    with open(path, "wb") as f:
        for i in range(int(size)):
            f.write(os.urandom(CHUNK_SIZE))
    # return path

if __name__ == "__main__":
    generate_test_files()