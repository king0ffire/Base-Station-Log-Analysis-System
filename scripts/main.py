import asyncio
from concurrent.futures import ThreadPoolExecutor, as_completed
import logging.handlers
import queue

import os
import tarfile
import csv
import glob
import gzip
import cProfile
import sys
import shutil
import logging


def mapget(map, key):
    if key in map:
        return map[key]
    else:
        return 0


debug = False

def get_tag(countmap, categories):
    tags = {}
    for key in countmap.keys():
        tags[key]=[]
    
    for key, categorylist in categories.items():
        for category in categorylist:
                tags[category].append(key)
                
    return tags
# mode 0 is single thread， mode 1 is multithread
def run(filelocation, mode=0):
    filter1 = "s1ap.MME_UE_S1AP_ID"
    filter2 = "s1ap.ENB_UE_S1AP_ID"
    basedir = os.path.dirname(filelocation)

    extracteddir = os.path.splitext(os.path.splitext(filelocation)[0])[0]
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation, "r:gz") as tar:
        tar.extractall(path=extracteddir)

    queue_listener = configure_logger(extracteddir)
    queue_listener.start()
    from ids_pyshark import pcapInfoToListBy2Filters, process_one_file_by2filters
    from dbcount import counter_FileListby2patterns, ParseFiles
    from category import get_category

    logger = logging.getLogger(__name__)
    logger.info("start logger")
    logger.info("Current Working Directory: %s", os.getcwd())
    logger.info("Current File Dirctory: %s", os.path.abspath("."))
    if not debug:
        os.remove(filelocation)
        logger.info("remove tar file")

    tracelocation = os.path.join(extracteddir, "logs", "trace.tgz")
    traceextraceteddir = os.path.splitext(tracelocation)[0]
    if not os.path.exists(traceextraceteddir):
        os.makedirs(traceextraceteddir)
    with tarfile.open(tracelocation, "r:gz") as tar:
        tar.extractall(path=traceextraceteddir)
    dbglogsdir = os.path.join(traceextraceteddir, "trace")
    csvfile_id = open(os.path.join(extracteddir, "ids.csv",), "w", newline="",encoding="utf-8")
    csvwriter_id = csv.writer(csvfile_id)
    csvfile_dbg = open(os.path.join(extracteddir, "dbg.csv"), "w", newline="",encoding="utf-8")
    csvwriter_dbg = csv.writer(csvfile_dbg)
    sctp_file_list = glob.glob(os.path.dirname(tracelocation) + "/sctp*")

    dbg_file_list = glob.glob(dbglogsdir + "/dbglog*")
    cache_path = os.path.join(extracteddir, "cache")
    if not os.path.exists(cache_path):
        os.makedirs(cache_path)

    logger.info("start dbg")
    fourEqualPattern = r"====[^\[]*"
    fiveDashPattern = r"-{5,}[^-\[\n]*"
    pattern1 = r"[X2AP]:Sending UE CONTEXT RELEASE"
    pattern2 = r"Received HANDOVER REQUEST"
    csvwriter_dbg.writerow(["Event Name", "Counts", "Tags"])
    countmap = counter_FileListby2patterns(
        dbg_file_list, fourEqualPattern, fiveDashPattern
    )

    categories = get_category(os.path.join(os.path.dirname(sys.argv[0]), "dbg信令分类.xlsx"))
    tags=get_tag(countmap, categories)

    eventsum = 0
    other = 0
    for key, value in countmap.items():
        eventsum += value
        if len(tags[key]) == 0:
            tags[key].append("未分类")
            other+=value
        csvwriter_dbg.writerow([key, value, tags[key]])

    csvfile_dbg.close()
    logger.info("dbg finished")

    categoriescount = {}
    listofpattern1 = len(ParseFiles(dbg_file_list, pattern1))
    listofpattern2 = len(ParseFiles(dbg_file_list, pattern2))
    with open(os.path.join(extracteddir, "accounting.csv"), "w", newline="",encoding="utf-8") as f:
        csvwriter_acc = csv.writer(f)

        csvwriter_acc.writerow(
            [
                mapget(countmap, "rrc connection setup complete"),
                mapget(countmap, "rrc connection request"),
            ]
        )
        csvwriter_acc.writerow(
            [
                mapget(countmap, "rrc connection reestablishement complete"),
                mapget(countmap, "rrc connection reestablishement request"),
            ]
        )
        csvwriter_acc.writerow(
            [
                mapget(countmap, "initial context setup response"),
                mapget(countmap, "initial ue message"),
            ]
        )
        csvwriter_acc.writerow(
            [
                mapget(countmap, "handover notify") + listofpattern1,
                mapget(countmap, "handover request") + listofpattern2,
            ]
        )

        for categoryname, eventlist in categories.items():
            categoriescount[categoryname] = 0
            for event in eventlist:
                categoriescount[categoryname] += mapget(countmap, event)

        csvwriter_acc.writerow(["事件总数", eventsum])
        csvwriter_acc.writerow(["UE接入", categoriescount["UE接入"]])
        csvwriter_acc.writerow(["S1切换入", categoriescount["S1切换入"]])
        csvwriter_acc.writerow(["S1切换出", categoriescount["S1切换出"]])
        csvwriter_acc.writerow(["未分类", other])

    logger.info("accounting finished")

    print(len(sctp_file_list))
    sys.stdout.flush()
    logger.info("sctp started")
    csvwriter_id.writerow(
        [
            "Filename",
            "Pkt Num",
            "Time",
            "Source IP",
            "Destination IP",
            "Protocol",
            "Summary Info",
            "MME-ID",
            "ENB-ID",
        ]
    )
    for i, filename in enumerate(sctp_file_list):
        if filename[-3:] == ".gz":
            with gzip.open(filename) as f:
                with open(
                    os.path.join(
                        cache_path, os.path.splitext(os.path.basename(filename))[0]
                    ),
                    "wb",
                ) as f2:
                    f2.write(f.read())
                    sctp_file_list[i] = f2.name
    if mode == 0:
        for filename in sctp_file_list:
            # csvwriter_id.writerow([os.path.basename(filename),'','','','','','',''])
            process_one_file_by2filters(csvwriter_id, filename, filter1, filter2)
            csvfile_id.flush()
            print("sctp_finished_one")
            sys.stdout.flush()

    elif mode == 1:
        with ThreadPoolExecutor(max_workers=10) as executor:
            fs = [
                executor.submit(
                    pcapInfoToListBy2Filters,
                    filename,
                    filter1,
                    filter2,
                    asyncio.new_event_loop(),
                )
                for filename in sctp_file_list
            ]
            for future in as_completed(fs):
                csvwriter_id.writerows(future.result())
                csvfile_id.flush()
                print("sctp_finished_one")
                sys.stdout.flush()
        print("multithread success")
    logger.info("sctp finished")
    shutil.rmtree(os.path.join(extracteddir, "logs"))
    shutil.rmtree(cache_path)
    logger.info("remove cache")
    queue_listener.stop()


def configure_logger(location):
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.DEBUG)

    if debug:
        logging.basicConfig(
            level=logging.DEBUG, format="%(asctime)s %(threadName)s %(message)s"
        )
    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(os.path.join(location, "python.log"), mode="w")
    queue_handler.setLevel(logging.INFO)
    file_handler.setLevel(logging.INFO)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s %(threadName)s %(message)s")
    )

    queuelistener = logging.handlers.QueueListener(que, file_handler)
    logger.addHandler(queue_handler)
    return queuelistener


if __name__ == "__main__":

    run(sys.argv[1], mode=int(sys.argv[2]))
