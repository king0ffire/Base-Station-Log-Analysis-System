import asyncio
from concurrent.futures import ThreadPoolExecutor, as_completed, ProcessPoolExecutor
import logging.handlers
import queue
import os
import csv
import glob
import gzip
import sys
import logging



# mode 0 is single thread， mode 1 is multithread
def run(filelocation, mode=0):
    filter1 = "s1ap.MME_UE_S1AP_ID"
    filter2 = "s1ap.ENB_UE_S1AP_ID"

    extracteddir = os.path.splitext(os.path.splitext(sys.argv[1])[0])[0]

    tracelocation = os.path.join(extracteddir, "logs", "trace.tgz")
    traceextraceteddir = os.path.splitext(tracelocation)[0]
    if not os.path.exists(traceextraceteddir):
        os.makedirs(traceextraceteddir)
        

    csvfile_id = open(os.path.join(extracteddir, "ids.csv",), "w", newline="",encoding="utf-8")
    csvwriter_id = csv.writer(csvfile_id)
    sctp_file_list = glob.glob(os.path.dirname(tracelocation) + "/sctp*")

    cache_path = os.path.join(extracteddir, "cache")
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)
    
    print(len(sctp_file_list))
    sys.stdout.flush()
    sctpanalysis(csvfile_id, csvwriter_id, sctp_file_list, cache_path, filter1,filter2,mode)





def uncaught_exception(exctype, value, tb):
    logger.error("Uncaught exception", exc_info=(exctype, value, tb))
    queue_listener.stop()
    exit(0)

def configure_logger(location:str, level:int=logging.DEBUG):
    logger = logging.getLogger()
    logger.setLevel(level)

    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(
        os.path.join(location, "sctp.log"), mode="w"
    )
    queue_handler.setLevel(level)
    file_handler.setLevel(level)
    file_handler.setFormatter(
        logging.Formatter(
            "%(asctime)s %(levelname)s %(threadName)s %(module)s.%(funcName)s:%(lineno)d %(message)s"
        )
    )

    queuelistener = logging.handlers.QueueListener(que, file_handler)
    logger.addHandler(queue_handler)
    return queuelistener



if __name__ == "__main__":
    
    extracteddir = os.path.splitext(os.path.splitext(sys.argv[1])[0])[0]
    fileuid=os.path.basename(extracteddir)
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)

    queue_listener = configure_logger(extracteddir)
    queue_listener.start()
    sys.excepthook = uncaught_exception
    logger = logging.getLogger(__name__)
    logger.info("start logger")
    logger.info("Current Working Directory: %s", os.getcwd())
    logger.info("Current File Dirctory: %s", os.path.abspath("."))
    from core.sctp import sctpanalysis
    
    run(sys.argv[1], mode=int(sys.argv[2]))
    

    logger.debug("python process ended")
    queue_listener.stop()