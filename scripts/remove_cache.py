import os
import shutil
import sys 

debug=True
def run(filelocation, mode=0):
    extracteddir = os.path.splitext(os.path.splitext(filelocation)[0])[0]
    cache_path = os.path.join(extracteddir, "cache")
    if not debug:
        os.remove(filelocation)
    shutil.rmtree(os.path.join(extracteddir, "logs"))
    shutil.rmtree(cache_path)

if __name__ == "__main__":
    run(sys.argv[1], mode=int(sys.argv[2]))
