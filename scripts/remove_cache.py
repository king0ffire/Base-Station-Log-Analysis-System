import os
import shutil
import sys 
import logging
debug=False
def removelocaltgz(cache_location, fileuid, mode=0):
    logger=logging.getLogger(__name__)
    try:
        location=os.path.join(cache_location, fileuid+".tar.gz")
        os.remove(location)
        logger.debug(f"remove {location} success")
    except Exception as e:
        logger.debug(f"remove local data failed:{e}")
        
        
def removelocaldirectory(cache_location, fileuid, mode=0):
    logger=logging.getLogger(__name__)
    try:
        location=os.path.join(cache_location, fileuid)
        shutil.rmtree(location)
        logger.debug(f"remove {location} success")
    except Exception as e:
        logger.debug(f"remove local data failed:{e}")        
                
        
if __name__ == "__main__":
    removelocaltgz(sys.argv[1], mode=int(sys.argv[2]))
