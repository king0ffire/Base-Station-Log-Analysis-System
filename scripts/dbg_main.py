#负责解压文件并分析dbglog文件

import cProfile
from concurrent.futures import ThreadPoolExecutor, as_completed
import logging.handlers
import queue
from collections import Counter
import os
import tarfile
import csv
import glob
import sys
import logging

@profile
# mode 0 is single thread， mode 1 is multithread
def run(filelocation, mode=0):
    extracteddir = os.path.splitext(os.path.splitext(filelocation)[0])[0]
    fileuid=os.path.basename(extracteddir)
    if not os.path.exists(extracteddir):
        os.makedirs(extracteddir)
    with tarfile.open(filelocation, "r:gz") as tar:
        tar.extractall(path=extracteddir,filter='fully_trusted')

    queue_listener = configure_logger(extracteddir)
    queue_listener.start()

    from util import mapget
    from dbcount import constructdbgcsv,Parsefile,Parsefilelist_2

    def sqlinit(fileuid):
        global sql
        import sql
        sql.init()
        sql.createtable(fileuid)
    with ThreadPoolExecutor() as executor:
        executor.submit(sqlinit,fileuid)
        logger = logging.getLogger(__name__)
        logger.info("start logger")
        logger.info("Current Working Directory: %s", os.getcwd())
        logger.info("Current File Dirctory: %s", os.path.abspath("."))

        tracelocation = os.path.join(extracteddir, "logs", "trace.tgz")
        traceextraceteddir = os.path.splitext(tracelocation)[0]
        if not os.path.exists(traceextraceteddir):
            os.makedirs(traceextraceteddir)
        with tarfile.open(tracelocation, "r:gz") as tar:
            tar.extractall(path=traceextraceteddir,filter='fully_trusted')
        dbglogsdir = os.path.join(traceextraceteddir, "trace")
        csvfile_dbg = open(os.path.join(extracteddir, "dbg.csv"), "w", newline="",encoding="utf-8")
        csvwriter_dbg = csv.writer(csvfile_dbg)

        dbg_file_list = glob.glob(dbglogsdir + "/dbglog*")
        cache_path = os.path.join(extracteddir, "cache")
        if not os.path.exists(cache_path):
            os.makedirs(cache_path)

        fourEqualPattern = r"====[^\[]*"
        fiveDashPattern = r"-----[^-\[\n]+"
        pattern1 = r"[X2AP]:Sending UE CONTEXT RELEASE"
        pattern2 = r"Received HANDOVER REQUEST"

        #countmap = counter_FileListby2patterns(dbg_file_list, fourEqualPattern, fiveDashPattern)
        #db
        #单独的线程-1
        logger.info("start dbg")

        formatteditems, countlist=Parsefilelist_2(dbg_file_list, [fourEqualPattern, fiveDashPattern,pattern1,pattern2],mode)
        
        logger.info("dbg analisis finished, dbg file wont open again")


    #单独的线程- 2<-1
    #cursor=sql.conn.cursor()
        '''
    sqlsentence=f"insert DELAYED into dbgitems_{os.path.fileuid} values (null,%s,%s,%s,%s,%s,'{fileuid}')"
    sql.batch_insert(sqlsentence,formatteditems)
    sql.conn.commit()
    '''
    with open(os.path.join(extracteddir, "dbgitems.csv"), "w", newline="",encoding="utf-8") as file:
        csvwriter_database = csv.writer(file)
        csvwriter_database.writerows(formatteditems)
    sqlsentence=f"load data infile '{os.path.abspath(extracteddir)}/dbgitems.csv' into table dbgitems_{fileuid} fields terminated by ',' enclosed by '\"' lines terminated by '\\n' (time,errortype,device,info,event) set id=null, fileid='{fileuid}';"
    print(sqlsentence)
    sql.conn.cursor().execute(sqlsentence)
    sql.conn.commit()

    #cursor.executemany(sqlsentence,formatteditems)

    #单独的线程- 2<-1
    from category import get_tagfromcsv
    countmap=Counter([tup[4] for tup in formatteditems])
    #categories = get_category(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.xlsx"))
    tags=get_tagfromcsv(os.path.join(os.path.dirname(sys.argv[0]), r"dbg信令分类_唯一分类.csv")) # not including non-categorized

    dbgfileinfo=constructdbgcsv(countmap,tags)
    csvwriter_dbg.writerows(dbgfileinfo)
    csvfile_dbg.close()

    logger.info("dbg finished")

    listofpattern1 = countlist[0]
    listofpattern2 = countlist[1]
    with open(os.path.join(extracteddir, "accounting.csv"), "w", newline="",encoding="utf-8") as f:
        csvwriter_acc = csv.writer(f)
        accinfo=[]
        accinfo.append(
            [
                mapget(countmap, "rrc connection setup complete"),
                mapget(countmap, "rrc connection request"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "rrc connection reestablishement complete"),
                mapget(countmap, "rrc connection reestablishement request"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "initial context setup response"),
                mapget(countmap, "initial ue message"),
            ]
        )
        accinfo.append(
            [
                mapget(countmap, "handover notify") + listofpattern1,
                mapget(countmap, "handover request") + listofpattern2,
            ]
        )
        csvwriter_acc.writerows(accinfo)
    logger.info("accounting finished")
    sql.conn.commit()
    print("dbg analysis success")
    queue_listener.stop()


def configure_logger(location):
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.DEBUG)

    que = queue.Queue(-1)
    queue_handler = logging.handlers.QueueHandler(que)
    file_handler = logging.FileHandler(os.path.join(location, "python_dbg.log"), mode="w")
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

